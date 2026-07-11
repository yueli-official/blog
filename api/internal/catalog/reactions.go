package catalog

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/model"
)

// Detail bundles everything the post detail page renders.
type Detail struct {
	Post       *model.Post
	Stats      *model.Stats
	SEO        *model.SEO
	Taxonomies []*model.Taxonomy
	Series     *model.Series
	Author     *model.AuthorProfile // the post author's profile overlay (may be nil)
	Liked      bool
	Bookmarked bool
}

// GetDetail returns a visible post plus its stats, SEO, and (for a logged-in
// viewer) the viewer's like/bookmark state.
func (s *Service) GetDetail(ctx context.Context, viewer, slug string) (*Detail, error) {
	p, err := s.GetBySlug(ctx, viewer, slug)
	if err != nil {
		return nil, err
	}
	st, err := s.dao.GetStats(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	seo, err := s.dao.GetSEO(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	tax, err := s.dao.GetPostTaxonomies(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	d := &Detail{Post: p, Stats: st, SEO: seo, Taxonomies: tax}
	if prof, err := s.dao.GetAuthorProfile(ctx, p.AuthorID); err == nil {
		d.Author = prof // best-effort byline enrichment; nil → UI falls back to the id
	}
	if p.SeriesID != "" {
		if se, err := s.dao.GetSeriesByID(ctx, p.SeriesID); err == nil && se != nil {
			d.Series = se
		}
	}
	if viewer != "" {
		d.Liked, _ = s.dao.IsLiked(ctx, viewer, p.ID)
		d.Bookmarked, _ = s.dao.IsBookmarked(ctx, viewer, p.ID)
	}
	return d, nil
}

// PutSEO upserts a post's SEO metadata (author-gated).
func (s *Service) PutSEO(ctx context.Context, author, postID string, fields g.Map) (*model.SEO, error) {
	if _, err := s.ownedPost(ctx, author, postID); err != nil {
		return nil, err
	}
	if err := s.dao.UpsertSEO(ctx, postID, fields); err != nil {
		return nil, err
	}
	return s.dao.GetSEO(ctx, postID)
}

// ToggleLike flips the viewer's like on a published post; returns new state.
func (s *Service) ToggleLike(ctx context.Context, user, slug string) (bool, error) {
	p, err := s.publishedBySlug(ctx, slug)
	if err != nil {
		return false, err
	}
	return s.dao.ToggleLike(ctx, user, p.ID)
}

// ToggleBookmark flips the viewer's bookmark on a published post.
func (s *Service) ToggleBookmark(ctx context.Context, user, slug string) (bool, error) {
	p, err := s.publishedBySlug(ctx, slug)
	if err != nil {
		return false, err
	}
	return s.dao.ToggleBookmark(ctx, user, p.ID)
}

func (s *Service) publishedBySlug(ctx context.Context, slug string) (*model.Post, error) {
	p, err := s.dao.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if p == nil || p.Status != model.StatusPublished {
		return nil, blogerr.NotFound(slug)
	}
	return p, nil
}
