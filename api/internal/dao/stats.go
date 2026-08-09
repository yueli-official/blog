package dao

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"github.com/yueli-official/blog/api/internal/model"
	"github.com/yueli-official/foundation/go/identifier"
)

const (
	tStats     = "post_stats"
	tRevisions = "post_revisions"
)

// EnsureStats creates the post's stats row if absent (idempotent).
func (p *PG) EnsureStats(ctx context.Context, postID string) error {
	_, err := p.db.Exec(ctx, "INSERT INTO post_stats (post_id) VALUES (?) ON CONFLICT (post_id) DO NOTHING", postID)
	return err
}

// AdvanceViewProjection moves the consumer-owned read projection forward to a
// module total. GREATEST makes replay and out-of-order completion harmless.
func (p *PG) AdvanceViewProjection(ctx context.Context, postID string, views int64) error {
	if err := p.EnsureStats(ctx, postID); err != nil {
		return err
	}
	_, err := p.db.Exec(ctx,
		"UPDATE post_stats SET view_count = GREATEST(view_count, ?), updated_at = NOW() WHERE post_id = ?",
		views, postID,
	)
	return err
}

// ReplaceViewProjection is used before the HTTP server starts, when no traffic
// writes are concurrent, to repair drift from module truth.
func (p *PG) ReplaceViewProjection(ctx context.Context, postID string, views int64) error {
	if err := p.EnsureStats(ctx, postID); err != nil {
		return err
	}
	_, err := p.db.Exec(ctx,
		"UPDATE post_stats SET view_count = ?, updated_at = NOW() WHERE post_id = ?",
		views, postID,
	)
	return err
}

func (p *PG) ListPostIDs(ctx context.Context) ([]string, error) {
	var rows []struct {
		ID string `orm:"id"`
	}
	if err := p.db.Model("posts").Ctx(ctx).Fields("id").OrderAsc("id").Scan(&rows); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids, nil
}

// GetStats returns the post's counters, or (nil, nil) when absent.
func (p *PG) GetStats(ctx context.Context, postID string) (*model.Stats, error) {
	var st *model.Stats
	if err := p.db.Model(tStats).Ctx(ctx).Where("post_id", postID).Limit(1).Scan(&st); err != nil {
		return nil, err
	}
	return st, nil
}

// InsertRevision records a title/content snapshot.
func (p *PG) InsertRevision(ctx context.Context, r *model.Revision) error {
	if r.ID == "" {
		r.ID = identifier.MustNew().String()
	}
	_, err := p.db.Model(tRevisions).Ctx(ctx).Data(g.Map{
		"id": r.ID, "post_id": r.PostID, "author_id": r.AuthorID,
		"title": r.Title, "content": r.Content, "rev_note": r.RevNote,
	}).Insert()
	return err
}

// ListRevisions returns a post's revisions, newest first.
func (p *PG) ListRevisions(ctx context.Context, postID string) ([]*model.Revision, error) {
	var out []*model.Revision
	err := p.db.Model(tRevisions).Ctx(ctx).Where("post_id", postID).OrderDesc("created_at").Scan(&out)
	return out, err
}

// GetRevision returns one revision, or (nil, nil).
func (p *PG) GetRevision(ctx context.Context, revID string) (*model.Revision, error) {
	var r *model.Revision
	if err := p.db.Model(tRevisions).Ctx(ctx).Where("id", revID).Limit(1).Scan(&r); err != nil {
		return nil, err
	}
	return r, nil
}
