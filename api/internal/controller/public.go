package controller

import (
	"context"
	"strings"
	"time"

	foundationauth "github.com/yueli-official/foundation/go/auth"
	"github.com/yueli-official/foundation/go/discovery"
	"github.com/yueli-official/foundation/go/traffic"
	v1 "platform/products/blog/api/api/v1"
	"platform/products/blog/api/internal/blogdiscovery"
	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/catalog"
	"platform/products/blog/api/internal/dao"
)

// PublicPosts handles the public browse/detail endpoints (optional login). It
// verifies a bearer token itself when present (not behind Foundation auth middleware),
// so an author can preview their own drafts.
type PublicPosts struct {
	svc       *catalog.Service
	verifier  *foundationauth.Verifier
	discovery *discovery.Module
}

func NewPublicPosts(svc *catalog.Service, v *foundationauth.Verifier, modules ...*discovery.Module) *PublicPosts {
	var module *discovery.Module
	if len(modules) > 0 {
		module = modules[0]
	}
	return &PublicPosts{svc: svc, verifier: v, discovery: module}
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
	items, total, page, size, err := c.svc.ListTaxonomiesPage(ctx, req.Taxonomy, req.Q, req.Sort, req.Direction, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	return &v1.ListTaxonomiesRes{Items: taxonomyViews(items), Total: total, Page: page, Size: size}, nil
}

func (c *PublicPosts) GetPost(ctx context.Context, req *v1.GetPostReq) (*v1.GetPostRes, error) {
	viewer := optionalSubject(ctx, c.verifier)
	d, err := c.svc.GetDetail(ctx, viewer, req.Slug)
	if err != nil {
		return nil, err
	}
	resolvedAuthor := c.svc.ResolveAuthor(ctx, d.Post.AuthorID)
	response := &v1.GetPostRes{
		Post:       postDetailView(d.Post, d.Stats),
		SEO:        seoView(d.SEO),
		Taxonomies: taxonomyViews(d.Taxonomies),
		Series:     seriesView(d.Series),
		Author:     authorView(d.Post.AuthorID, nil, resolvedAuthor, 0),
		Liked:      d.Liked,
		Bookmarked: d.Bookmarked,
	}
	if c.discovery != nil && d.Post.Status == "published" {
		projection, err := blogdiscovery.ProjectPost(c.discovery, d.Post, d.SEO, resolvedAuthor.DisplayName)
		if err != nil {
			return nil, err
		}
		response.Discovery = &projection
	}
	return response, nil
}

// GetAuthor returns a public author profile plus a page of their published posts.
func (c *PublicPosts) GetAuthor(ctx context.Context, req *v1.GetAuthorReq) (*v1.GetAuthorRes, error) {
	ap, err := c.svc.GetAuthorPage(ctx, req.ID, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	return &v1.GetAuthorRes{
		Author: authorView(req.ID, nil, c.svc.ResolveAuthor(ctx, req.ID), ap.Total),
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

func (c *PublicPosts) RecordView(ctx context.Context, req *v1.RecordViewReq) (*v1.RecordViewRes, error) {
	occurredAt, err := time.Parse(time.RFC3339Nano, req.OccurredAt)
	if err != nil {
		return nil, blogerr.InvalidInput("occurredAt must be an RFC3339 timestamp")
	}
	ip, ua := clientMeta(ctx)
	subject := optionalSubject(ctx, c.verifier)
	seed := anonymousVisitorSeed(ip, ua)
	if subject != "" {
		seed = []byte("subject\x00" + subject)
	}
	result, err := c.svc.RecordView(ctx, req.Slug, catalog.ViewInput{
		EventID: req.EventID, OccurredAt: occurredAt,
		Class: classifyVisit(ua), VisitorSeed: seed,
	})
	if err != nil {
		if traffic.IsKind(err, traffic.ErrorInvalidInput) || traffic.IsKind(err, traffic.ErrorConflict) {
			return nil, blogerr.InvalidInput(err.Error())
		}
		return nil, err
	}
	return &v1.RecordViewRes{
		Ok: true, Counted: result.Counted, Replay: result.Replay,
		ViewCount: result.ResourceTotals.Views,
	}, nil
}

func anonymousVisitorSeed(ip, userAgent string) []byte {
	ip = strings.TrimSpace(ip)
	userAgent = strings.TrimSpace(userAgent)
	if ip == "" && userAgent == "" {
		return nil
	}
	return []byte("network\x00" + ip + "\x00" + userAgent)
}

func classifyVisit(userAgent string) traffic.VisitClass {
	value := strings.ToLower(userAgent)
	for _, marker := range []string{
		"bot", "crawler", "spider", "slurp", "headless", "monitoring",
		"facebookexternalhit", "twitterbot", "preview",
	} {
		if strings.Contains(value, marker) {
			return traffic.VisitBot
		}
	}
	if strings.TrimSpace(value) == "" {
		return traffic.VisitUnknown
	}
	return traffic.VisitHuman
}
