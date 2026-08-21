package controller

import "testing"

func TestNormalizeTrafficSource(t *testing.T) {
	tests := map[string]string{
		"":                         "direct",
		"direct":                   "direct",
		"internal":                 "internal",
		" WWW.Google.COM. ":        "google.com",
		"news.example.com":         "news.example.com",
		"https://example.com/path": "direct",
		"localhost":                "direct",
		"bad..example":             "direct",
	}
	for input, expected := range tests {
		if actual := normalizeTrafficSource(input); actual != expected {
			t.Errorf("normalizeTrafficSource(%q) = %q, want %q", input, actual, expected)
		}
	}
}
