package v1

import "testing"

func TestSeriesMutationContractsCarrySlug(t *testing.T) {
	slug := "release-notes"
	created := CreateSeriesReq{Slug: slug}
	updated := UpdateSeriesReq{Slug: &slug}
	if created.Slug != slug || updated.Slug == nil || *updated.Slug != slug {
		t.Fatal("series create and update contracts must carry slug")
	}
}
