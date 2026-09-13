package controller

import (
	"context"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/catalog"
	foundationauth "github.com/yueli-official/foundation/go/auth"
	"github.com/yueli-official/foundation/go/authorization"
)

// Images handles the author (JWT) inline content-image upload endpoints (editor
// E2). The image is standalone (not post-bound); the editor embeds the returned
// public URL into the markdown.
type Images struct{ svc *catalog.Service }

func NewImages(svc *catalog.Service) *Images { return &Images{svc: svc} }

func (c *Images) ImageInit(ctx context.Context, req *v1.ImageInitReq) (*v1.ImageInitRes, error) {
	if err := requireImageUpload(ctx); err != nil {
		return nil, err
	}
	out, err := c.svc.InitImage(ctx, bearerOf(ctx), req.Filename, req.Mime, req.Size, req.Preprocessed)
	if err != nil {
		return nil, err
	}
	writeCreated(ctx)
	return &v1.ImageInitRes{UploadURL: out.UploadURL, UploadToken: out.UploadToken, UploadHeaders: out.UploadHeaders}, nil
}

func (c *Images) ImageFinalize(ctx context.Context, req *v1.ImageFinalizeReq) (*v1.ImageFinalizeRes, error) {
	if err := requireImageUpload(ctx); err != nil {
		return nil, err
	}
	url, err := c.svc.FinalizeImage(ctx, bearerOf(ctx), req.UploadToken)
	if err != nil {
		return nil, err
	}
	return &v1.ImageFinalizeRes{URL: url}, nil
}

func requireImageUpload(ctx context.Context) error {
	p, _ := foundationauth.FromContext(ctx)
	if p != nil && p.IsPersonalToken() {
		_, err := NewPersonalPermissions(p.ClientID, authorizationService(ctx)).AuthorizePersonalMedia(ctx, &v1.PersonalMediaAuthorizationReq{})
		return err
	}
	return requireCapability(ctx, blogauthz.CapabilityPostCreate, blogauthz.RootScopeID, authorization.ResourceFacts{})
}
