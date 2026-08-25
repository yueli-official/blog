package controller

import (
	"testing"

	"github.com/yueli-official/blog/api/internal/identityclient"
	"github.com/yueli-official/blog/api/internal/model"
)

func TestAuthorViewProjectsIdentityPublicProfile(t *testing.T) {
	view := authorView("user-1", &authorAccessState{Role: "author", Status: "active"}, identityclient.PublicUser{
		UserKey:     "user-1",
		Handle:      "writer",
		DisplayName: "Writer",
		Bio:         "Public introduction",
		Avatar:      &identityclient.MediaRef{MediaKey: "identity/avatar"},
		Cover:       &identityclient.MediaRef{MediaKey: "identity/cover"},
		SocialLinks: []identityclient.SocialLink{{Label: "GitHub", URL: "https://github.com/writer"}},
	}, 3)
	if view.Handle != "writer" || view.DisplayName != "Writer" || view.Bio != "Public introduction" {
		t.Fatalf("author identity projection = %#v", view)
	}
	if view.AvatarURL != "/media/identity/avatar?format=webp&name=thumbnail" || view.BannerURL != "/media/identity/cover?format=webp&name=cover" {
		t.Fatalf("author media projection = %#v", view)
	}
	if len(view.SocialLinks) != 1 || view.SocialLinks[0].Label != "GitHub" || view.PostCount != 3 {
		t.Fatalf("author social/count projection = %#v", view)
	}
}

func TestCommentViewProjectsCurrentIdentityProfile(t *testing.T) {
	comment := &model.Comment{ID: "comment-1", UserID: "user-1", AuthorName: "legacy-subject", Content: "hello"}
	profiles := map[string]identityclient.PublicUser{
		"user-1": {
			UserKey: "user-1", DisplayName: "Current Name",
			Avatar: &identityclient.MediaRef{MediaKey: "identity/comment-avatar"},
		},
	}
	view := commentViewWithProfiles(comment, profiles)
	if view.AuthorName != "Current Name" || view.AvatarURL != "/media/identity/comment-avatar?format=webp&name=thumbnail" {
		t.Fatalf("comment identity projection = %#v", view)
	}
}

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
