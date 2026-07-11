package controller

import (
	"context"

	v1 "platform/products/blog/api/api/v1"
	"platform/products/blog/api/internal/catalog"
)

// Images handles the author (JWT) inline content-image upload endpoints (editor
// E2). The image is standalone (not post-bound); the editor embeds the returned
// public URL into the markdown.
type Images struct{ svc *catalog.Service }

func NewImages(svc *catalog.Service) *Images { return &Images{svc: svc} }

func (c *Images) ImageInit(ctx context.Context, req *v1.ImageInitReq) (*v1.ImageInitRes, error) {
	if _, err := subject(ctx); err != nil { // auth gate (sub unused for a standalone image)
		return nil, err
	}
	out, err := c.svc.InitImage(ctx, bearerOf(ctx), req.Filename, req.Mime, req.Size)
	if err != nil {
		return nil, err
	}
	return &v1.ImageInitRes{UploadURL: out.UploadURL, UploadToken: out.UploadToken, UploadHeaders: out.UploadHeaders}, nil
}

func (c *Images) ImageFinalize(ctx context.Context, req *v1.ImageFinalizeReq) (*v1.ImageFinalizeRes, error) {
	if _, err := subject(ctx); err != nil {
		return nil, err
	}
	url, err := c.svc.FinalizeImage(ctx, bearerOf(ctx), req.UploadToken)
	if err != nil {
		return nil, err
	}
	return &v1.ImageFinalizeRes{URL: url}, nil
}
