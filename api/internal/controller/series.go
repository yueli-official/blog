package controller

import (
	"context"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/catalog"
	"github.com/yueli-official/foundation/go/authorization"
)

// PublicSeries handles the public series browse endpoints (no mandatory auth).
type PublicSeries struct{ svc *catalog.Service }

func NewPublicSeries(svc *catalog.Service) *PublicSeries { return &PublicSeries{svc: svc} }

func (c *PublicSeries) ListSeries(ctx context.Context, _ *v1.ListSeriesReq) (*v1.ListSeriesRes, error) {
	items, err := c.svc.ListSeries(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ListSeriesRes{Items: seriesViews(items)}, nil
}

func (c *PublicSeries) GetSeries(ctx context.Context, req *v1.GetSeriesReq) (*v1.GetSeriesRes, error) {
	se, posts, err := c.svc.SeriesWithPosts(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return &v1.GetSeriesRes{Series: seriesView(se), Posts: postViews(posts)}, nil
}

// Series handles the author (JWT) series-management endpoints.
type Series struct{ svc *catalog.Service }

func NewSeries(svc *catalog.Service) *Series { return &Series{svc: svc} }

func (c *Series) CreateSeries(ctx context.Context, req *v1.CreateSeriesReq) (*v1.CreateSeriesRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(
		ctx, blogauthz.CapabilitySeriesCreate, blogauthz.RootScopeID,
		authorization.ResourceFacts{},
	); err != nil {
		return nil, err
	}
	se, err := c.svc.CreateSeries(ctx, author, req.Name, req.Slug, req.Description)
	if err != nil {
		return nil, err
	}
	if err := authorizationService(ctx).EnsureSeriesScope(ctx, se.ID); err != nil {
		return nil, mapAuthorizationError(err)
	}
	writeCreated(ctx)
	return &v1.CreateSeriesRes{Series: seriesView(se)}, nil
}

func (c *Series) UpdateSeries(ctx context.Context, req *v1.UpdateSeriesReq) (*v1.UpdateSeriesRes, error) {
	_, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	resource, err := seriesResource(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(
		ctx, blogauthz.CapabilitySeriesUpdate, blogauthz.SeriesScopeID(req.ID), resource,
	); err != nil {
		return nil, err
	}
	se, err := c.svc.UpdateSeries(ctx, resourceOwner(resource), isAdmin(ctx), req.ID, req.Name, req.Slug, req.Description)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateSeriesRes{Series: seriesView(se)}, nil
}

func (c *Series) DeleteSeries(ctx context.Context, req *v1.DeleteSeriesReq) (*v1.DeleteSeriesRes, error) {
	_, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	resource, err := seriesResource(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(
		ctx, blogauthz.CapabilitySeriesDelete, blogauthz.SeriesScopeID(req.ID), resource,
	); err != nil {
		return nil, err
	}
	if err := c.svc.DeleteSeries(ctx, resourceOwner(resource), isAdmin(ctx), req.ID); err != nil {
		return nil, err
	}
	writeNoContent(ctx)
	return &v1.DeleteSeriesRes{}, nil
}
