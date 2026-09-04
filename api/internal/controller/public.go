package controller

import (
	"context"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/appconfig"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/blogdiscovery"
	"github.com/yueli-official/blog/api/internal/blogerr"
	"github.com/yueli-official/blog/api/internal/catalog"
	"github.com/yueli-official/blog/api/internal/dao"
	foundationauth "github.com/yueli-official/foundation/go/auth"
	"github.com/yueli-official/foundation/go/discovery"
	"github.com/yueli-official/foundation/go/traffic"
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
	authorizedCtx, viewer := optionalAuthenticatedContext(ctx, c.verifier)
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
		CanEdit:    canEditPost(authorizedCtx, d.Post.ID),
	}
	if c.discovery != nil && d.Post.Status == "published" {
		projection, err := blogdiscovery.ProjectPost(c.discovery, d.Post, d.SEO, resolvedAuthor.DisplayName)
		if err != nil {
			g.Log().Warningf(ctx, "discovery projection omitted for post %s: %v", d.Post.ID, err)
		} else {
			response.Discovery = &projection
		}
	}
	return response, nil
}

func canEditPost(ctx context.Context, postID string) bool {
	service := authorizationService(ctx)
	if service == nil || service.Subject(ctx).ID == "" {
		return false
	}
	resource, err := service.PostResource(ctx, postID)
	if err != nil {
		return false
	}
	decision, err := service.Decide(
		ctx,
		blogauthz.CapabilityPostUpdate,
		blogauthz.PostScopeID(postID),
		resource,
	)
	return err == nil && decision.Allowed
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
	location, err := time.LoadLocation(appconfig.TrafficTimeZone(ctx))
	if err != nil {
		return nil, err
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
		Day: occurredAt.In(location).Format(time.DateOnly), Source: normalizeTrafficSource(req.Source),
	})
	if err != nil {
		if traffic.IsKind(err, traffic.ErrorInvalidInput) || traffic.IsKind(err, traffic.ErrorConflict) {
			return nil, blogerr.InvalidInput("view_event_invalid")
		}
		return nil, err
	}
	return &v1.RecordViewRes{
		Ok: true, Counted: result.Counted, Replay: result.Replay,
		ViewCount: result.ResourceTotals.Views,
	}, nil
}

func normalizeTrafficSource(value string) string {
	value = strings.TrimSuffix(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "www."), ".")
	if value == "" || value == "direct" {
		return "direct"
	}
	if value == "internal" {
		return value
	}
	if len(value) > 200 || !strings.Contains(value, ".") || strings.Contains(value, "..") {
		return "direct"
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '.' && character != '-' {
			return "direct"
		}
	}
	return value
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
