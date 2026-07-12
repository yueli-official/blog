package controller

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"platform/gokit/authjwt"
	v1 "platform/products/blog/api/api/v1"
	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/catalog"
	"platform/products/blog/api/internal/identityclient"
	"platform/products/blog/api/internal/model"
)

// Markdown-stripping for auto-excerpt: image → drop, link → its text, list
// markers at line start → drop, inline/heading/emphasis tokens → drop.
var (
	mdImg    = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	mdLink   = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	mdList   = regexp.MustCompile("(?m)^[ \t]*([-+*]|\\d+\\.)[ \t]+")
	mdInline = regexp.MustCompile("[#>`*_~]")
)

// deriveExcerpt builds a short plain-text preview from markdown content, used
// when a post has no author-set excerpt — so list cards, SEO/OG descriptions
// and the RSS feed still get a sensible summary. Minimal stripping, not a parser.
func deriveExcerpt(md string, max int) string {
	s := mdImg.ReplaceAllString(md, "")
	s = mdLink.ReplaceAllString(s, "$1")
	s = mdList.ReplaceAllString(s, "")
	s = mdInline.ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(s), " ") // collapse all whitespace to single spaces
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return strings.TrimSpace(string(r[:max])) + "…"
}

// subject extracts the authenticated subject (JWT group), or a forbidden error.
func subject(ctx context.Context) (string, error) {
	p, ok := authjwt.From(ctx)
	if !ok {
		return "", blogerr.Forbidden()
	}
	return p.Subject, nil
}

// isAdmin reports whether the caller is a blog OWNER — the site's top role, held
// by whoever the catalog lists in blog.operatorSubs (their identity `sub`). This is
// the blog's OWN authorization: it deliberately does NOT read the IdP's global
// "roles" claim, because the IdP is shared across the site cluster and a global
// admin would implicitly own every site. Authentication is centralized (the IdP
// issues the verified sub); authorization is per-site (this list). author/
// contributor roles live in author_profiles; owner is config so it can't be
// edited away in the UI and needs no bootstrap row.
func isAdmin(ctx context.Context) bool {
	p, ok := authjwt.From(ctx)
	if !ok {
		return false
	}
	operators, err := g.Cfg().Get(ctx, "blog.operatorSubs")
	if err != nil {
		return false
	}
	return slices.Contains(operators.Strings(), p.Subject)
}

// requireAdmin returns a 403 unless the caller is a configured site operator.
func requireAdmin(ctx context.Context) error {
	if !isAdmin(ctx) {
		return blogerr.Forbidden()
	}
	return nil
}

// bearerOf returns the raw bearer token on the request (to forward to the asset
// service for cover uploads; author == asset owner).
func bearerOf(ctx context.Context) string {
	r := ghttp.RequestFromCtx(ctx)
	if r == nil {
		return ""
	}
	return stripBearer(r.Request.Header.Get("Authorization"))
}

// optionalSubject verifies the bearer token if present, returning the subject or
// "" (anonymous). Used by the public browse/detail endpoints (optional login).
func optionalSubject(ctx context.Context, v *authjwt.Verifier) string {
	raw := bearerOf(ctx)
	if raw == "" || v == nil {
		return ""
	}
	p, err := v.Verify(ctx, raw)
	if err != nil {
		return ""
	}
	return p.Subject
}

func stripBearer(h string) string {
	const p = "bearer "
	if len(h) < len(p) || !strings.EqualFold(h[:len(p)], p) {
		return ""
	}
	return strings.TrimSpace(h[len(p):])
}

