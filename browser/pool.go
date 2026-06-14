// Package browser drives a real headless Chrome through the DevTools Protocol so
// JavaScript-built pages are captured as they actually render. kage always goes
// through here: navigate, let the page settle, then serialise the final DOM —
// the same markup a human would have seen — which the rest of the pipeline then
// strips of scripts and localises.
package browser

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

// Options configure a Pool.
type Options struct {
	Headless      bool          // run Chrome without a window
	Workers       int           // max concurrent pages
	Settle        time.Duration // network-idle quiet period after load
	RenderTimeout time.Duration // hard cap per page render
	Scroll        bool          // auto-scroll to trigger lazy-loaded media
	ChromeBin     string        // explicit binary; empty = autodetect
	ControlURL    string        // attach to an existing Chrome instead of launching
}

// DefaultOptions returns the baseline render settings.
func DefaultOptions() Options {
	return Options{
		Headless:      true,
		Workers:       4,
		Settle:        1500 * time.Millisecond,
		RenderTimeout: 30 * time.Second,
	}
}

// Pool owns one Chrome process shared across a run and bounds the number of
// pages open at once.
type Pool struct {
	opts Options
	sem  chan struct{}

	mu      sync.Mutex
	browser *rod.Browser
	closed  bool
}

// New creates a Pool. Chrome is launched lazily on the first Render.
func New(opts Options) *Pool {
	if opts.Workers < 1 {
		opts.Workers = 1
	}
	return &Pool{opts: opts, sem: make(chan struct{}, opts.Workers)}
}

// RenderResult is the outcome of rendering one page.
type RenderResult struct {
	HTML     string // the serialised final DOM
	FinalURL string // URL after any client-side redirects
	Title    string
}

// Render navigates to rawURL, lets it settle, and returns the final rendered
// HTML. It acquires a page slot from the pool and releases it when done.
func (p *Pool) Render(ctx context.Context, rawURL string) (RenderResult, error) {
	select {
	case p.sem <- struct{}{}:
		defer func() { <-p.sem }()
	case <-ctx.Done():
		return RenderResult{}, ctx.Err()
	}

	b, err := p.getBrowser()
	if err != nil {
		return RenderResult{}, err
	}

	page, err := stealth.Page(b)
	if err != nil {
		return RenderResult{}, fmt.Errorf("new page: %w", err)
	}
	defer func() { _ = page.Close() }()

	page = page.Context(ctx).Timeout(p.opts.RenderTimeout)

	if err := page.Navigate(rawURL); err != nil {
		return RenderResult{}, fmt.Errorf("navigate %s: %w", rawURL, err)
	}
	if err := page.WaitLoad(); err != nil {
		return RenderResult{}, fmt.Errorf("wait load %s: %w", rawURL, err)
	}
	settle(page, p.opts.Settle)
	if p.opts.Scroll {
		autoScroll(page)
		settle(page, p.opts.Settle)
	}

	html, err := page.HTML()
	if err != nil {
		return RenderResult{}, fmt.Errorf("serialise %s: %w", rawURL, err)
	}

	res := RenderResult{HTML: html, FinalURL: rawURL}
	if info, err := page.Info(); err == nil && info != nil {
		res.FinalURL = info.URL
		res.Title = info.Title
	}
	return res, nil
}

// getBrowser lazily connects to or launches Chrome.
func (p *Pool) getBrowser() (*rod.Browser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, fmt.Errorf("pool is closed")
	}
	if p.browser != nil {
		return p.browser, nil
	}

	controlURL := p.opts.ControlURL
	if controlURL == "" {
		l := launcher.New().
			Headless(p.opts.Headless).
			Set("disable-blink-features", "AutomationControlled").
			Set("disable-dev-shm-usage", "").
			Set("no-sandbox", "").
			Set("disable-gpu", "")
		if bin := p.chromeBin(); bin != "" {
			l = l.Bin(bin)
		}
		u, err := l.Launch()
		if err != nil {
			return nil, fmt.Errorf("launch Chrome: %w", err)
		}
		controlURL = u
	}

	b := rod.New().ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		return nil, fmt.Errorf("connect Chrome: %w", err)
	}
	p.browser = b
	return b, nil
}

// Close shuts down the managed Chrome process.
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	if p.browser == nil {
		return nil
	}
	err := p.browser.Close()
	p.browser = nil
	return err
}

// LookChrome reports the path of a usable Chrome/Chromium binary and whether one
// was found, checking KAGE_CHROME, CHROME_BIN, rod's own lookup, and the common
// system install locations. Tests use it to skip when no browser is present.
func LookChrome() (string, bool) {
	for _, env := range []string{"KAGE_CHROME", "CHROME_BIN"} {
		if v := os.Getenv(env); v != "" {
			return v, true
		}
	}
	if bin, ok := launcher.LookPath(); ok {
		return bin, true
	}
	for _, c := range systemChromeCandidates() {
		if _, err := os.Stat(c); err == nil {
			return c, true
		}
	}
	return "", false
}

// chromeBin returns an explicit Chrome path from options or the environment, or
// "" to let the launcher find/download one.
func (p *Pool) chromeBin() string {
	if p.opts.ChromeBin != "" {
		return p.opts.ChromeBin
	}
	for _, env := range []string{"KAGE_CHROME", "CHROME_BIN"} {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	for _, c := range systemChromeCandidates() {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func systemChromeCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "windows":
		return []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
	default:
		return []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
		}
	}
}

// settle waits for the network to go quiet for d, recovering from any rod
// panic and capping the wait so a chatty page can never hang the worker.
func settle(page *rod.Page, d time.Duration) {
	if d <= 0 {
		return
	}
	defer func() { _ = recover() }()
	done := make(chan struct{})
	go func() {
		defer func() { _ = recover(); close(done) }()
		wait := page.WaitRequestIdle(d, nil, nil, []proto.NetworkResourceType{})
		wait()
	}()
	select {
	case <-done:
	case <-time.After(d + 5*time.Second):
	}
}

// autoScroll scrolls to the bottom in steps to trigger lazy-loaded images.
func autoScroll(page *rod.Page) {
	defer func() { _ = recover() }()
	_, _ = page.Eval(`() => new Promise((resolve) => {
		let total = 0;
		const step = 800;
		const timer = setInterval(() => {
			window.scrollBy(0, step);
			total += step;
			if (total >= document.body.scrollHeight) {
				clearInterval(timer);
				window.scrollTo(0, 0);
				resolve(true);
			}
		}, 100);
	})`)
}
