package catalog

import (
	"regexp"
	"strings"
)

// SpamPolicy is the comment anti-spam configuration. It's code/config-level
// (built from blog.commentGuard.*) with sensible defaults — there is no runtime
// admin to edit it (self-hosted site, not a multi-tenant SaaS — see the donor
// absorption principle). A zero value disables every guard.
type SpamPolicy struct {
	Blacklist         []string // case-insensitive substrings → hard-reject the comment
	MaxLinks          int      // a comment with more URLs than this is forced to pending (0 = off)
	RatePerWindow     int      // max comments per IP within the window (0 = off)
	RateWindowSeconds int      // the rate window, in seconds
}

// linkRe matches the start of a URL — http(s):// or a bare www. host. Good
// enough to count links for spam heuristics (not a strict URL parser).
var linkRe = regexp.MustCompile(`(?i)(https?://|www\.)`)

// countLinks returns how many URL-like tokens the content contains.
func countLinks(content string) int {
	return len(linkRe.FindAllString(content, -1))
}

// containsBlacklisted reports whether content contains any blacklisted substring
// (case-insensitive). Empty/blank list entries are ignored.
func containsBlacklisted(content string, list []string) bool {
	if len(list) == 0 {
		return false
	}
	lc := strings.ToLower(content)
	for _, w := range list {
		w = strings.ToLower(strings.TrimSpace(w))
		if w != "" && strings.Contains(lc, w) {
			return true
		}
	}
	return false
}
