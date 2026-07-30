package controller

import (
	"context"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/catalog"
)

// Reactions handles the author (JWT) like/bookmark toggle endpoints.
type Reactions struct{ svc *catalog.Service }

func NewReactions(svc *catalog.Service) *Reactions { return &Reactions{svc: svc} }

func (c *Reactions) Like(ctx context.Context, req *v1.LikeReq) (*v1.LikeRes, error) {
	user, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	liked, err := c.svc.ToggleLike(ctx, user, req.Slug)
	if err != nil {
		return nil, err
	}
	return &v1.LikeRes{Liked: liked}, nil
}

func (c *Reactions) Bookmark(ctx context.Context, req *v1.BookmarkReq) (*v1.BookmarkRes, error) {
	user, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	bookmarked, err := c.svc.ToggleBookmark(ctx, user, req.Slug)
	if err != nil {
		return nil, err
	}
	return &v1.BookmarkRes{Bookmarked: bookmarked}, nil
}