func postView(p *model.Post) *v1.PostView {
	v := &v1.PostView{
		ID: p.ID, AuthorID: p.AuthorID, Title: p.Title, Slug: p.Slug,
		Content: p.Content, Excerpt: p.Excerpt, CoverAssetID: p.CoverAssetID,
		CoverURL: p.CoverURL, Status: string(p.Status), CommentStatus: p.CommentStatus,
		ViewCount: p.ViewCount, // joined from post_stats by List/Archive (0 elsewhere)
		Pinned:    p.Pinned, Featured: p.Featured,
		SeriesID: p.SeriesID, SeriesOrder: p.SeriesOrder,
		Taxonomies: taxonomyViews(p.Taxonomies),
	}
	if v.Excerpt == "" {
		v.Excerpt = deriveExcerpt(p.Content, 150)
	}
	if p.PublishedAt != nil {
		v.PublishedAt = p.PublishedAt.Time.UTC().Format(time.RFC3339)
	}
	if p.CreatedAt != nil {
		v.CreatedAt = p.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	if p.UpdatedAt != nil {
		v.UpdatedAt = p.UpdatedAt.Time.UTC().Format(time.RFC3339)
	}
	return v
}

func postViews(ps []*model.Post) []*v1.PostView {
	out := make([]*v1.PostView, 0, len(ps))
	for _, p := range ps {
		out = append(out, postView(p))
	}
	return out
}

func postDetailView(p *model.Post, st *model.Stats) *v1.PostView {
	v := postView(p)
	if st != nil {
		v.ViewCount = st.ViewCount
	}
	return v
}

func seoView(s *model.SEO) *v1.SEOView {
	if s == nil {
		return nil
	}
	return &v1.SEOView{
		MetaTitle: s.MetaTitle, MetaDesc: s.MetaDesc, OgTitle: s.OgTitle,
		OgImage: s.OgImage, CanonicalURL: s.CanonicalURL, Robots: s.Robots,
	}
}

func revisionView(r *model.Revision) *v1.RevisionView {
	v := &v1.RevisionView{ID: r.ID, Title: r.Title, RevNote: r.RevNote}
	if r.CreatedAt != nil {
		v.CreatedAt = r.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	return v
}

func revisionViews(rs []*model.Revision) []*v1.RevisionView {
	out := make([]*v1.RevisionView, 0, len(rs))
	for _, r := range rs {
		out = append(out, revisionView(r))
	}
	return out
}

func taxonomyView(t *model.Taxonomy) *v1.TaxonomyView {
	return &v1.TaxonomyView{
		ID: t.ID, Taxonomy: t.Taxonomy, Name: t.Name, Slug: t.Slug,
		Description: t.Description, ParentID: t.ParentID, PostCount: t.PostCount,
	}
}

// authorView projects an author: domain-specific role/status/joined-date come
// from the local author_profiles row (p, may be nil), while the display fields
// (name/avatar/cover/bio/social) come from the identity public profile (idp, the
// single source of truth — empty when unresolved, so the UI falls back to the
// id). postCount is set on the public author page, 0 (omitted) elsewhere.
func authorView(id string, p *model.AuthorProfile, idp identityclient.Profile, postCount int) *v1.AuthorView {
	v := &v1.AuthorView{
		ID:          id,
		PostCount:   postCount,
		DisplayName: idp.DisplayName,
		Bio:         idp.Bio,
		AvatarURL:   idp.AvatarURL,
		BannerURL:   idp.CoverURL,
		SocialLinks: socialLinksView(idp.SocialLinks),
	}
	if p != nil {
		v.Role = p.Role
		v.Status = p.Status
		if p.CreatedAt != nil {
			v.CreatedAt = p.CreatedAt.Time.UTC().Format(time.RFC3339)
		}
	}
	if v.Role == "" {
		v.Role = "contributor" // conservative default: 主笔(author) is the elevated, owner-granted role
	}
	return v
}

func socialLinksView(in []identityclient.SocialLink) []v1.SocialLink {
	out := make([]v1.SocialLink, 0, len(in))
	for _, l := range in {
		out = append(out, v1.SocialLink{Label: l.Label, URL: l.URL})
	}
	return out
}

// adminAuthorView projects one author roster row; displayName is resolved from
// the identity profile (the local roster only holds role/status/post counts).
func adminAuthorView(r *model.AuthorRoster, displayName string) *v1.AdminAuthorView {
	role := r.Role
	if role == "" {
		role = "contributor"
	}
	status := r.Status
	if status == "" {
		status = "active"
	}
	return &v1.AdminAuthorView{ID: r.AuthorID, DisplayName: displayName, Role: role, Status: status, PostCount: r.PostCount}
}

func adminAuthorViews(rs []*model.AuthorRoster, idps map[string]identityclient.Profile) []*v1.AdminAuthorView {
	out := make([]*v1.AdminAuthorView, 0, len(rs))
	for _, r := range rs {
		out = append(out, adminAuthorView(r, idps[r.AuthorID].DisplayName))
	}
	return out
}

func seriesView(s *model.Series) *v1.SeriesView {
	if s == nil {
		return nil
	}
	return &v1.SeriesView{
		ID: s.ID, Slug: s.Slug, Name: s.Name, Description: s.Description,
		CoverURL: s.CoverURL, AuthorID: s.AuthorID, PostCount: s.PostCount,
		RecentPosts: postViews(s.Recent),
	}
}

func seriesViews(ss []*model.Series) []*v1.SeriesView {
	out := make([]*v1.SeriesView, 0, len(ss))
	for _, s := range ss {
		out = append(out, seriesView(s))
	}
	return out
}

func taxonomyViews(ts []*model.Taxonomy) []*v1.TaxonomyView {
	out := make([]*v1.TaxonomyView, 0, len(ts))
	for _, t := range ts {
		out = append(out, taxonomyView(t))
	}
	return out
}

// clientMeta returns the request's client IP + user-agent (for spam triage).
func clientMeta(ctx context.Context) (ip, ua string) {
	r := ghttp.RequestFromCtx(ctx)
	if r == nil {
		return "", ""
	}
	return r.GetClientIp(), r.Request.UserAgent()
}

// commentDisplayName is the public label for a comment author: the anonymous
// name, else a short member tag derived from the sub, else "匿名". The raw sub
// is never sent to the public list.
func commentDisplayName(c *model.Comment) string {
	if c.AuthorName != "" {
		return c.AuthorName
	}
	if c.UserID != "" {
		if len(c.UserID) > 8 {
			return c.UserID[:8]
		}
		return c.UserID
	}
	return "匿名"
}

func commentView(c *model.Comment) *v1.CommentView {
	v := &v1.CommentView{
		ID: c.ID, ParentID: c.ParentID, AuthorName: commentDisplayName(c),
		IsMember: c.UserID != "", Content: c.Content,
	}
	if c.CreatedAt != nil {
		v.CreatedAt = c.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	return v
}

func commentThreadViews(ts []*catalog.CommentThread) []*v1.CommentView {
	out := make([]*v1.CommentView, 0, len(ts))
	for _, t := range ts {
		cv := commentView(t.Comment)
		for _, r := range t.Replies {
			cv.Replies = append(cv.Replies, commentView(r))
		}
		out = append(out, cv)
	}
	return out
}

func commentAdminView(a *catalog.AdminComment) *v1.CommentAdminView {
	c := a.Comment
	v := &v1.CommentAdminView{
		ID: c.ID, PostID: c.PostID, PostTitle: a.PostTitle, PostSlug: a.PostSlug,
		ParentID: c.ParentID, AuthorName: commentDisplayName(c), AuthorEmail: c.AuthorEmail,
		UserID: c.UserID, Content: c.Content, Status: int(c.Status), IP: c.IP,
	}
	if c.CreatedAt != nil {
		v.CreatedAt = c.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	return v
}

// commentAdminViewBare projects a bare comment (no joined post head) — used by the
// moderation PATCH response, where the caller already knows the post.
func commentAdminViewBare(c *model.Comment) *v1.CommentAdminView {
	return commentAdminView(&catalog.AdminComment{Comment: c})
}

func commentAdminViews(as []*catalog.AdminComment) []*v1.CommentAdminView {
	out := make([]*v1.CommentAdminView, 0, len(as))
	for _, a := range as {
		out = append(out, commentAdminView(a))
	}
	return out
}
