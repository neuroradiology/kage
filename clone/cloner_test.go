package clone

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/kage/browser"
	"github.com/tamnd/kage/urlx"
)

// testSite is a tiny two-page site with a stylesheet, an image, an inline
// script, an onclick handler, and a javascript: link, so a full clone exercises
// rendering, asset localisation, and JavaScript stripping at once.
func testSite(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head>
<link rel="stylesheet" href="/site.css">
<script src="/app.js"></script>
</head><body>
<h1>Home</h1>
<img src="/logo.png" alt="logo">
<a href="/about">About</a>
<a href="javascript:void(0)" onclick="boom()">Danger</a>
<script>console.log("inline")</script>
</body></html>`))
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head>
<link rel="stylesheet" href="/site.css">
</head><body><h1>About</h1><a href="/">Home</a></body></html>`))
	})
	mux.HandleFunc("/site.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte(`body{background:url("/bg.png")} h1{color:red}`))
	})
	mux.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = w.Write([]byte(`document.body.dataset.ran = "1";`))
	})
	mux.HandleFunc("/logo.png", servePNG)
	mux.HandleFunc("/bg.png", servePNG)
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
	})
	return httptest.NewServer(mux)
}

// a 1x1 transparent PNG.
var pngBytes = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func servePNG(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(pngBytes)
}

func TestCloneEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("clone end-to-end drives Chrome; skipped under -short")
	}
	if _, ok := browser.LookChrome(); !ok {
		t.Skip("no Chrome/Chromium found; skipping clone end-to-end")
	}

	srv := testSite(t)
	defer srv.Close()

	seed, err := urlx.ParseSeed(srv.URL)
	if err != nil {
		t.Fatalf("parse seed: %v", err)
	}

	out := t.TempDir()
	cfg := DefaultConfig()
	cfg.OutDir = out
	cfg.Settle = 300 * time.Millisecond
	cfg.RenderTimeout = 20 * time.Second
	cfg.Timeout = 10 * time.Second

	c := New(seed, cfg, t.Logf)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	res, err := c.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Pages < 2 {
		t.Fatalf("expected at least 2 pages, got %d", res.Pages)
	}

	root := res.OutDir
	indexPath := filepath.Join(root, "index.html")
	body := readFile(t, indexPath)

	// JavaScript is gone: no <script>, no onclick, no javascript: URL.
	if strings.Contains(strings.ToLower(body), "<script") {
		t.Error("index.html still contains a <script> tag")
	}
	if strings.Contains(strings.ToLower(body), "onclick") {
		t.Error("index.html still contains an onclick handler")
	}
	if strings.Contains(strings.ToLower(body), "javascript:") {
		t.Error("index.html still contains a javascript: URL")
	}

	// Layout is preserved: the stylesheet link survives and points local.
	if !strings.Contains(body, "stylesheet") {
		t.Error("index.html lost its stylesheet link")
	}
	if strings.Contains(body, srv.URL+"/site.css") {
		t.Error("stylesheet still points at the live origin")
	}

	// The about page and the localised assets exist on disk.
	if !fileExists(filepath.Join(root, "about", "index.html")) {
		t.Error("about page was not written")
	}
	assetDir := filepath.Join(root, cfg.Reserved)
	if !anyFileUnder(t, assetDir, "site.css") {
		t.Error("site.css was not downloaded")
	}
	if !anyFileUnder(t, assetDir, "logo.png") {
		t.Error("logo.png was not downloaded")
	}

	// The localised CSS had its url() rewritten away from the origin.
	css := readAnyFile(t, assetDir, "site.css")
	if strings.Contains(css, srv.URL) {
		t.Error("site.css still references the live origin in url()")
	}
}

func TestCloneResumeSkipsVisited(t *testing.T) {
	if testing.Short() {
		t.Skip("resume test drives Chrome; skipped under -short")
	}
	if _, ok := browser.LookChrome(); !ok {
		t.Skip("no Chrome/Chromium found; skipping resume test")
	}

	srv := testSite(t)
	defer srv.Close()
	seed, _ := urlx.ParseSeed(srv.URL)

	out := t.TempDir()
	cfg := DefaultConfig()
	cfg.OutDir = out
	cfg.Settle = 300 * time.Millisecond

	c1 := New(seed, cfg, t.Logf)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if _, err := c1.Run(ctx); err != nil {
		t.Fatalf("first run: %v", err)
	}

	// Second run with resume on should find the state and re-render nothing new.
	c2 := New(seed, cfg, t.Logf)
	res2, err := c2.Run(ctx)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if res2.Pages != 0 {
		t.Fatalf("resume should skip all visited pages, but rendered %d", res2.Pages)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func anyFileUnder(t *testing.T, dir, name string) bool {
	t.Helper()
	found := false
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, name) {
			found = true
		}
		return nil
	})
	return found
}

func readAnyFile(t *testing.T, dir, name string) string {
	t.Helper()
	var out string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, name) {
			b, _ := os.ReadFile(p)
			out = string(b)
		}
		return nil
	})
	return out
}
