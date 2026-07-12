// Package catalog is the blog site's core logic: post CRUD, the draft/publish
// lifecycle, and (later tasks) taxonomy, covers, stats, reactions.
package catalog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"unicode"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	"platform/gokit/mail"
	"platform/products/blog/api/internal/blogclient"
	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/dao"
	"platform/products/blog/api/internal/identityclient"
	"platform/products/blog/api/internal/model"
)

// Service owns the catalog logic.
type Service struct {
	dao           *dao.PG
	asset         blogclient.Client
	coverCategory string
	mailer        mail.Sender
	siteURL       string // base for newsletter confirm/unsubscribe links
	spam          SpamPolicy
	identity      identityclient.Client // public display data (name/avatar/cover/bio/social)
}

func New(d *dao.PG, asset blogclient.Client, coverCategory string, mailer mail.Sender, siteURL string, spam SpamPolicy) *Service {
	return &Service{dao: d, asset: asset, coverCategory: coverCategory, mailer: mailer, siteURL: siteURL, spam: spam}
}

// SetIdentityClient wires the identity public-profiles client (display data
// source for author pages / bylines). Optional in tests (a nil client resolves
// to empty profiles, and the UI falls back to the bare id).
func (s *Service) SetIdentityClient(c identityclient.Client) { s.identity = c }

// ResolveAuthor returns the public display profile for one author id (empty when
// no client is wired or the id is unknown).
func (s *Service) ResolveAuthor(ctx context.Context, id string) identityclient.Profile {
	if s.identity == nil {
		return identityclient.Profile{ID: id}
	}
	return s.identity.Get(ctx, id)
}

// ResolveAuthors batch-resolves display profiles by id (empty map when unwired).
func (s *Service) ResolveAuthors(ctx context.Context, ids []string) map[string]identityclient.Profile {
	if s.identity == nil {
		return map[string]identityclient.Profile{}
	}
	return s.identity.GetMany(ctx, ids)
}

// Create inserts a draft post with a unique slug derived from the title.
func (s *Service) Create(ctx context.Context, author, title, content, excerpt string) (*model.Post, error) {
	p := &model.Post{
		AuthorID: author, Title: title, Content: content, Excerpt: excerpt,
		Status: model.StatusDraft, CommentStatus: 1, // comments open by default
	}
	if err := s.insertWithSlug(ctx, p, slugify(title)); err != nil {
		return nil, err
	}
	_ = s.dao.EnsureStats(ctx, p.ID) // best-effort; view increment also ensures
	return s.dao.GetByID(ctx, p.ID)
}

// insertWithSlug tries base, base-2..4, then base-<rand> until the slug is free.
func (s *Service) insertWithSlug(ctx context.Context, p *model.Post, base string) error {
	if base == "" {
		base = "post"
	}
	candidates := []string{base, base + "-2", base + "-3", base + "-4"}
	for i := 0; i < 8; i++ {
		var slug string
		if i < len(candidates) {
			slug = candidates[i]
		} else {
			slug = base + "-" + randHex(3)
		}
		p.Slug = slug
		err := s.dao.Insert(ctx, p)
		if err == nil {
			return nil
		}
		if err != dao.ErrSlugTaken {
			return err
		}
	}
	return blogerr.SlugTaken(base)
}

// Get returns a post visible to the viewer (published, or any status owned by
// the viewer).
func (s *Service) Get(ctx context.Context, viewer, id string) (*model.Post, error) {
	p, err := s.dao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !visible(p, viewer) {
		return nil, blogerr.NotFound(id)
	}
	return p, nil
}

// GetBySlug is Get keyed by slug (public detail page).
func (s *Service) GetBySlug(ctx context.Context, viewer, slug string) (*model.Post, error) {
	p, err := s.dao.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if !visible(p, viewer) {
		return nil, blogerr.NotFound(slug)
	}
	return p, nil
}

func visible(p *model.Post, viewer string) bool {
	if p == nil {
		return false
	}
	return p.Status == model.StatusPublished || (viewer != "" && p.AuthorID == viewer)
}

