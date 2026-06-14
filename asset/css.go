package asset

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/tamnd/kage/urlx"
)

// RefSink registers a resolved asset/page URL with the cloner and returns the
// string to write back into the markup or CSS — a relative local path for
// things kage saves, or the absolute URL for anything it leaves on the live web.
type RefSink func(u *url.URL, kind urlx.Kind) string

var (
	// url( ... ) with optional single/double quotes or bare token.
	cssURLRe = regexp.MustCompile(`url\(\s*("[^"]*"|'[^']*'|[^)'"]*)\s*\)`)
	// @import "..."  /  @import '...'  (the @import url(...) form is caught by cssURLRe).
	cssImportRe = regexp.MustCompile(`@import\s+("[^"]*"|'[^']*')`)
)

// RewriteCSS rewrites every url(...) and @import in a stylesheet so its
// references point at local files. base is the stylesheet's own URL (so relative
// references resolve correctly); sink maps each absolute URL to its local path.
// data: URLs and unparseable references are left untouched.
func RewriteCSS(css []byte, base *url.URL, sink RefSink) []byte {
	s := string(css)
	s = cssImportRe.ReplaceAllStringFunc(s, func(m string) string {
		raw := cssImportRe.FindStringSubmatch(m)[1]
		ref := unquote(raw)
		if newRef, ok := resolveRef(base, ref, sink); ok {
			return `@import "` + newRef + `"`
		}
		return m
	})
	s = cssURLRe.ReplaceAllStringFunc(s, func(m string) string {
		raw := cssURLRe.FindStringSubmatch(m)[1]
		ref := unquote(raw)
		if newRef, ok := resolveRef(base, ref, sink); ok {
			return `url("` + newRef + `")`
		}
		return m
	})
	return []byte(s)
}

// resolveRef normalises ref against base and runs it through the sink. It
// reports ok=false (leave the original text) for empty, data:, or unparseable
// references.
func resolveRef(base *url.URL, ref string, sink RefSink) (string, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(strings.ToLower(ref), "data:") || strings.HasPrefix(ref, "#") {
		return "", false
	}
	u, err := urlx.Normalize(base, ref)
	if err != nil {
		return "", false
	}
	return sink(u, urlx.Asset), true
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}
