package clone

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tamnd/kage/asset"
	"github.com/tamnd/kage/browser"
	"github.com/tamnd/kage/robots"
	"github.com/tamnd/kage/sanitize"
	"github.com/tamnd/kage/urlx"
	"golang.org/x/net/html"
)

// Logf is an optional sink for human-readable progress lines.
type Logf func(format string, args ...any)

// Cloner runs one clone. Build it with New, then call Run.
type Cloner struct {
	cfg      Config
	seed     *url.URL
	seedHost string
	outRoot  string
	statePth string

	pool   *browser.Pool
	dl     *asset.Downloader
	httpC  *http.Client
	robots *robots.Matcher
	front  *frontier
	stats  stats
	logf   Logf

	mu         sync.Mutex
	seenAssets map[string]bool
	enqueued   int // pages offered to the queue
	wg         sync.WaitGroup
	pageJobs   chan pageItem
	assetJobs  chan assetItem
}

type pageItem struct {
	u     *url.URL
	depth int
}

type assetItem struct {
	u       *url.URL
	referer string
}

// New builds a Cloner for seed under cfg. It does not touch the network until
// Run is called.
func New(seed *url.URL, cfg Config, logf Logf) *Cloner {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	host := seed.Hostname()
	outRoot := cfg.HostDir(host)
	return &Cloner{
		cfg:        cfg,
		seed:       seed,
		seedHost:   host,
		outRoot:    outRoot,
		statePth:   filepath.Join(outRoot, cfg.Reserved, "state.json"),
		dl:         asset.NewDownloader(cfg.UserAgent, cfg.Timeout, cfg.MaxAssetBytes),
		httpC:      &http.Client{Timeout: cfg.Timeout},
		robots:     robots.AllowAll(),
		front:      newFrontier(),
		logf:       logf,
		seenAssets: map[string]bool{},
		pageJobs:   make(chan pageItem),
		assetJobs:  make(chan assetItem),
	}
}

// Snapshot returns the current progress, for a CLI ticker.
func (c *Cloner) Snapshot() Progress { return c.stats.snapshot() }

// Run executes the clone until the frontier drains, MaxPages is hit, or ctx is
// cancelled (which flushes the resume state). It returns the final Result.
func (c *Cloner) Run(ctx context.Context) (Result, error) {
	if c.cfg.Force {
		_ = os.RemoveAll(c.outRoot)
	}
	if c.cfg.Resume {
		if err := c.front.load(c.statePth); err != nil {
			c.logf("resume: could not load state: %v", err)
		} else if n := c.front.visitedCount(); n > 0 {
			c.logf("resume: %d pages already done", n)
		}
	}

	c.pool = browser.New(browser.Options{
		Headless:      c.cfg.Headless,
		Workers:       c.cfg.BrowserPages,
		Settle:        c.cfg.Settle,
		RenderTimeout: c.cfg.RenderTimeout,
		Scroll:        c.cfg.Scroll,
		ChromeBin:     c.cfg.ChromeBin,
		ControlURL:    c.cfg.ControlURL,
	})
	defer func() { _ = c.pool.Close() }()

	c.loadRobots(ctx)

	// Start workers.
	var workers sync.WaitGroup
	for range max1(c.cfg.Workers) {
		workers.Go(func() {
			for j := range c.pageJobs {
				c.processPage(ctx, j)
				c.wg.Done()
			}
		})
	}
	for range max1(c.cfg.AssetWorkers) {
		workers.Go(func() {
			for j := range c.assetJobs {
				c.processAsset(ctx, j)
				c.wg.Done()
			}
		})
	}

	// Seed.
	c.enqueuePage(ctx, c.seed, 0)
	if c.cfg.FollowSitemap {
		c.seedSitemaps(ctx)
	}

	// Close the job channels once every outstanding item is processed.
	go func() {
		c.wg.Wait()
		close(c.pageJobs)
		close(c.assetJobs)
	}()
	workers.Wait()

	if err := c.front.save(c.statePth); err != nil {
		c.logf("could not save resume state: %v", err)
	}

	res := Result{Progress: c.stats.snapshot(), OutDir: c.outRoot}
	if ctx.Err() != nil {
		return res, ctx.Err()
	}
	return res, nil
}

// loadRobots fetches and parses robots.txt (unless disabled) and seeds the
// sitemap list it advertises.
func (c *Cloner) loadRobots(ctx context.Context) {
	if !c.cfg.RespectRobots {
		return
	}
	robotsURL := c.seed.Scheme + "://" + c.seed.Host + "/robots.txt"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	resp, err := c.httpC.Do(req)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return
	}
	c.robots = robots.Parse(string(data), "kage")
}

// seedSitemaps adds in-scope sitemap URLs (from robots and the default path) to
// the frontier.
func (c *Cloner) seedSitemaps(ctx context.Context) {
	seeds := append([]string{}, c.robots.Sitemaps...)
	seeds = append(seeds, c.seed.Scheme+"://"+c.seed.Host+"/sitemap.xml")
	locs := collectSitemaps(ctx, c.httpC, c.cfg.UserAgent, seeds)
	added := 0
	for _, loc := range locs {
		u, err := urlx.Normalize(c.seed, loc)
		if err != nil {
			continue
		}
		if urlx.InScope(c.seed, u, c.cfg.scope()) && urlx.LikelyPage(u) {
			if c.enqueuePage(ctx, u, 1) {
				added++
			}
		}
	}
	if added > 0 {
		c.logf("sitemap: seeded %d URLs", added)
	}
}

