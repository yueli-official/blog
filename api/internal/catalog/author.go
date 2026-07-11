package catalog

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/dao"
	"platform/products/blog/api/internal/model"
)

// AuthorPage bundles a public author page: the profile overlay (may be nil if
// the author never edited it), their published posts, and the total count.
type AuthorPage struct {
	Profile    *model.AuthorProfile
	Posts      []*model.Post
	Total      int
	TotalViews int64
	Page       int
	Size       int
}

// GetAuthorPage returns an author's public profile + a page of their published
// posts (newest first).
func (s *Service) GetAuthorPage(ctx context.Context, authorID string, page, size int) (*AuthorPage, error) {
	prof, err := s.dao.GetAuthorProfile(ctx, authorID)
	if err != nil {
		return nil, err
	}
	page, size = norm(page, size)
	posts, total, err := s.dao.List(ctx, dao.ListFilter{AuthorID: authorID}, size, (page-1)*size)
	if err != nil {
		return nil, err
	}
	views, err := s.dao.TotalViewsByAuthor(ctx, authorID)
	if err != nil {
		return nil, err
	}
	return &AuthorPage{Profile: prof, Posts: posts, Total: total, TotalViews: views, Page: page, Size: size}, nil
}

// GetMyProfile returns the caller's own author profile (may be nil).
func (s *Service) GetMyProfile(ctx context.Context, author string) (*model.AuthorProfile, error) {
	return s.dao.GetAuthorProfile(ctx, author)
}

// RequireAuthor gates write actions: admins always pass; everyone else needs an
// approved (active) author profile. Returns Forbidden otherwise.
func (s *Service) RequireAuthor(ctx context.Context, author string, admin bool) error {
	if admin {
		return nil
	}
	ok, err := s.dao.IsActiveAuthor(ctx, author)
	if err != nil {
		return err
	}
	if !ok {
		return blogerr.Forbidden()
	}
	return nil
}

// RequestAuthor records an authorship request (a pending profile) for a logged-in
// user who isn't yet an author, then returns their profile.
func (s *Service) RequestAuthor(ctx context.Context, author string) (*model.AuthorProfile, error) {
	if err := s.dao.EnsureAuthorProfile(ctx, author); err != nil {
		return nil, err
	}
	return s.dao.GetAuthorProfile(ctx, author)
}

// AdminApproveAuthor approves a pending request → active (admin).
func (s *Service) AdminApproveAuthor(ctx context.Context, authorID string) (*model.AuthorProfile, error) {
	if err := s.dao.UpsertAuthorProfile(ctx, authorID, g.Map{"status": "active"}); err != nil {
		return nil, err
	}
	return s.dao.GetAuthorProfile(ctx, authorID)
}

// AdminRemoveAuthor rejects a request / revokes authorship by deleting the
// profile row (admin).
func (s *Service) AdminRemoveAuthor(ctx context.Context, authorID string) error {
	return s.dao.DeleteAuthorProfile(ctx, authorID)
}

// AdminListAuthors returns the full author roster with post counts (admin).
func (s *Service) AdminListAuthors(ctx context.Context) ([]*model.AuthorRoster, error) {
	return s.dao.ListAuthorRoster(ctx)
}

// AdminSetAuthorRole sets an author's role (主笔 author / 客座 contributor),
// creating the profile row if needed. Owner-governed.
func (s *Service) AdminSetAuthorRole(ctx context.Context, authorID, role string) (*model.AuthorProfile, error) {
	if role != "author" && role != "contributor" {
		return nil, blogerr.InvalidState("role must be author or contributor")
	}
	if err := s.dao.UpsertAuthorProfile(ctx, authorID, g.Map{"role": role}); err != nil {
		return nil, err
	}
	return s.dao.GetAuthorProfile(ctx, authorID)
}
