package asset

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Downloader fetches asset bytes over plain HTTP. It is separate from the Chrome
// pool: assets are public bytes that rarely need a real browser, so a fast HTTP
// client keeps the crawl cheap. Failures are returned to the caller, which logs
// them and moves on — a missing asset degrades a page, it never aborts a clone.
type Downloader struct {
	Client    *http.Client
	UserAgent string
	MaxBytes  int64 // per-asset cap; 0 = unlimited
}

// NewDownloader builds a Downloader with a sane client and the given timeout.
func NewDownloader(userAgent string, timeout time.Duration, maxBytes int64) *Downloader {
	return &Downloader{
		Client:    &http.Client{Timeout: timeout},
		UserAgent: userAgent,
		MaxBytes:  maxBytes,
	}
}

// Result is a downloaded asset.
type Result struct {
	Body        []byte
	ContentType string
	IsCSS       bool
}

// Get fetches u, sending referer as the Referer header. It reads at most
// MaxBytes and reports whether the body is CSS (so the caller can rewrite it).
func (d *Downloader) Get(ctx context.Context, u *url.URL, referer string) (*Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if d.UserAgent != "" {
		req.Header.Set("User-Agent", d.UserAgent)
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d for %s", resp.StatusCode, u)
	}
	var r io.Reader = resp.Body
	if d.MaxBytes > 0 {
		r = io.LimitReader(resp.Body, d.MaxBytes)
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	ct := resp.Header.Get("Content-Type")
	return &Result{
		Body:        body,
		ContentType: ct,
		IsCSS:       isCSS(ct, u),
	}, nil
}

// isCSS reports whether a response is a stylesheet, by content-type or by a
// .css path when the server sends no useful type.
func isCSS(contentType string, u *url.URL) bool {
	if strings.Contains(strings.ToLower(contentType), "text/css") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(u.Path), ".css")
}
