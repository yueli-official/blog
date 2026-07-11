package controller

import (
	"context"
	"strings"

	"platform/gokit/authjwt"
	v1 "platform/products/blog/api/api/v1"
	"platform/products/blog/api/internal/catalog"
	"platform/products/blog/api/internal/dao"
)

// PublicPosts handles the public browse/detail endpoints (optional login). It
// verifies a bearer token itself when present (not behind authjwt middleware),
// so an author can preview their own drafts.
type PublicPosts struct {
	svc      *catalog.Service
	verifier *authjwt.Verifier
}

func NewPublicPosts(svc *catalog.Service, v *authjwt.Verifier) *PublicPosts {
	return &PublicPosts{svc: svc, verifier: v}
}

func (c *PublicPosts) ListPosts(ctx context.Context, req *v1.ListPostsReq) (*v1.ListPostsRes, error) {
	f := dao.ListFilter{Q: strings.TrimSpace(req.Q), Featured: req.Featured, Pinned: req.Pinned, Sort: req.Sort}
	if req.Taxonomy != "" {
		ids, err := c.svc.PostIDsByTaxonomy(ctx, req.Taxonomy)
		if err != nil {
			return nil, err
		}
		f.IDs = ids // non-nil; empty → no posts in this taxonomy
	}
	items, total, page, size, err := c.svc.List(ctx, f, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	return &v1.ListPostsRes{Items: postViews(items), Total: total, Page: page, Size: size}, nil
}

func (c *PublicPosts) ListTaxonomies(ctx context.Context, req *v1.ListTaxonomiesReq) (*v1.ListTaxonomiesRes, error) {
	items, err := c.svc.ListTaxonomies(ctx, req.Taxonomy)
	if err != nil {
		return nil, err
	}
	return &v1.ListTaxonomiesRes{Items: taxonomyViews(items)}, nil
}

func (c *PublicPosts) GetPost(ctx context.Context, req *v1.GetPostReq) (*v1.GetPostRes, error) {
	viewer := optionalSubject(ctx, c.verifier)
	d, err := c.svc.GetDetail(ctx, viewer, req.Slug)
	if err != nil {
		return nil, err
	}
	return &v1.GetPostRes{
		Post:       postDetailView(d.Post, d.Stats),
		SEO:        seoView(d.SEO),
		Taxonomies: taxonomyViews(d.Taxonomies),
		Series:     seriesView(d.Series),
		Author:     authorView(d.Post.AuthorID, d.Author, c.svc.ResolveAuthor(ctx, d.Post.AuthorID), 0),
		Liked:      d.Liked,
		Bookmarked: d.Bookmarked,
	}, nil
}

// GetAuthor returns a public author profile plus a page of their published posts.
func (c *PublicPosts) GetAuthor(ctx context.Context, req *v1.GetAuthorReq) (*v1.GetAuthorRes, error) {
	ap, err := c.svc.GetAuthorPage(ctx, req.ID, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	return &v1.GetAuthorRes{
		Author: authorView(req.ID, ap.Profile, c.svc.ResolveAuthor(ctx, req.ID), ap.Total),
		Posts:  postViews(ap.Posts),
		Total:  ap.Total, TotalViews: ap.TotalViews, Page: ap.Page, Size: ap.Size,
	}, nil
}

// Siblings returns the published posts adjacent to the given one (prev/next nav).
func (c *PublicPosts) Siblings(ctx context.Context, req *v1.SiblingsReq) (*v1.SiblingsRes, error) {
	prev, next, err := c.svc.Siblings(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	res := &v1.SiblingsRes{}
	if prev != nil {
		res.Prev = postView(prev)
	}
	if next != nil {
		res.Next = postView(next)
	}
	return res, nil
}

// Archive returns one page of lightweight published posts for the date archive page.
func (c *PublicPosts) Archive(ctx context.Context, req *v1.ArchiveReq) (*v1.ArchiveRes, error) {
	items, total, page, size, err := c.svc.Archive(ctx, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	return &v1.ArchiveRes{Items: postViews(items), Total: total, Page: page, Size: size}, nil
}

func (c *PublicPosts) Related(ctx context.Context, req *v1.RelatedPostsReq) (*v1.RelatedPostsRes, error) {
	viewer := optionalSubject(ctx, c.verifier)
	items, err := c.svc.Related(ctx, viewer, req.Slug, req.Limit)
	if err != nil {
		return nil, err
	}
	return &v1.RelatedPostsRes{Items: postViews(items)}, nil
}

func (c *PublicPosts) IncrView(ctx context.Context, req *v1.IncrViewReq) (*v1.IncrViewRes, error) {
	if err := c.svc.IncrementView(ctx, req.Slug); err != nil {
		return nil, err
	}
	return &v1.IncrViewRes{Ok: true}, nil
}
