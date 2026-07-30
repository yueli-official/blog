package controller

import (
	"context"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/catalog"
)

// Cover handles the author (JWT) cover-image upload endpoints.
type Cover struct{ svc *catalog.Service }

func NewCover(svc *catalog.Service) *Cover { return &Cover{svc: svc} }

func (c *Cover) CoverInit(ctx context.Context, req *v1.CoverInitReq) (*v1.CoverInitRes, error) {
	_, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	resource, err := postResource(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(
		ctx, blogauthz.CapabilityPostUpdate, blogauthz.PostScopeID(req.ID), resource,
	); err != nil {
		return nil, err
	}
	out, err := c.svc.AddCover(ctx, resourceOwner(resource), bearerOf(ctx), req.ID, req.Filename, req.Mime, req.Size)
	if err != nil {
		return nil, err
	}
	return &v1.CoverInitRes{UploadURL: out.UploadURL, UploadToken: out.UploadToken, UploadHeaders: out.UploadHeaders}, nil
}

func (c *Cover) CoverFinalize(ctx context.Context, req *v1.CoverFinalizeReq) (*v1.CoverFinalizeRes, error) {
	_, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	resource, err := postResource(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(
		ctx, blogauthz.CapabilityPostUpdate, blogauthz.PostScopeID(req.ID), resource,
	); err != nil {
		return nil, err
	}
	assetID, coverURL, err := c.svc.FinalizeCover(ctx, resourceOwner(resource), bearerOf(ctx), req.ID, req.UploadToken)
	if err != nil {
		return nil, err
	}
	return &v1.CoverFinalizeRes{CoverAssetID: assetID, CoverURL: coverURL}, nil
}
