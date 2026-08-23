package controller

import (
	"testing"

	"github.com/yueli-official/blog/api/internal/model"
)

func TestPostViewHidesLegacyBackendCoverURL(t *testing.T) {
	view := postView(&model.Post{
		ID: "post-1", Title: "Post", CoverAssetID: "asset-1",
		CoverURL: "https://bucket.cos.ap-shanghai.myqcloud.com/public/blog/cover.webp",
	})
	if view.CoverURL != "/asset-api/assets/public/blog/cover.webp" || view.CoverAssetID != "asset-1" {
		t.Fatalf("post view = %#v", view)
	}
}

func TestPostViewKeepsCanonicalMediaCoverURL(t *testing.T) {
	view := postView(&model.Post{
		ID: "post-1", Title: "Post", CoverAssetID: "asset-1",
		CoverURL: "/media/key?format=webp&name=home",
	})
	if view.CoverURL != "/media/key?format=webp&name=home" {
		t.Fatalf("post view = %#v", view)
	}
}
