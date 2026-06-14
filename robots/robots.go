// Package robots is a small, pure robots.txt parser and matcher. kage obeys it
// by default so a clone stays polite; --no-robots bypasses the matcher and puts
// the consequences on the user.
package robots

import (
	"strconv"
	"strings"
	"time"
)

// rule is one Allow or Disallow directive.
type rule struct {
	pattern string
	allow   bool
}

// Matcher answers whether a path may be crawled for one user-agent group, and
// carries the sitemaps and crawl-delay declared in the file.
type Matcher struct {
	rules      []rule
	Sitemaps   []string
	CrawlDelay time.Duration
}

// AllowAll returns a Matcher that permits every path (used when robots is
// disabled or the file is absent).
func AllowAll() *Matcher { return &Matcher{} }

// group accumulates the rules attached to a set of user-agent lines.
type group struct {
	agents []string
	rules  []rule
	delay  time.Duration
}

// Parse reads robots.txt content and returns a Matcher for the given agent
// token (e.g. "kage"). The most specific matching group is used, falling back to
// the "*" group. Sitemaps are collected globally.
func Parse(data string, agent string) *Matcher {
	agent = strings.ToLower(agent)
	var groups []*group
	var cur *group
	var sitemaps []string
	startedRules := false

	for line := range strings.SplitSeq(data, "\n") {
		line = stripComment(line)
		key, val, ok := splitField(line)
		if !ok {
			continue
		}
		switch strings.ToLower(key) {
		case "user-agent":
			// A user-agent line after rules starts a fresh group.
			if cur == nil || startedRules {
				cur = &group{}
				groups = append(groups, cur)
				startedRules = false
			}
			cur.agents = append(cur.agents, strings.ToLower(val))
		case "allow":
			if cur != nil {
				cur.rules = append(cur.rules, rule{pattern: val, allow: true})
				startedRules = true
			}
		case "disallow":
			if cur != nil {
				cur.rules = append(cur.rules, rule{pattern: val, allow: false})
				startedRules = true
			}
		case "crawl-delay":
			if cur != nil {
				if d := parseDelay(val); d > 0 {
					cur.delay = d
				}
				startedRules = true
			}
		case "sitemap":
			if val != "" {
				sitemaps = append(sitemaps, val)
			}
		}
	}

	m := &Matcher{Sitemaps: sitemaps}
	if g := selectGroup(groups, agent); g != nil {
		m.rules = g.rules
		m.CrawlDelay = g.delay
	}
	return m
}

// selectGroup picks the group whose agent best matches: an exact/substring
// match on the agent token beats the wildcard "*" group.
func selectGroup(groups []*group, agent string) *group {
	var star, specific *group
	bestLen := -1
	for _, g := range groups {
		for _, a := range g.agents {
			if a == "*" {
				star = g
				continue
			}
			if strings.HasPrefix(agent, a) || strings.Contains(agent, a) {
				if len(a) > bestLen {
					bestLen = len(a)
					specific = g
				}
			}
		}
	}
	if specific != nil {
		return specific
	}
	return star
}

// Allowed reports whether path may be crawled. The longest matching rule wins;
// on a tie, Allow beats Disallow. An empty Disallow means "allow everything".
func (m *Matcher) Allowed(path string) bool {
	if path == "" {
		path = "/"
	}
	bestLen := -1
	bestAllow := true
	for _, r := range m.rules {
		if r.pattern == "" {
			// Empty Disallow = allow all; empty Allow is a no-op.
			continue
		}
		if matchPattern(r.pattern, path) {
			pl := len(r.pattern)
			if pl > bestLen || (pl == bestLen && r.allow) {
				bestLen = pl
				bestAllow = r.allow
			}
		}
	}
	if bestLen < 0 {
		return true
	}
	return bestAllow
}

// matchPattern matches a robots path pattern against path, honouring '*'
// (any run) and a trailing '$' (end anchor).
func matchPattern(pattern, path string) bool {
	anchored := strings.HasSuffix(pattern, "$")
	if anchored {
		pattern = pattern[:len(pattern)-1]
	}
	segs := strings.Split(pattern, "*")
	pos := 0
	for i, seg := range segs {
		if seg == "" {
			continue
		}
		if i == 0 {
			if !strings.HasPrefix(path[pos:], seg) {
				return false
			}
			pos += len(seg)
			continue
		}
		idx := strings.Index(path[pos:], seg)
		if idx < 0 {
			return false
		}
		pos += idx + len(seg)
	}
	if anchored {
		// The last non-empty segment must land exactly at the end.
		last := ""
		for i := len(segs) - 1; i >= 0; i-- {
			if segs[i] != "" {
				last = segs[i]
				break
			}
		}
		if last != "" && !strings.HasSuffix(path, last) {
			return false
		}
		// A pattern with no trailing wildcard must consume the whole path.
		if !strings.HasSuffix(pattern, "*") && pos != len(path) {
			return false
		}
	}
	return true
}

func stripComment(line string) string {
	if before, _, found := strings.Cut(line, "#"); found {
		return before
	}
	return line
}

func splitField(line string) (key, val string, ok bool) {
	k, v, found := strings.Cut(line, ":")
	if !found {
		return "", "", false
	}
	key = strings.TrimSpace(k)
	val = strings.TrimSpace(v)
	if key == "" {
		return "", "", false
	}
	return key, val, true
}

func parseDelay(val string) time.Duration {
	secs, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
	if err != nil || secs <= 0 {
		return 0
	}
	return time.Duration(secs * float64(time.Second))
}
