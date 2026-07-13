package dao

import "testing"

func TestManageOrderColumnUsesWhitelistedValues(t *testing.T) {
	tests := map[string]string{
		"updated":   "updated_at",
		"title":     "title",
		"published": "published_at",
		"unknown":   "updated_at",
	}

	for sort, want := range tests {
		if got := manageOrderColumn(sort); got != want {
			t.Fatalf("manageOrderColumn(%q) = %q, want %q", sort, got, want)
		}
	}
}
