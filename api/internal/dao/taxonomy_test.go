package dao

import "testing"

func TestTaxonomyListOrderAllowlist(t *testing.T) {
	tests := []struct {
		name, sort, direction, want string
	}{
		{"default rejects SQL", "name; DROP TABLE blog_tags", "desc;--", "name ASC, id ASC"},
		{"count descending", "postCount", "desc", "post_count DESC, name ASC, id ASC"},
		{"slug ascending", "slug", "asc", "slug ASC, name ASC, id ASC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := taxonomyListOrder(tt.sort, tt.direction); got != tt.want {
				t.Fatalf("taxonomyListOrder() = %q, want %q", got, tt.want)
			}
		})
	}
}
