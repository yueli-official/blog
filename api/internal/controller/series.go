package controller

import (
	"context"

	v1 "platform/products/blog/api/api/v1"
	"platform/products/blog/api/internal/catalog"
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
	se, err := c.svc.CreateSeries(ctx, author, req.Name, req.Description)
	if err != nil {
		return nil, err
	}
	return &v1.CreateSeriesRes{Series: seriesView(se)}, nil
}

func (c *Series) UpdateSeries(ctx context.Context, req *v1.UpdateSeriesReq) (*v1.UpdateSeriesRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	se, err := c.svc.UpdateSeries(ctx, author, isAdmin(ctx), req.ID, req.Name, req.Description)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateSeriesRes{Series: seriesView(se)}, nil
}

func (c *Series) DeleteSeries(ctx context.Context, req *v1.DeleteSeriesReq) (*v1.DeleteSeriesRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.svc.DeleteSeries(ctx, author, isAdmin(ctx), req.ID); err != nil {
		return nil, err
	}
	return &v1.DeleteSeriesRes{Deleted: true}, nil
}
