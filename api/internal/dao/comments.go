package dao

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	"github.com/yueli-official/blog/api/internal/model"
	"github.com/yueli-official/foundation/go/identifier"
)

const tComments = "comments"

// InsertComment writes a new comment and recomputes the post's approved
// comment_count in the same transaction. ParentID must be "" (top-level) or the
// id of an existing top-level comment on the same post.
func (p *PG) InsertComment(ctx context.Context, m *model.Comment) error {
	if m.ID == "" {
		m.ID = identifier.MustNew().String()
	}
	var parent any // nil → SQL NULL (top-level)
	if m.ParentID != "" {
		parent = m.ParentID
	}
	return p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Model(tComments).Ctx(ctx).Data(g.Map{
			"id": m.ID, "post_id": m.PostID, "parent_id": parent,
			"user_id": m.UserID, "author_name": m.AuthorName, "author_email": m.AuthorEmail,
			"content": m.Content, "status": int(m.Status), "ip": m.IP, "user_agent": m.UserAgent,
		}).Insert(); err != nil {
			return err
		}
		return recomputeCommentCount(ctx, tx, m.PostID)
	})
}

// ListApproved returns a post's approved top-level comments in the requested
// date order plus the total count of approved top-level comments.
func (p *PG) ListApproved(ctx context.Context, postID string, ascending bool, limit, offset int) ([]*model.Comment, int, error) {
	m := p.db.Model(tComments).Ctx(ctx).
		Where("post_id", postID).Where("status", int(model.CommentApproved)).
		Where("parent_id IS NULL").Where("deleted_at IS NULL")
	total, err := m.Clone().Count()
	if err != nil {
		return nil, 0, err
	}
	var out []*model.Comment
	if ascending {
		m = m.OrderAsc("created_at").OrderAsc("id")
	} else {
		m = m.OrderDesc("created_at").OrderDesc("id")
	}
	if err := m.Limit(limit).Offset(offset).Scan(&out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// RepliesByParents returns approved replies grouped by parent id (oldest first
// within each thread).
func (p *PG) RepliesByParents(ctx context.Context, parentIDs []string) (map[string][]*model.Comment, error) {
	out := map[string][]*model.Comment{}
	if len(parentIDs) == 0 {
		return out, nil
	}
	var rows []*model.Comment
	if err := p.db.Model(tComments).Ctx(ctx).
		WhereIn("parent_id", parentIDs).Where("status", int(model.CommentApproved)).Where("deleted_at IS NULL").
		OrderAsc("created_at").Scan(&rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ParentID] = append(out[r.ParentID], r)
	}
	return out, nil
}

// GetComment returns one non-deleted comment, or (nil, nil).
func (p *PG) GetComment(ctx context.Context, id string) (*model.Comment, error) {
	var c *model.Comment
	if err := p.db.Model(tComments).Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Limit(1).Scan(&c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListMineComments returns comments on posts authored by `author` (any status,
// or filtered when status != 0), ordered by creation time, plus the total.
// When author is "" (site-wide admin moderation), all posts are included.
func (p *PG) ListMineComments(ctx context.Context, author string, status int, keyword string, ascending bool, limit, offset int) ([]*model.Comment, int, error) {
	m := p.db.Model(tComments+" c").Ctx(ctx).
		LeftJoin(tPosts+" p", "p.id=c.post_id").
		Where("c.deleted_at IS NULL")
	if author != "" { // empty author = site-wide (admin)
		m = m.Where("p.author_id", author)
	}
	if status != 0 {
		m = m.Where("c.status", status)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		m = m.Where("(c.content ILIKE ? OR c.author_name ILIKE ? OR c.author_email ILIKE ? OR p.title ILIKE ? OR p.slug ILIKE ?)", like, like, like, like, like)
	}
	total, err := m.Clone().Count()
	if err != nil {
		return nil, 0, err
	}
	var out []*model.Comment
	ordered := m.Fields("c.*")
	if ascending {
		ordered = ordered.OrderAsc("c.created_at")
	} else {
		ordered = ordered.OrderDesc("c.created_at")
	}
	if err := ordered.Limit(limit).Offset(offset).Scan(&out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// PostHeadsByIDs returns id→head for the given post ids.
func (p *PG) PostHeadsByIDs(ctx context.Context, ids []string) (map[string]model.PostHead, error) {
	out := map[string]model.PostHead{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []*model.PostHead
	if err := p.db.Model(tPosts).Ctx(ctx).Fields("id, title, slug").WhereIn("id", ids).Scan(&rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = *r
	}
	return out, nil
}

// SetCommentStatus updates a comment's status and recomputes the post's approved
// count. Returns rows affected (0 = absent/deleted).
func (p *PG) SetCommentStatus(ctx context.Context, id string, status int) (int64, error) {
	var affected int64
	err := p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		var c *model.Comment
		if err := tx.Model(tComments).Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Limit(1).Scan(&c); err != nil {
			return err
		}
		if c == nil {
			return nil
		}
		r, err := tx.Model(tComments).Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Update(g.Map{"status": status})
		if err != nil {
			return err
		}
		affected, _ = r.RowsAffected()
		return recomputeCommentCount(ctx, tx, c.PostID)
	})
	return affected, err
}

// SoftDeleteComment soft-deletes a comment and (for a top-level comment) its
// direct replies, then recomputes the post's approved count. Returns rows
// affected for the target comment (0 = absent/already deleted).
func (p *PG) SoftDeleteComment(ctx context.Context, id string) (int64, error) {
	var affected int64
	err := p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		var c *model.Comment
		if err := tx.Model(tComments).Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Limit(1).Scan(&c); err != nil {
			return err
		}
		if c == nil {
			return nil
		}
		now := gtime.Now()
		r, err := tx.Model(tComments).Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Update(g.Map{"deleted_at": now})
		if err != nil {
			return err
		}
		affected, _ = r.RowsAffected()
		if _, err := tx.Model(tComments).Ctx(ctx).Where("parent_id", id).Where("deleted_at IS NULL").Update(g.Map{"deleted_at": now}); err != nil {
			return err
		}
		return recomputeCommentCount(ctx, tx, c.PostID)
	})
	return affected, err
}

// recomputeCommentCount sets post_stats.comment_count to the current number of
// approved, non-deleted comments (drift-free; the comment service owns this
// count rather than relying on incremental events).
func recomputeCommentCount(ctx context.Context, tx gdb.TX, postID string) error {
	if _, err := tx.Ctx(ctx).Exec("INSERT INTO post_stats (post_id) VALUES (?) ON CONFLICT (post_id) DO NOTHING", postID); err != nil {
		return err
	}
	n, err := tx.Model(tComments).Ctx(ctx).
		Where("post_id", postID).Where("status", int(model.CommentApproved)).Where("deleted_at IS NULL").Count()
	if err != nil {
		return err
	}
	_, err = tx.Model(tStats).Ctx(ctx).Where("post_id", postID).Update(g.Map{"comment_count": n})
	return err
}
