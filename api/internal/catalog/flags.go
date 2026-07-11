package catalog

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/model"
)

// SetFlags sets a post's editorial flags (pinned/featured); admin-gated at the
// controller. Nil pointers are left unchanged.
func (s *Service) SetFlags(ctx context.Context, id string, pinned, featured *bool) (*model.Post, error) {
	p, err := s.dao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, blogerr.NotFound(id)
	}
	fields := g.Map{}
	if pinned != nil {
		fields["pinned"] = *pinned
	}
	if featured != nil {
		fields["featured"] = *featured
	}
	if len(fields) > 0 {
		if err := s.dao.PatchByID(ctx, id, fields); err != nil {
			return nil, err
		}
	}
	return s.dao.GetByID(ctx, id)
}

// BatchStatus applies a lifecycle action to many posts, each gated by ownership
// (or admin). Returns the number actually changed. action ∈ publish|draft|
// archive|delete. Non-owned posts are skipped silently.
func (s *Service) BatchStatus(ctx context.Context, author string, isAdmin bool, ids []string, action string) (int, error) {
	statusFor := map[string]string{"publish": "published", "draft": "draft", "archive": "archived"}
	st, ok := statusFor[action]
	if !ok && action != "delete" {
		return 0, blogerr.InvalidInput("unknown batch action")
	}
	n := 0
	for _, id := range ids {
		p, err := s.dao.GetByID(ctx, id)
		if err != nil {
			return n, err
		}
		if p == nil || (p.AuthorID != author && !isAdmin) {
			continue
		}
		if action == "delete" {
			if err := s.dao.SoftDeleteByID(ctx, id); err != nil {
				return n, err
			}
		} else if err := s.dao.PatchByID(ctx, id, g.Map{"status": st}); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
