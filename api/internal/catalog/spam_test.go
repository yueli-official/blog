package catalog

import "testing"

// Pure-logic tests for the anti-spam helpers — no PG, always run.

func TestCountLinks(t *testing.T) {
	cases := map[string]int{
		"no links here":                    0,
		"visit https://example.com":        1,
		"http://a.com and http://b.com":    2,
		"mix https://a.com and www.b.com":  2,
		"WWW.UPPER.COM and HTTPS://X.COM":  2, // case-insensitive
		"plain example.com without scheme": 0, // bare host isn't matched
	}
	for in, want := range cases {
		if got := countLinks(in); got != want {
			t.Errorf("countLinks(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestContainsBlacklisted(t *testing.T) {
	list := []string{"casino", " Free Money "} // trimmed + case-folded on match
	hits := []string{"play at the CASINO tonight", "claim your free money now"}
	for _, s := range hits {
		if !containsBlacklisted(s, list) {
			t.Errorf("containsBlacklisted(%q) = false, want true", s)
		}
	}
	// substring match is intentional ("casino" also catches "casinos") — a miss
	// must share no blacklisted substring at all.
	clean := "a perfectly ordinary comment about gardening"
	if containsBlacklisted(clean, list) {
		t.Errorf("containsBlacklisted(%q) = true, want false", clean)
	}
	// empty/blank list never matches
	if containsBlacklisted("anything", nil) || containsBlacklisted("anything", []string{"", "   "}) {
		t.Error("empty/blank blacklist should never match")
	}
}
