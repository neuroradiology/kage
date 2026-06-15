package clone

import (
	"testing"

	"github.com/tamnd/kage/urlx"
)

// TestWantAsset checks the two URL-only skip rules a page worker applies before
// downloading an asset: off-domain hosts and bulk-media extensions are left on
// the live web, while the seed's own images, fonts, and stylesheets localize.
func TestWantAsset(t *testing.T) {
	seed, _ := urlx.ParseSeed("https://developer.apple.com/")
	c := New(seed, DefaultConfig(), nil)

	cases := []struct {
		u    string
		want bool
	}{
		// Same registrable domain: localize.
		{"https://developer.apple.com/css/main.css", true},
		{"https://www.apple.com/img/logo.png", true},
		{"https://images.apple.com/fonts/sf.woff2", true},
		// Bulk media, installers, archives, and PDFs: leave remote.
		{"https://developer.apple.com/videos/wwdc.mp4", false},
		{"https://developer.apple.com/downloads/Xcode.dmg", false},
		{"https://developer.apple.com/bundle.zip", false},
		{"https://developer.apple.com/guide.pdf", false},
		{"https://developer.apple.com/clip.MP4", false}, // case-insensitive
		// Off-domain hosts: leave remote even for a normal image.
		{"https://cdn-apple.com/img/x.png", false},
		{"https://ec.europa.eu/banner.png", false},
		{"https://mmbiz.qpic.cn/x.jpg", false},
	}
	for _, tc := range cases {
		u := mustURL(t, tc.u)
		if got := c.wantAsset(u); got != tc.want {
			t.Errorf("wantAsset(%q) = %v, want %v", tc.u, got, tc.want)
		}
	}
}

// TestWantAssetAllHosts checks that turning the domain scope off localizes a
// third-party asset, while the media extension rule still applies.
func TestWantAssetAllHosts(t *testing.T) {
	seed, _ := urlx.ParseSeed("https://developer.apple.com/")
	cfg := DefaultConfig()
	cfg.AssetSameDomain = false
	c := New(seed, cfg, nil)

	if !c.wantAsset(mustURL(t, "https://cdn-apple.com/img/x.png")) {
		t.Error("with AssetSameDomain off, an off-domain image should localize")
	}
	if c.wantAsset(mustURL(t, "https://cdn-apple.com/video/x.mp4")) {
		t.Error("a media file should still be skipped regardless of host scope")
	}
}

// TestWantAssetKeepMedia checks that an empty skip set (the --keep-media case)
// localizes media too.
func TestWantAssetKeepMedia(t *testing.T) {
	seed, _ := urlx.ParseSeed("https://developer.apple.com/")
	cfg := DefaultConfig()
	cfg.SkipAssetExts = map[string]bool{}
	c := New(seed, cfg, nil)

	if !c.wantAsset(mustURL(t, "https://developer.apple.com/videos/wwdc.mp4")) {
		t.Error("with an empty skip set, an on-domain video should localize")
	}
}
