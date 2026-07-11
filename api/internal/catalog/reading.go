package catalog

import (
	"context"

	"platform/products/blog/api/internal/model"
)

// Siblings returns the published posts adjacent to the given one by publish time
// (article prev/next nav, M6). Either may be nil at the ends of the timeline.
func (s *Service) Siblings(ctx context.Context, slug string) (prev, next *model.Post, err error) {
	p, err := s.publishedBySlug(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	return s.dao.SiblingPosts(ctx, p.PublishedAt, p.ID)
}

// Archive returns one normalized page of lightweight rows (newest first) plus the
// total published count, for the date-grouped archive page (M6).
func (s *Service) Archive(ctx context.Context, page, size int) ([]*model.Post, int, int, int, error) {
	page, size = norm(page, size)
	items, total, err := s.dao.ListArchive(ctx, size, (page-1)*size)
	return items, total, page, size, err
}
