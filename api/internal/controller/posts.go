package controller

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/yueli-official/foundation/go/authorization"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/blogerr"
	"github.com/yueli-official/blog/api/internal/catalog"
)

// Posts handles the author (JWT) post-management endpoints.
type Posts struct{ svc *catalog.Service }

func NewPosts(svc *catalog.Service) *Posts { return &Posts{svc: svc} }

func (c *Posts) ListMine(ctx context.Context, req *v1.ListMineReq) (*v1.ListMineRes, error) {
	scope, err := authorizationService(ctx).ManagePostOwner(ctx)
	if err != nil {
		return nil, mapAuthorizationError(err)
	}
	if scope == "" {
		if req.AuthorID != "" {
			scope = req.AuthorID
		}
	}
	items, total, page, size, err := c.svc.ListManage(ctx, scope, req.Status, req.Q, req.TaxonomyIds, req.Pinned, req.Featured, req.SortBy, req.SortOrder, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	if err := c.svc.HydratePostTaxonomies(ctx, items); err != nil {
		return nil, err
	}
	counts, _ := c.svc.StatusCounts(ctx, scope)
	var views int64
	if scope != "" {
		views, _ = c.svc.MyTotalViews(ctx, scope)
	}
	return &v1.ListMineRes{Items: postViews(items), Total: total, Page: page, Size: size, Counts: counts, TotalViews: views}, nil
}

func (c *Posts) CreatePost(ctx context.Context, req *v1.CreatePostReq) (*v1.CreatePostRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(
		ctx, blogauthz.CapabilityPostCreate, blogauthz.RootScopeID,
		authorization.ResourceFacts{},
	); err != nil {
		return nil, err
	}
	p, err := c.svc.Create(ctx, author, req.Title, req.Content, req.Excerpt)
	if err != nil {
		return nil, err
	}
	if err := authorizationService(ctx).EnsurePostScope(ctx, p.ID); err != nil {
		return nil, blogerr.AuthorizationUnavailable()
	}
	writeCreated(ctx)
	return &v1.CreatePostRes{Post: postView(p)}, nil
}

func (c *Posts) PatchPost(ctx context.Context, req *v1.PatchPostReq) (*v1.PatchPostRes, error) {
	_, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	resource, err := postResource(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	fields := g.Map{}
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Slug != nil {
		fields["slug"] = *req.Slug
	}
	if req.Content != nil {
		fields["content"] = *req.Content
	}
	if req.Excerpt != nil {
		fields["excerpt"] = *req.Excerpt
	}
	if req.PublishedAt != nil {
		publishedAt, parseErr := time.Parse(time.RFC3339, *req.PublishedAt)
		if parseErr != nil {
			return nil, blogerr.InvalidInput("published_at_invalid")
		}
		fields["published_at"] = publishedAt.UTC()
	}
	if req.Status != nil {
		fields["status"] = *req.Status
		capability := blogauthz.CapabilityPostUpdate
		switch *req.Status {
		case "published":
			capability = blogauthz.CapabilityPostPublish
		case "archived":
			capability = blogauthz.CapabilityPostArchive
		}
		if err := requireCapability(ctx, capability, blogauthz.PostScopeID(req.ID), resource); err != nil {
			return nil, err
		}
	} else if err := requireCapability(
		ctx, blogauthz.CapabilityPostUpdate, blogauthz.PostScopeID(req.ID), resource,
	); err != nil {
		return nil, err
	}
	p, err := c.svc.Patch(ctx, resourceOwner(resource), req.ID, fields)
	if err != nil {
		return nil, err
	}
	return &v1.PatchPostRes{Post: postView(p)}, nil
}

// SetFlags changes site-wide editorial placement and remains administrator-only.
func (c *Posts) SetFlags(ctx context.Context, req *v1.SetFlagsReq) (*v1.SetFlagsRes, error) {
	if err := requireCapability(
		ctx, blogauthz.CapabilityPostFlagsManage, blogauthz.RootScopeID,
		authorization.ResourceFacts{},
	); err != nil {
		return nil, err
	}
	p, err := c.svc.SetFlags(ctx, req.ID, req.Pinned, req.Featured)
	if err != nil {
		return nil, err
	}
	return &v1.SetFlagsRes{Post: postView(p)}, nil
}

// Batch applies a lifecycle action to many of the caller's posts (admin: any).
func (c *Posts) Batch(ctx context.Context, req *v1.BatchReq) (*v1.BatchRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := authorizationService(ctx).ManagePostOwner(ctx); err != nil {
		return nil, mapAuthorizationError(err)
	}
	n, failures, err := c.svc.BatchStatus(ctx, author, isAdmin(ctx), req.IDs, req.Action)
	if err != nil {
		return nil, err
	}
	views := make([]*v1.BatchFailure, 0, len(failures))
	for _, failure := range failures {
		views = append(views, &v1.BatchFailure{ID: failure.ID, Code: failure.Code, Message: failure.Message})
	}
	return &v1.BatchRes{Changed: n, Failures: views}, nil
}

// SetSeries assigns the post to a series at a given order (empty seriesId clears).
func (c *Posts) SetSeries(ctx context.Context, req *v1.SetPostSeriesReq) (*v1.SetPostSeriesRes, error) {
	author, err := subject(ctx)
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
	if err := c.svc.SetPostSeries(ctx, author, isAdmin(ctx), req.ID, req.SeriesID, req.SeriesOrder); err != nil {
		return nil, err
	}
	writeNoContent(ctx)
	return &v1.SetPostSeriesRes{}, nil
}

func (c *Posts) DeletePost(ctx context.Context, req *v1.DeletePostReq) (*v1.DeletePostRes, error) {
	_, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	resource, err := postResource(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(
		ctx, blogauthz.CapabilityPostDelete, blogauthz.PostScopeID(req.ID), resource,
	); err != nil {
		return nil, err
	}
	if err := c.svc.Trash(ctx, resourceOwner(resource), req.ID); err != nil {
		return nil, err
	}
	writeNoContent(ctx)
	return &v1.DeletePostRes{}, nil
}

func (c *Posts) RestorePost(ctx context.Context, req *v1.RestorePostReq) (*v1.RestorePostRes, error) {
	_, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	resource, err := postResource(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(
		ctx, blogauthz.CapabilityPostDelete, blogauthz.PostScopeID(req.ID), resource,
	); err != nil {
		return nil, err
	}
	post, err := c.svc.Restore(ctx, resourceOwner(resource), req.ID)
	if err != nil {
		return nil, err
	}
	return &v1.RestorePostRes{Post: postView(post)}, nil
}

func (c *Posts) PermanentlyDeletePost(ctx context.Context, req *v1.PermanentlyDeletePostReq) (*v1.PermanentlyDeletePostRes, error) {
	_, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	resource, err := postResource(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(
		ctx, blogauthz.CapabilityPostDelete, blogauthz.PostScopeID(req.ID), resource,
	); err != nil {
		return nil, err
	}
	if err := c.svc.PermanentDelete(ctx, resourceOwner(resource), req.ID); err != nil {
		return nil, err
	}
	writeNoContent(ctx)
	return &v1.PermanentlyDeletePostRes{}, nil
}

func (c *Posts) ListRevisions(ctx context.Context, req *v1.ListRevisionsReq) (*v1.ListRevisionsRes, error) {
	_, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	resource, err := postResource(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := requireCapability(
		ctx, blogauthz.CapabilityPostRead, blogauthz.PostScopeID(req.ID), resource,
	); err != nil {
		return nil, err
	}
	revs, err := c.svc.ListRevisions(ctx, resourceOwner(resource), req.ID)
	if err != nil {
		return nil, err
	}
	return &v1.ListRevisionsRes{Items: revisionViews(revs)}, nil
}

func (c *Posts) RestoreRevision(ctx context.Context, req *v1.RestoreRevisionReq) (*v1.RestoreRevisionRes, error) {
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
	p, err := c.svc.RestoreRevision(ctx, resourceOwner(resource), req.ID, req.RevID)
	if err != nil {
		return nil, err
	}
	return &v1.RestoreRevisionRes{Post: postView(p)}, nil
}

// GetMyProfile returns the caller's own author profile (for the settings form).
func (c *Posts) GetMyProfile(ctx context.Context, req *v1.GetMyProfileReq) (*v1.GetMyProfileRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	access, err := authorizationService(ctx).EffectiveAccess(ctx)
	if err != nil {
		return nil, blogerr.AuthorizationUnavailable()
	}
	var authorState *authorAccessState
	for _, grant := range access.Grants {
		if grant.Role == blogauthz.RoleAuthor || grant.Role == blogauthz.RoleAdministrator {
			authorState = &authorAccessState{Role: "author", Status: "active"}
			break
		}
	}
	if authorState == nil {
		applications, listErr := authorizationService(ctx).Runtime().ListApplications(
			ctx,
			authorization.ApplicationListQuery{
				Actor:   authorizationService(ctx).Subject(ctx),
				Subject: authorizationService(ctx).Subject(ctx),
				ScopeID: blogauthz.RootScopeID, State: authorization.ApplicationPending, Limit: 100,
			},
		)
		if listErr != nil {
			return nil, mapAuthorizationError(listErr)
		}
		for _, application := range applications.Applications {
			if application.Role == blogauthz.RoleAuthor {
				authorState = &authorAccessState{Role: "author", Status: "pending"}
				break
			}
		}
	}
	capabilities := make([]string, len(access.Capabilities))
	for index, capability := range access.Capabilities {
		capabilities[index] = string(capability)
	}
	administrator := isAdmin(ctx)
	return &v1.GetMyProfileRes{
		Author:          authorView(author, authorState, c.svc.ResolveAuthor(ctx, author), 0),
		IsAdministrator: administrator, Capabilities: capabilities,
	}, nil
}

func (c *Posts) PutSEO(ctx context.Context, req *v1.PutSEOReq) (*v1.PutSEORes, error) {
	_, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	fields := g.Map{}
	if req.MetaTitle != nil {
		fields["meta_title"] = *req.MetaTitle
	}
	if req.MetaDesc != nil {
		fields["meta_desc"] = *req.MetaDesc
	}
	if req.OgTitle != nil {
		fields["og_title"] = *req.OgTitle
	}
	if req.OgImage != nil {
		fields["og_image"] = *req.OgImage
	}
	if req.CanonicalURL != nil {
		fields["canonical_url"] = *req.CanonicalURL
	}
	if req.Robots != nil {
		fields["robots"] = *req.Robots
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
	seo, err := c.svc.PutSEO(ctx, resourceOwner(resource), req.ID, fields)
	if err != nil {
		return nil, err
	}
	return &v1.PutSEORes{SEO: seoView(seo)}, nil
}
