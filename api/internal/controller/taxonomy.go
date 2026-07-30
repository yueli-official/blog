package controller

import (
	"context"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/catalog"
	"github.com/yueli-official/foundation/go/authorization"
)

// Taxonomy handles the author (JWT) category/tag management endpoints.
type Taxonomy struct{ svc *catalog.Service }

func NewTaxonomy(svc *catalog.Service) *Taxonomy { return &Taxonomy{svc: svc} }

func (c *Taxonomy) CreateTaxonomy(ctx context.Context, req *v1.CreateTaxonomyReq) (*v1.CreateTaxonomyRes, error) {
	// Categories are curated structure (administrator only);
	// tags are folksonomy any logged-in author may create freely.
	if req.Taxonomy == "category" {
		if err := requireCapability(
			ctx, blogauthz.CapabilityTaxonomyManage, blogauthz.RootScopeID,
			authorization.ResourceFacts{},
		); err != nil {
			return nil, err
		}
	} else if err := requireCapability(
		ctx, blogauthz.CapabilityTagCreate, blogauthz.RootScopeID,
		authorization.ResourceFacts{},
	); err != nil {
		return nil, err
	}
	tx, err := c.svc.CreateTaxonomy(ctx, req.Name, req.Taxonomy, req.Slug, req.ParentID, req.Description)
	if err != nil {
		return nil, err
	}
	return &v1.CreateTaxonomyRes{Taxonomy: taxonomyView(tx)}, nil
}

func (c *Taxonomy) UpdateTaxonomy(ctx context.Context, req *v1.UpdateTaxonomyReq) (*v1.UpdateTaxonomyRes, error) {
	if err := requireCapability(
		ctx, blogauthz.CapabilityTaxonomyManage, blogauthz.RootScopeID,
		authorization.ResourceFacts{},
	); err != nil {
		return nil, err
	}
	tx, err := c.svc.UpdateTaxonomy(ctx, req.ID, req.Name, req.Slug, req.Description, req.ParentID)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateTaxonomyRes{Taxonomy: taxonomyView(tx)}, nil
}

func (c *Taxonomy) DeleteTaxonomy(ctx context.Context, req *v1.DeleteTaxonomyReq) (*v1.DeleteTaxonomyRes, error) {
	if err := requireCapability(
		ctx, blogauthz.CapabilityTaxonomyManage, blogauthz.RootScopeID,
		authorization.ResourceFacts{},
	); err != nil {
		return nil, err
	}
	if err := c.svc.DeleteTaxonomy(ctx, req.ID); err != nil {
		return nil, err
	}
	return &v1.DeleteTaxonomyRes{Deleted: true}, nil
}

func (c *Taxonomy) MergeTaxonomy(ctx context.Context, req *v1.MergeTaxonomyReq) (*v1.MergeTaxonomyRes, error) {
	if err := requireCapability(
		ctx, blogauthz.CapabilityTaxonomyManage, blogauthz.RootScopeID,
		authorization.ResourceFacts{},
	); err != nil {
		return nil, err
	}
	if err := c.svc.MergeTaxonomy(ctx, req.ID, req.TargetID); err != nil {
		return nil, err
	}
	return &v1.MergeTaxonomyRes{Merged: true}, nil
}

func (c *Taxonomy) AssignTaxonomies(ctx context.Context, req *v1.AssignTaxonomiesReq) (*v1.AssignTaxonomiesRes, error) {
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
	if err := c.svc.AssignTaxonomies(ctx, resourceOwner(resource), req.ID, req.TaxonomyIDs); err != nil {
		return nil, err
	}
	return &v1.AssignTaxonomiesRes{Updated: true}, nil
}
