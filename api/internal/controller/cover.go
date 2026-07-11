package controller

import (
	"context"

	v1 "platform/products/blog/api/api/v1"
	"platform/products/blog/api/internal/catalog"
)

// Cover handles the author (JWT) cover-image upload endpoints.
type Cover struct{ svc *catalog.Service }

func NewCover(svc *catalog.Service) *Cover { return &Cover{svc: svc} }

func (c *Cover) CoverInit(ctx context.Context, req *v1.CoverInitReq) (*v1.CoverInitRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	out, err := c.svc.AddCover(ctx, author, bearerOf(ctx), req.ID, req.Filename, req.Mime, req.Size)
	if err != nil {
		return nil, err
	}
	return &v1.CoverInitRes{UploadURL: out.UploadURL, UploadToken: out.UploadToken, UploadHeaders: out.UploadHeaders}, nil
}

func (c *Cover) CoverFinalize(ctx context.Context, req *v1.CoverFinalizeReq) (*v1.CoverFinalizeRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	assetID, coverURL, err := c.svc.FinalizeCover(ctx, author, bearerOf(ctx), req.ID, req.UploadToken)
	if err != nil {
		return nil, err
	}
	return &v1.CoverFinalizeRes{CoverAssetID: assetID, CoverURL: coverURL}, nil
}
