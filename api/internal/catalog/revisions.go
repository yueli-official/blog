package catalog

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/model"
)

// snapshotRevision records the post's current title/content as a revision
// (best-effort — a failed snapshot must not block the edit).
func (s *Service) snapshotRevision(ctx context.Context, author string, cur *model.Post, note string) {
	_ = s.dao.InsertRevision(ctx, &model.Revision{
		PostID: cur.ID, AuthorID: author, Title: cur.Title, Content: cur.Content, RevNote: note,
	})
}

// IncrementView bumps the view counter of a published post (by slug).
func (s *Service) IncrementView(ctx context.Context, slug string) error {
	p, err := s.dao.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}
	if p == nil || p.Status != model.StatusPublished {
		return blogerr.NotFound(slug)
	}
	return s.dao.IncrementView(ctx, p.ID)
}

// ListRevisions returns a post's revisions (author-gated).
func (s *Service) ListRevisions(ctx context.Context, author, postID string) ([]*model.Revision, error) {
	if _, err := s.ownedPost(ctx, author, postID); err != nil {
		return nil, err
	}
	return s.dao.ListRevisions(ctx, postID)
}

// RestoreRevision snapshots the current content, then restores the post to a
// past revision's title/content (author-gated).
func (s *Service) RestoreRevision(ctx context.Context, author, postID, revID string) (*model.Post, error) {
	cur, err := s.ownedPost(ctx, author, postID)
	if err != nil {
		return nil, err
	}
	rev, err := s.dao.GetRevision(ctx, revID)
	if err != nil {
		return nil, err
	}
	if rev == nil || rev.PostID != postID {
		return nil, blogerr.NotFound(revID)
	}
	s.snapshotRevision(ctx, author, cur, "pre-restore")
	if _, err := s.dao.Patch(ctx, author, postID, g.Map{"title": rev.Title, "content": rev.Content}); err != nil {
		return nil, err
	}
	return s.dao.GetByID(ctx, postID)
}
