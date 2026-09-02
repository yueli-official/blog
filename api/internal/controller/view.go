package controller

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/blogerr"
	"github.com/yueli-official/blog/api/internal/catalog"
	"github.com/yueli-official/blog/api/internal/coverurl"
	"github.com/yueli-official/blog/api/internal/identityclient"
	"github.com/yueli-official/blog/api/internal/model"
	foundationauth "github.com/yueli-official/foundation/go/auth"
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
	p, ok := foundationauth.FromContext(ctx)
	if !ok || p == nil || !isUserPrincipal(p) {
		return "", blogerr.Forbidden()
	}
	return p.Subject, nil
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
func optionalSubject(ctx context.Context, v *foundationauth.Verifier) string {
	_, subject := optionalAuthenticatedContext(ctx, v)
	return subject
}

// optionalAuthenticatedContext verifies an optional public-route bearer token
// and returns a child context that the authorization Module can evaluate.
func optionalAuthenticatedContext(ctx context.Context, v *foundationauth.Verifier) (context.Context, string) {
	raw := bearerOf(ctx)
	if raw == "" || v == nil {
		return ctx, ""
	}
	p, err := v.Verify(ctx, raw)
	if err != nil || !isUserPrincipal(p) {
		return ctx, ""
	}
	return foundationauth.NewContext(ctx, p), p.Subject
}

func isUserPrincipal(principal *foundationauth.Principal) bool {
	if principal == nil || strings.TrimSpace(principal.Subject) == "" {
		return false
	}
	kind, _ := principal.Claim("subject_kind")
	return kind == "user"
}

func stripBearer(h string) string {
	const p = "bearer "
	if len(h) < len(p) || !strings.EqualFold(h[:len(p)], p) {
		return ""
	}
	return strings.TrimSpace(h[len(p):])
}

func postView(p *model.Post) *v1.PostView {
	coverURL := coverurl.NormalizeManaged(p.CoverAssetID, p.CoverURL)
	v := &v1.PostView{
		ID: p.ID, AuthorID: p.AuthorID, Title: p.Title, Slug: p.Slug,
		Content: p.Content, Excerpt: p.Excerpt, CoverAssetID: p.CoverAssetID,
		CoverURL: coverURL, Status: string(p.Status), CommentStatus: p.CommentStatus,
		ViewCount: p.ViewCount,
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
	if p.DeletedAt != nil {
		v.DeletedAt = p.DeletedAt.Time.UTC().Format(time.RFC3339)
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

type authorAccessState struct {
	Role   string
	Status string
}

// authorView projects Identity-owned public display fields together with an
// optional request-local Authorization state. No author role is read from a
// Blog profile table.
func authorView(id string, state *authorAccessState, idp identityclient.PublicUser, postCount int) *v1.AuthorView {
	v := &v1.AuthorView{
		ID:          id,
		Handle:      idp.Handle,
		PostCount:   postCount,
		DisplayName: idp.DisplayName,
		Bio:         idp.Bio,
		AvatarURL:   publicMediaURL(idp.Avatar, "thumbnail"),
		BannerURL:   publicMediaURL(idp.Cover, "cover"),
		SocialLinks: socialLinksView(idp.SocialLinks),
	}
	if state != nil {
		v.Role = state.Role
		v.Status = state.Status
	}
	return v
}

func publicMediaURL(reference *identityclient.MediaRef, rendition string) string {
	if reference == nil || reference.MediaKey == "" {
		return ""
	}
	return "/media/" + reference.MediaKey + "?format=webp&name=" + rendition + "&v=1"
}

func socialLinksView(in []identityclient.SocialLink) []v1.SocialLink {
	out := make([]v1.SocialLink, 0, len(in))
	for _, l := range in {
		out = append(out, v1.SocialLink{Label: l.Label, URL: l.URL})
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

func commentPresentation(c *model.Comment, profiles map[string]identityclient.PublicUser) (string, string) {
	name := commentDisplayName(c)
	if c.UserID == "" {
		return name, ""
	}
	profile, ok := profiles[c.UserID]
	if !ok {
		return name, ""
	}
	if profile.DisplayName != "" {
		name = profile.DisplayName
	}
	return name, publicMediaURL(profile.Avatar, "thumbnail")
}

func commentViewWithProfiles(c *model.Comment, profiles map[string]identityclient.PublicUser) *v1.CommentView {
	name, avatarURL := commentPresentation(c, profiles)
	v := &v1.CommentView{
		ID: c.ID, ParentID: c.ParentID, AuthorName: name, AvatarURL: avatarURL,
		IsAnonymous: c.UserID == "", Content: c.Content,
	}
	if c.CreatedAt != nil {
		v.CreatedAt = c.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	return v
}

func commentView(c *model.Comment) *v1.CommentView {
	return commentViewWithProfiles(c, nil)
}

func commentThreadUserIDs(ts []*catalog.CommentThread) []string {
	ids := make([]string, 0, len(ts))
	for _, thread := range ts {
		ids = append(ids, thread.Comment.UserID)
		for _, reply := range thread.Replies {
			ids = append(ids, reply.UserID)
		}
	}
	return ids
}

func commentThreadViews(ts []*catalog.CommentThread, profiles map[string]identityclient.PublicUser) []*v1.CommentView {
	out := make([]*v1.CommentView, 0, len(ts))
	for _, t := range ts {
		cv := commentViewWithProfiles(t.Comment, profiles)
		for _, r := range t.Replies {
			cv.Replies = append(cv.Replies, commentViewWithProfiles(r, profiles))
		}
		out = append(out, cv)
	}
	return out
}

func commentAdminView(a *catalog.AdminComment, profiles map[string]identityclient.PublicUser) *v1.CommentAdminView {
	c := a.Comment
	name, avatarURL := commentPresentation(c, profiles)
	v := &v1.CommentAdminView{
		ID: c.ID, PostID: c.PostID, PostTitle: a.PostTitle, PostSlug: a.PostSlug,
		ParentID: c.ParentID, AuthorName: name, AvatarURL: avatarURL, AuthorEmail: c.AuthorEmail,
		UserID: c.UserID, Content: c.Content, Status: int(c.Status), IP: c.IP,
	}
	if c.CreatedAt != nil {
		v.CreatedAt = c.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	return v
}

// commentAdminViewBare projects a bare comment (no joined post head) — used by the
// moderation PATCH response, where the caller already knows the post.
func commentAdminViewBare(c *model.Comment, profiles map[string]identityclient.PublicUser) *v1.CommentAdminView {
	return commentAdminView(&catalog.AdminComment{Comment: c}, profiles)
}

func adminCommentUserIDs(as []*catalog.AdminComment) []string {
	ids := make([]string, 0, len(as))
	for _, comment := range as {
		ids = append(ids, comment.Comment.UserID)
	}
	return ids
}

func commentAdminViews(as []*catalog.AdminComment, profiles map[string]identityclient.PublicUser) []*v1.CommentAdminView {
	out := make([]*v1.CommentAdminView, 0, len(as))
	for _, a := range as {
		out = append(out, commentAdminView(a, profiles))
	}
	return out
}
