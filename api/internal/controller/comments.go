package controller

import (
	"context"

	v1 "platform/products/blog/api/api/v1"
	"platform/products/blog/api/internal/catalog"
	"platform/products/blog/api/internal/model"
)

// Comments handles the author (JWT) comment-moderation endpoints.
type Comments struct{ svc *catalog.Service }

func NewComments(svc *catalog.Service) *Comments { return &Comments{svc: svc} }

func (c *Comments) ListMine(ctx context.Context, req *v1.ListMyCommentsReq) (*v1.ListMyCommentsRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	items, total, page, size, err := c.svc.ListMineComments(ctx, author, isAdmin(ctx), req.Status, req.Keyword, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	return &v1.ListMyCommentsRes{Items: commentAdminViews(items), Total: total, Page: page, Size: size}, nil
}

func (c *Comments) SetStatus(ctx context.Context, req *v1.SetCommentStatusReq) (*v1.SetCommentStatusRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	cm, err := c.svc.SetCommentStatus(ctx, author, isAdmin(ctx), req.ID, model.CommentStatus(req.Status))
	if err != nil {
		return nil, err
	}
	return &v1.SetCommentStatusRes{Comment: commentAdminViewBare(cm)}, nil
}

func (c *Comments) Delete(ctx context.Context, req *v1.DeleteCommentReq) (*v1.DeleteCommentRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.svc.DeleteComment(ctx, author, isAdmin(ctx), req.ID); err != nil {
		return nil, err
	}
	return &v1.DeleteCommentRes{Deleted: true}, nil
}
