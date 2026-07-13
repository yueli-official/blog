package dao

import (
	"strings"
	"testing"
)

func TestManageOrderUsesWhitelistedColumnsAndKeepsUnpublishedLast(t *testing.T) {
	tests := map[string]string{
		"updated/desc":   "updated_at DESC",
		"title/asc":      "title ASC",
		"published/desc": "published_at DESC NULLS LAST",
		"published/asc":  "published_at ASC NULLS LAST",
		"unknown/other":  "updated_at DESC",
	}

	for input, want := range tests {
		parts := strings.SplitN(input, "/", 2)
		if got := manageOrder(parts[0], parts[1]); got != want {
			t.Fatalf("manageOrder(%q, %q) = %q, want %q", parts[0], parts[1], got, want)
		}
	}
}