// processPage renders one page, rewrites its references to local paths, strips
// every script, and writes the result.
func (c *Cloner) processPage(ctx context.Context, j pageItem) {
	if ctx.Err() != nil {
		return
	}
	key := urlx.Key(j.u)
	if c.cfg.RespectRobots && !c.robots.Allowed(j.u.Path) {
		c.stats.skipped.Add(1)
		return
	}

	res, err := c.pool.Render(ctx, j.u.String())
	if err != nil {
		c.stats.pageErrors.Add(1)
		c.logf("page error %s: %v", j.u, err)
		return
	}

	root, err := html.Parse(strings.NewReader(res.HTML))
	if err != nil {
		c.stats.pageErrors.Add(1)
		c.logf("parse error %s: %v", j.u, err)
		return
	}

	localFile := urlx.LocalPath(c.seedHost, j.u, urlx.Page, c.cfg.Reserved)
	fileDir := urlx.Dir(localFile)

	sink := func(u *url.URL, kind urlx.Kind) string {
		switch kind {
		case urlx.Page:
			if urlx.InScope(c.seed, u, c.cfg.scope()) {
				c.enqueuePage(ctx, u, j.depth+1)
				local := urlx.LocalPath(c.seedHost, u, urlx.Page, c.cfg.Reserved)
				return urlx.Rel(fileDir, local)
			}
			return u.String() // external page link stays on the live web
		default: // Asset
			c.enqueueAsset(ctx, u, j.u.String())
			local := urlx.LocalPath(c.seedHost, u, urlx.Asset, c.cfg.Reserved)
			return urlx.Rel(fileDir, local)
		}
	}

	asset.RewriteHTML(root, j.u, sink)
	sanitize.CleanTree(root, sanitize.Options{
		KeepNoscript: c.cfg.KeepNoscript,
		Banner:       "cloned by kage from " + j.u.String(),
	})

	var buf strings.Builder
	if err := html.Render(&buf, root); err != nil {
		c.stats.pageErrors.Add(1)
		return
	}
	if err := c.writeFile(localFile, []byte(buf.String())); err != nil {
		c.stats.pageErrors.Add(1)
		c.logf("write error %s: %v", localFile, err)
		return
	}
	c.front.markVisited(key)
	c.stats.pages.Add(1)
}

// processAsset downloads one asset, rewriting CSS references on the way, and
// writes it to its deterministic local path.
func (c *Cloner) processAsset(ctx context.Context, j assetItem) {
	if ctx.Err() != nil {
		return
	}
	res, err := c.dl.Get(ctx, j.u, j.referer)
	if err != nil {
		c.stats.assetErrors.Add(1)
		c.logf("asset error %s: %v", j.u, err)
		return
	}

	localFile := urlx.LocalPath(c.seedHost, j.u, urlx.Asset, c.cfg.Reserved)
	body := res.Body
	if res.IsCSS {
		fileDir := urlx.Dir(localFile)
		cssSink := func(u *url.URL, _ urlx.Kind) string {
			c.enqueueAsset(ctx, u, j.u.String())
			local := urlx.LocalPath(c.seedHost, u, urlx.Asset, c.cfg.Reserved)
			return urlx.Rel(fileDir, local)
		}
		body = asset.RewriteCSS(body, j.u, cssSink)
	}
	if err := c.writeFile(localFile, body); err != nil {
		c.stats.assetErrors.Add(1)
		c.logf("write error %s: %v", localFile, err)
		return
	}
	c.stats.assets.Add(1)
}

// enqueuePage offers a page URL to the frontier, honouring the visited set, the
// depth cap, and the page budget. It reports whether the page was newly queued.
func (c *Cloner) enqueuePage(ctx context.Context, u *url.URL, depth int) bool {
	if c.cfg.MaxDepth > 0 && depth > c.cfg.MaxDepth {
		return false
	}
	key := urlx.Key(u)
	if c.front.isVisited(key) {
		return false
	}
	if !c.front.offer(key) {
		return false
	}
	c.mu.Lock()
	if c.cfg.MaxPages > 0 && c.enqueued >= c.cfg.MaxPages {
		c.mu.Unlock()
		return false
	}
	c.enqueued++
	c.mu.Unlock()

	c.wg.Add(1)
	go func() {
		select {
		case c.pageJobs <- pageItem{u: u, depth: depth}:
		case <-ctx.Done():
			c.wg.Done()
		}
	}()
	return true
}

// enqueueAsset offers an asset URL for download, deduping by canonical URL.
func (c *Cloner) enqueueAsset(ctx context.Context, u *url.URL, referer string) {
	key := urlx.Key(u)
	c.mu.Lock()
	if c.seenAssets[key] {
		c.mu.Unlock()
		return
	}
	c.seenAssets[key] = true
	c.mu.Unlock()

	c.wg.Add(1)
	go func() {
		select {
		case c.assetJobs <- assetItem{u: u, referer: referer}:
		case <-ctx.Done():
			c.wg.Done()
		}
	}()
}

// writeFile writes data to a slash path relative to the mirror root, creating
// parent directories. The path is cleaned so it can never escape the root.
func (c *Cloner) writeFile(relSlash string, data []byte) error {
	rel := filepath.FromSlash(relSlash)
	full := filepath.Join(c.outRoot, rel)
	if !strings.HasPrefix(full, filepath.Clean(c.outRoot)+string(os.PathSeparator)) && full != c.outRoot {
		return fmt.Errorf("refusing to write outside mirror root: %s", relSlash)
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0o644)
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
