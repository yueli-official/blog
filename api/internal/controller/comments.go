package controller

import (
	"context"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/catalog"
	"github.com/yueli-official/blog/api/internal/model"
)

// Comments handles the author (JWT) comment-moderation endpoints.
type Comments struct{ svc *catalog.Service }

func NewComments(svc *catalog.Service) *Comments { return &Comments{svc: svc} }

func (c *Comments) ListMine(ctx context.Context, req *v1.ListMyCommentsReq) (*v1.ListMyCommentsRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := authorizationService(ctx).ManagePostOwner(ctx); err != nil {
		return nil, mapAuthorizationError(err)
	}
	items, total, page, size, err := c.svc.ListMineComments(ctx, author, isAdmin(ctx), req.Status, req.Keyword, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	return &v1.ListMyCommentsRes{Items: commentAdminViews(items), Total: total, Page: page, Size: size}, nil
}

func (c *Comments) SetStatus(ctx context.Context, req *v1.SetCommentStatusReq) (*v1.SetCommentStatusRes, error) {
	_, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	resource, err := commentResource(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(
		ctx, blogauthz.CapabilityCommentModerate, blogauthz.CommentScopeID(req.ID), resource,
	); err != nil {
		return nil, err
	}
	cm, err := c.svc.SetCommentStatus(ctx, resourceOwner(resource), isAdmin(ctx), req.ID, model.CommentStatus(req.Status))
	if err != nil {
		return nil, err
	}
	return &v1.SetCommentStatusRes{Comment: commentAdminViewBare(cm)}, nil
}

func (c *Comments) Delete(ctx context.Context, req *v1.DeleteCommentReq) (*v1.DeleteCommentRes, error) {
	_, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	resource, err := commentResource(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(
		ctx, blogauthz.CapabilityCommentDelete, blogauthz.CommentScopeID(req.ID), resource,
	); err != nil {
		return nil, err
	}
	if err := c.svc.DeleteComment(ctx, resourceOwner(resource), isAdmin(ctx), req.ID); err != nil {
		return nil, err
	}
	return &v1.DeleteCommentRes{Deleted: true}, nil
}
