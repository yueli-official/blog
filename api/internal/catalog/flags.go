package catalog

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	"github.com/yueli-official/blog/api/internal/blogerr"
	"github.com/yueli-official/blog/api/internal/dao"
	"github.com/yueli-official/blog/api/internal/model"
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

type BatchFailure struct {
	ID      string
	Code    string
	Message string
}

// BatchStatus applies a lifecycle action to many posts, each gated by ownership
// (or admin). Publish uses the same title/content constraint as single-item
// Patch. Business-rule failures are returned per item; database failures stop
// the operation. Non-owned posts are skipped silently.
func (s *Service) BatchStatus(ctx context.Context, author string, isAdmin bool, ids []string, action string) (int, []*BatchFailure, error) {
	statusFor := map[string]string{"publish": "published", "draft": "draft", "archive": "archived"}
	st, ok := statusFor[action]
	if !ok && action != "delete" {
		return 0, nil, blogerr.InvalidInput("unknown batch action")
	}
	var (
		changed  int
		failures []*BatchFailure
	)
	for _, id := range ids {
		p, err := s.dao.GetByID(ctx, id)
		if err != nil {
			return changed, failures, err
		}
		if p == nil || (p.AuthorID != author && !isAdmin) {
			continue
		}
		if action == "delete" {
			if err := s.dao.SoftDeleteByIDWithHook(ctx, id, dao.ComposeTransactionHooks(
				s.urlDeleteHook(postURLState(p)), s.searchHook(id),
			)); err != nil {
				return changed, failures, err
			}
			changed++
			continue
		}

		fields := g.Map{"status": st}
		firstPublish := action == "publish" && p.Status != model.StatusPublished
		if action == "publish" {
			if strings.TrimSpace(p.Title) == "" || strings.TrimSpace(p.Content) == "" {
				failures = append(failures, &BatchFailure{
					ID:      id,
					Code:    "incomplete",
					Message: "发布前需要补充标题和正文",
				})
				continue
			}
			if firstPublish {
				fields["published_at"] = gtime.Now()
			}
		}
		beforeURL := postURLState(p)
		afterURL := beforeURL
		if firstPublish {
			afterURL.Published = true
		}
		if err := s.dao.PatchByIDWithHook(ctx, id, fields, dao.ComposeTransactionHooks(
			s.urlChangeHook(beforeURL, afterURL), s.searchHook(id),
		)); err != nil {
			return changed, failures, err
		}
		changed++
		if firstPublish {
			updated, getErr := s.dao.GetByID(ctx, id)
			if getErr != nil {
				return changed, failures, getErr
			}
			if updated != nil {
				go s.NotifyNewPost(context.Background(), updated)
			}
		}
	}
	return changed, failures, nil
}
