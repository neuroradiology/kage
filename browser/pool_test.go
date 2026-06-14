package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLookChromeReadsEnv(t *testing.T) {
	t.Setenv("KAGE_CHROME", "/custom/chrome")
	bin, ok := LookChrome()
	if !ok || bin != "/custom/chrome" {
		t.Fatalf("LookChrome() = %q, %v; want /custom/chrome, true", bin, ok)
	}
}

func TestRenderCapturesFinalDOM(t *testing.T) {
	if testing.Short() {
		t.Skip("render test drives Chrome; skipped under -short")
	}
	if _, ok := LookChrome(); !ok {
		t.Skip("no Chrome/Chromium found; skipping render test")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// A page whose visible content is built by JavaScript: only a real
		// browser render captures the injected node.
		_, _ = w.Write([]byte(`<!doctype html><html><body>
<div id="app"></div>
<script>document.getElementById("app").textContent = "rendered-by-js";</script>
</body></html>`))
	}))
	defer srv.Close()

	p := New(Options{Headless: true, Workers: 1, Settle: 300 * time.Millisecond, RenderTimeout: 20 * time.Second})
	defer func() { _ = p.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	res, err := p.Render(ctx, srv.URL)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(res.HTML, "rendered-by-js") {
		t.Errorf("render did not capture the JS-built DOM:\n%s", res.HTML)
	}
}