// List returns published posts matching the filter, plus the total and the
// normalized page/size.
func (s *Service) List(ctx context.Context, f dao.ListFilter, page, size int) ([]*model.Post, int, int, int, error) {
	page, size = norm(page, size)
	items, total, err := s.dao.List(ctx, f, size, (page-1)*size)
	return items, total, page, size, err
}

// ListMine returns the author's posts (manage console), filtered by status/query.
// ListManage powers the manage console list. authorID "" = all authors (admin
// view); taxonomyIDs (AND) narrows by category/tag.
func (s *Service) ListManage(ctx context.Context, authorID, status, q string, taxonomyIDs []string, pinned, featured bool, sort, direction string, page, size int) ([]*model.Post, int, int, int, error) {
	page, size = norm(page, size)
	items, total, err := s.dao.ListManage(ctx, authorID, status, q, taxonomyIDs, pinned, featured, sort, direction, size, (page-1)*size)
	return items, total, page, size, err
}

// StatusCounts returns per-status post counts for the manage filter tabs.
// authorID "" = all authors (admin view).
func (s *Service) StatusCounts(ctx context.Context, authorID string) (map[string]int, error) {
	return s.dao.StatusCounts(ctx, authorID)
}

// MyTotalViews sums view_count across all the author's posts (dashboard stat).
func (s *Service) MyTotalViews(ctx context.Context, author string) (int64, error) {
	return s.dao.TotalViewsByAuthor(ctx, author)
}

// Patch updates mutable fields for the author's post. Promoting to published
// enforces the publish constraint (non-empty title + content) and stamps
// published_at on first publish.
func (s *Service) Patch(ctx context.Context, author, id string, fields g.Map) (*model.Post, error) {
	cur, err := s.dao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur == nil || cur.AuthorID != author {
		return nil, blogerr.NotFound(id)
	}
	// a custom slug is normalized + must stay unique
	slugStr := ""
	if v, ok := fields["slug"].(string); ok {
		slugStr = slugify(v)
		if slugStr == "" {
			return nil, blogerr.InvalidInput("slug produces an empty value")
		}
		fields["slug"] = slugStr
	}
	firstPublish := false
	if st, ok := fields["status"].(string); ok && st == string(model.StatusPublished) {
		title := cur.Title
		if v, ok := fields["title"].(string); ok {
			title = v
		}
		content := cur.Content
		if v, ok := fields["content"].(string); ok {
			content = v
		}
		if strings.TrimSpace(title) == "" || strings.TrimSpace(content) == "" {
			return nil, blogerr.InvalidState("a published post needs a title and content")
		}
		if cur.Status != model.StatusPublished {
			fields["published_at"] = gtime.Now()
			firstPublish = true
		}
	}
	// Snapshot the pre-edit title/content as a revision when either changes.
	if _, ok := fields["title"]; ok {
		s.snapshotRevision(ctx, author, cur, "edit")
	} else if _, ok := fields["content"]; ok {
		s.snapshotRevision(ctx, author, cur, "edit")
	}
	n, err := s.dao.Patch(ctx, author, id, fields)
	if err != nil {
		if err == dao.ErrSlugTaken {
			return nil, blogerr.SlugTaken(slugStr)
		}
		return nil, err
	}
	if n == 0 {
		return nil, blogerr.NotFound(id)
	}
	updated, err := s.dao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if firstPublish && updated != nil {
		go s.NotifyNewPost(context.Background(), updated) // newsletter digest to confirmed subscribers
	}
	return updated, nil
}

// Delete soft-deletes the author's post.
func (s *Service) Delete(ctx context.Context, author, id string) error {
	n, err := s.dao.SoftDelete(ctx, author, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return blogerr.NotFound(id)
	}
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func norm(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	return page, size
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		switch {
		// keep Unicode letters/digits so a pure-CJK name (e.g. 技术) yields a
		// non-empty slug instead of being stripped to "" by ASCII-only rules.
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevHyphen = false
		case b.Len() > 0 && !prevHyphen:
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
