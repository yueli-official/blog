package catalog

import (
	"context"

	"platform/products/blog/api/internal/dao"
	"platform/products/blog/api/internal/model"
)

// AuthorPage bundles a public author page. Identity owns the author's display
// profile; Blog only owns the published content and its aggregate statistics.
type AuthorPage struct {
	Posts      []*model.Post
	Total      int
	TotalViews int64
	Page       int
	Size       int
}

// GetAuthorPage returns an author's public profile + a page of their published
// posts (newest first).
func (s *Service) GetAuthorPage(ctx context.Context, authorID string, page, size int) (*AuthorPage, error) {
	page, size = norm(page, size)
	posts, total, err := s.dao.List(ctx, dao.ListFilter{AuthorID: authorID}, size, (page-1)*size)
	if err != nil {
		return nil, err
	}
	views, err := s.dao.TotalViewsByAuthor(ctx, authorID)
	if err != nil {
		return nil, err
	}
	return &AuthorPage{Posts: posts, Total: total, TotalViews: views, Page: page, Size: size}, nil
}
