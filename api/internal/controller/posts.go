package controller

import (
	"context"
	"slices"

	"github.com/gogf/gf/v2/frame/g"

	v1 "platform/products/blog/api/api/v1"
	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/catalog"
)

// Posts handles the author (JWT) post-management endpoints.
type Posts struct{ svc *catalog.Service }

func NewPosts(svc *catalog.Service) *Posts { return &Posts{svc: svc} }

func (c *Posts) ListMine(ctx context.Context, req *v1.ListMineReq) (*v1.ListMineRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	// scope: a non-admin only ever sees their own posts; an admin may switch to a
	// specific author (req.AuthorID) or all authors (req.All).
	scope := author
	if isAdmin(ctx) {
		if req.AuthorID != "" {
			scope = req.AuthorID
		} else if req.All {
			scope = ""
		}
	}
	items, total, page, size, err := c.svc.ListManage(ctx, scope, req.Status, req.Q, req.TaxonomyIds, req.Pinned, req.Featured, req.Sort, req.Direction, req.Page, req.Size)
	if err != nil {
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
	if err := c.svc.RequireAuthor(ctx, author, isAdmin(ctx)); err != nil {
		return nil, err
	}
	p, err := c.svc.Create(ctx, author, req.Title, req.Content, req.Excerpt)
	if err != nil {
		return nil, err
	}
	return &v1.CreatePostRes{Post: postView(p)}, nil
}

func (c *Posts) PatchPost(ctx context.Context, req *v1.PatchPostReq) (*v1.PatchPostRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.svc.RequireAuthor(ctx, author, isAdmin(ctx)); err != nil {
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
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	p, err := c.svc.Patch(ctx, author, req.ID, fields)
	if err != nil {
		return nil, err
	}
	return &v1.PatchPostRes{Post: postView(p)}, nil
}

// SetFlags sets a post's editorial flags pinned/featured (superadmin only).
func (c *Posts) SetFlags(ctx context.Context, req *v1.SetFlagsReq) (*v1.SetFlagsRes, error) {
	if err := requireAdmin(ctx); err != nil {
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
	if err := c.svc.SetPostSeries(ctx, author, isAdmin(ctx), req.ID, req.SeriesID, req.SeriesOrder); err != nil {
		return nil, err
	}
	return &v1.SetPostSeriesRes{Updated: true}, nil
}

func (c *Posts) DeletePost(ctx context.Context, req *v1.DeletePostReq) (*v1.DeletePostRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.svc.Delete(ctx, author, req.ID); err != nil {
		return nil, err
	}
	return &v1.DeletePostRes{Deleted: true}, nil
}

func (c *Posts) ListRevisions(ctx context.Context, req *v1.ListRevisionsReq) (*v1.ListRevisionsRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	revs, err := c.svc.ListRevisions(ctx, author, req.ID)
	if err != nil {
		return nil, err
	}
	return &v1.ListRevisionsRes{Items: revisionViews(revs)}, nil
}

func (c *Posts) RestoreRevision(ctx context.Context, req *v1.RestoreRevisionReq) (*v1.RestoreRevisionRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	p, err := c.svc.RestoreRevision(ctx, author, req.ID, req.RevID)
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
	prof, err := c.svc.GetMyProfile(ctx, author)
	if err != nil {
		return nil, err
	}
	return &v1.GetMyProfileRes{Author: authorView(author, prof, c.svc.ResolveAuthor(ctx, author), 0), IsOwner: isAdmin(ctx)}, nil
}

// RequestAuthor records the caller's authorship request (pending → admin approves).
func (c *Posts) RequestAuthor(ctx context.Context, req *v1.RequestAuthorReq) (*v1.RequestAuthorRes, error) {
	author, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	prof, err := c.svc.RequestAuthor(ctx, author)
	if err != nil {
		return nil, err
	}
	return &v1.RequestAuthorRes{Author: authorView(author, prof, c.svc.ResolveAuthor(ctx, author), 0)}, nil
}

// AdminApproveAuthor approves a pending author request (admin only).
func (c *Posts) AdminApproveAuthor(ctx context.Context, req *v1.AdminApproveAuthorReq) (*v1.AdminApproveAuthorRes, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	prof, err := c.svc.AdminApproveAuthor(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &v1.AdminApproveAuthorRes{Author: authorView(req.ID, prof, c.svc.ResolveAuthor(ctx, req.ID), 0)}, nil
}

// AdminRemoveAuthor rejects a request / revokes authorship (admin only). An admin
// cannot remove themselves (footgun guard).
func (c *Posts) AdminRemoveAuthor(ctx context.Context, req *v1.AdminRemoveAuthorReq) (*v1.AdminRemoveAuthorRes, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if me, _ := subject(ctx); me == req.ID {
		return nil, blogerr.InvalidState("不能移除你自己")
	}
	if err := c.svc.AdminRemoveAuthor(ctx, req.ID); err != nil {
		return nil, err
	}
	return &v1.AdminRemoveAuthorRes{Removed: true}, nil
}

// AdminListAuthors returns the author roster + roles + post counts (admin only).
func (c *Posts) AdminListAuthors(ctx context.Context, req *v1.AdminListAuthorsReq) (*v1.AdminListAuthorsRes, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	rows, err := c.svc.AdminListAuthors(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.AuthorID)
	}
	views := adminAuthorViews(rows, c.svc.ResolveAuthors(ctx, ids))
	// Flag site operators so the UI shows 站长 and hides role/remove for them.
	operators := g.Cfg().MustGet(ctx, "blog.operatorSubs").Strings()
	for _, v := range views {
		v.Owner = slices.Contains(operators, v.ID)
	}
	return &v1.AdminListAuthorsRes{Authors: views}, nil
}

// AdminSetAuthorRole sets an author's 主笔/客座 role (admin only).
func (c *Posts) AdminSetAuthorRole(ctx context.Context, req *v1.AdminSetAuthorRoleReq) (*v1.AdminSetAuthorRoleRes, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if me, _ := subject(ctx); me == req.ID {
		return nil, blogerr.InvalidState("不能修改你自己的署名")
	}
	prof, err := c.svc.AdminSetAuthorRole(ctx, req.ID, req.Role)
	if err != nil {
		return nil, err
	}
	return &v1.AdminSetAuthorRoleRes{Author: authorView(req.ID, prof, c.svc.ResolveAuthor(ctx, req.ID), 0)}, nil
}

func (c *Posts) PutSEO(ctx context.Context, req *v1.PutSEOReq) (*v1.PutSEORes, error) {
	author, err := subject(ctx)
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
	seo, err := c.svc.PutSEO(ctx, author, req.ID, fields)
	if err != nil {
		return nil, err
	}
	return &v1.PutSEORes{SEO: seoView(seo)}, nil
}
