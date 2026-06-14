package robots

import "testing"

const sample = `
# example robots
User-agent: *
Disallow: /private/
Allow: /private/public
Disallow: /*.json$
Crawl-delay: 2

User-agent: kage
Disallow: /no-kage/

Sitemap: https://ex.com/sitemap.xml
Sitemap: https://ex.com/news.xml
`

func TestParseAndAllowedSpecificAgent(t *testing.T) {
	m := Parse(sample, "kage")
	// kage's own group only disallows /no-kage/.
	if m.Allowed("/private/secret") == false {
		t.Error("kage group should allow /private/secret (only /no-kage/ is blocked)")
	}
	if m.Allowed("/no-kage/x") {
		t.Error("/no-kage/x should be disallowed for kage")
	}
}

func TestAllowedWildcardGroup(t *testing.T) {
	m := Parse(sample, "somebot")
	cases := map[string]bool{
		"/":                true,
		"/private/x":       false, // Disallow /private/
		"/private/public":  true,  // longer Allow wins on tie/length
		"/data/file.json":  false, // Disallow /*.json$
		"/data/file.jsonx": true,  // $ anchors the extension
		"/ok":              true,
	}
	for p, want := range cases {
		if got := m.Allowed(p); got != want {
			t.Errorf("Allowed(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestSitemapsAndDelay(t *testing.T) {
	m := Parse(sample, "somebot")
	if len(m.Sitemaps) != 2 {
		t.Fatalf("got %d sitemaps, want 2", len(m.Sitemaps))
	}
	if m.Sitemaps[0] != "https://ex.com/sitemap.xml" {
		t.Errorf("sitemap[0] = %q", m.Sitemaps[0])
	}
	if m.CrawlDelay.Seconds() != 2 {
		t.Errorf("crawl-delay = %v, want 2s", m.CrawlDelay)
	}
}

func TestEmptyDisallowAllowsAll(t *testing.T) {
	m := Parse("User-agent: *\nDisallow:\n", "x")
	if !m.Allowed("/anything") {
		t.Error("empty Disallow should allow everything")
	}
}

func TestNoRobotsAllowsAll(t *testing.T) {
	m := AllowAll()
	if !m.Allowed("/whatever") {
		t.Error("AllowAll must allow everything")
	}
}
