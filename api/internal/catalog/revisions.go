package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/yueli-official/foundation/go/traffic"

	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/model"
)

const postTrafficKind traffic.ResourceKind = "post"

type ViewInput struct {
	EventID     string
	OccurredAt  time.Time
	Class       traffic.VisitClass
	VisitorSeed []byte
}

// snapshotRevision records the post's current title/content as a revision
// (best-effort — a failed snapshot must not block the edit).
func (s *Service) snapshotRevision(ctx context.Context, author string, cur *model.Post, note string) {
	_ = s.dao.InsertRevision(ctx, &model.Revision{
		PostID: cur.ID, AuthorID: author, Title: cur.Title, Content: cur.Content, RevNote: note,
	})
}

// RecordView records one idempotent view of a published post. The traffic
// module is the source of truth; post_stats is only a monotonic read projection.
func (s *Service) RecordView(ctx context.Context, slug string, input ViewInput) (traffic.RecordResult, error) {
	p, err := s.dao.GetBySlug(ctx, slug)
	if err != nil {
		return traffic.RecordResult{}, err
	}
	if p == nil || p.Status != model.StatusPublished {
		return traffic.RecordResult{}, blogerr.NotFound(slug)
	}
	if s.traffic == nil {
		return traffic.RecordResult{}, errors.New("blog traffic module is not configured")
	}
	if s.privacy != nil {
		gpc := false
		if request := ghttp.RequestFromCtx(ctx); request != nil {
			gpc = request.Header.Get("Sec-GPC") == "1"
		}
		allowed, err := s.privacy.CanMeasure(ctx, gpc)
		if err != nil {
			return traffic.RecordResult{}, err
		}
		if !allowed {
			return traffic.RecordResult{}, nil
		}
	}
	observation := traffic.Observation{
		EventID:    traffic.EventID(input.EventID),
		Resource:   traffic.Resource{Kind: postTrafficKind, ID: p.ID},
		OccurredAt: input.OccurredAt,
		Class:      input.Class,
	}
	if len(input.VisitorSeed) > 0 {
		token, err := s.traffic.TokenizeVisitor(ctx, input.OccurredAt, input.VisitorSeed)
		if err != nil {
			return traffic.RecordResult{}, err
		}
		observation.HasVisitor = true
		observation.VisitorToken = token
	}
	result, err := s.traffic.Record(ctx, observation)
	if err != nil {
		return traffic.RecordResult{}, err
	}
	if err := s.dao.AdvanceViewProjection(ctx, p.ID, result.ResourceTotals.Views); err != nil {
		return traffic.RecordResult{}, err
	}
	return result, nil
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
