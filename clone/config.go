// Package clone is kage's engine: it ties the Chrome pool, the JavaScript
// stripper, the asset localiser, and the URL↔path mapper into one resumable,
// polite crawl that turns a live site into a browsable offline folder.
package clone

import (
	"path/filepath"
	"time"

	"github.com/tamnd/kage/urlx"
)

// Config is the full set of knobs for a clone run. DefaultConfig fills the
// baseline; the CLI overlays flags on top.
type Config struct {
	OutDir   string // output root; the mirror lands in <OutDir>/<host>/
	Reserved string // reserved dir name for assets and state (default "_kage")

	Workers       int // page render workers
	AssetWorkers  int // HTTP asset download workers
	BrowserPages  int // Chrome page-pool size
	MaxPages      int // stop after N pages (0 = unlimited)
	MaxDepth      int // BFS/DFS depth cap (0 = unlimited)
	Traversal     string
	MaxAssetBytes int64

	Timeout       time.Duration // per HTTP request
	Settle        time.Duration // network-idle quiet period
	RenderTimeout time.Duration // hard cap per page render
	Scroll        bool

	UserAgent         string
	IncludeSubdomains bool
	ScopePrefix       string
	ExcludePaths      []string

	RespectRobots bool
	FollowSitemap bool
	Headless      bool
	KeepNoscript  bool
	ChromeBin     string
	ControlURL    string

	Resume bool
	Force  bool
}

// DefaultUserAgent is a current desktop Chrome UA, used by the asset fetcher and
// the robots fetch so a site treats kage like the browser it drives.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// DefaultConfig returns the baseline configuration.
func DefaultConfig() Config {
	return Config{
		OutDir:        "kage-out",
		Reserved:      urlx.DefaultReserved,
		Workers:       4,
		AssetWorkers:  8,
		BrowserPages:  4,
		MaxAssetBytes: 25 << 20,
		Traversal:     "bfs",
		Timeout:       30 * time.Second,
		Settle:        1500 * time.Millisecond,
		RenderTimeout: 30 * time.Second,
		UserAgent:     DefaultUserAgent,
		RespectRobots: true,
		FollowSitemap: true,
		Headless:      true,
		Resume:        true,
	}
}

// HostDir returns the mirror root for a seed host: <OutDir>/<host>.
func (c Config) HostDir(host string) string {
	return filepath.Join(c.OutDir, host)
}

// scope builds the urlx scope config from the run config.
func (c Config) scope() urlx.ScopeConfig {
	return urlx.ScopeConfig{
		IncludeSubdomains: c.IncludeSubdomains,
		ScopePrefix:       c.ScopePrefix,
		ExcludePaths:      c.ExcludePaths,
	}
}
