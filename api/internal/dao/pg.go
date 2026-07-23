// Package dao is the PostgreSQL data-access layer for the blog catalog.
package dao

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"

	"platform/products/blog/api/internal/model"
)

const tPosts = "posts"

// ErrSlugTaken is returned by Insert when the slug unique index is violated.
var ErrSlugTaken = errors.New("dao: slug taken")

// PG wraps the GoFrame gdb handle.
type PG struct{ db gdb.DB }

func NewPG(db gdb.DB) *PG { return &PG{db: db} }

// TransactionHook lets a product mutation update a Foundation module through
// the exact same database transaction. The hook must not commit or roll back.
type TransactionHook func(context.Context, *sql.Tx) error
type CreateTransactionHook func(context.Context, *sql.Tx, string) error

func runTransactionHook(ctx context.Context, tx gdb.TX, hook TransactionHook) error {
	if hook == nil {
		return nil
	}
	return hook(ctx, tx.GetSqlTX())
}

func runCreateTransactionHook(ctx context.Context, tx gdb.TX, id string, hook CreateTransactionHook) error {
	if hook == nil {
		return nil
	}
	return hook(ctx, tx.GetSqlTX(), id)
}

// ListFilter narrows a List query. Status defaults to published; IDs (when
// non-nil) restricts to a set of post ids (taxonomy archive filter, Task 4); Q
// is a site-search query (PG full-text over title/excerpt/content via zhparser).
type ListFilter struct {
	Status   string
	IDs      []string
	Q        string
	Featured bool   // when true, restrict to featured posts (home carousel)
	Pinned   bool   // when true, restrict to pinned posts
	AuthorID string // when set, restrict to this author's posts (author page)
	Sort     string // "" (pinned+newest) | "popular" (most viewed) | "random"
}

// Insert writes a new post (generating id when unset). A slug unique violation
// maps to ErrSlugTaken so the service can retry with a suffix.
func (p *PG) Insert(ctx context.Context, m *model.Post) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	postType := m.PostType
	if postType == "" {
		postType = "post"
	}
	locale := m.Locale
	if locale == "" {
		locale = "zh-CN"
	}
	data := g.Map{
		"id": m.ID, "author_id": m.AuthorID, "post_type": postType,
		"title": m.Title, "slug": m.Slug, "content": m.Content, "excerpt": m.Excerpt,
		"cover_asset_id": m.CoverAssetID, "cover_url": m.CoverURL,
		"comment_status": m.CommentStatus, "status": string(m.Status),
		"locale": locale, "published_at": m.PublishedAt,
	}
	if _, err := p.db.Model(tPosts).Ctx(ctx).Data(data).Insert(); err != nil {
		if isDupSlug(err) {
			return ErrSlugTaken
		}
		return err
	}
	return nil
}

// GetByID returns the post, or (nil, nil) when absent or soft-deleted.
func (p *PG) GetByID(ctx context.Context, id string) (*model.Post, error) {
	return p.one(ctx, "id", id)
}

// GetBySlug returns the post by slug, or (nil, nil).
func (p *PG) GetBySlug(ctx context.Context, slug string) (*model.Post, error) {
	return p.one(ctx, "slug", slug)
}

// List returns posts matching the filter plus the total count (newest published
// first). Status defaults to published; soft-deleted rows are excluded.
func (p *PG) List(ctx context.Context, f ListFilter, limit, offset int) ([]*model.Post, int, error) {
	status := f.Status
	if status == "" {
		status = string(model.StatusPublished)
	}
	if f.IDs != nil && len(f.IDs) == 0 {
		return []*model.Post{}, 0, nil
	}
	if f.Q != "" {
		return p.search(ctx, status, f, limit, offset)
	}
	// Aliased `p` so the popular sort can LeftJoin post_stats without ambiguity.
	m := p.db.Model(tPosts+" p").Ctx(ctx).Where("p.status", status).Where("p.deleted_at IS NULL")
	if len(f.IDs) > 0 {
		m = m.WhereIn("p.id", f.IDs)
	}
	if f.Featured {
		m = m.Where("p.featured", true)
	}
	if f.Pinned {
		m = m.Where("p.pinned", true)
	}
	if f.AuthorID != "" {
		m = m.Where("p.author_id", f.AuthorID)
	}
	total, err := m.Clone().Count()
	if err != nil {
		return nil, 0, err
	}
	var out []*model.Post
	// Join stats so list cards/widgets carry the real view_count (the popular sort
	// also orders by it). Count above ran on the un-joined model.
	q := m.LeftJoin(tStats+" s", "s.post_id = p.id").Fields("p.*, COALESCE(s.view_count, 0) AS view_count")
	switch f.Sort {
	case "popular":
		q = q.OrderDesc("s.view_count").OrderDesc("p.published_at")
	case "random":
		q = q.OrderRandom()
	default:
		// pinned posts surface first, then newest.
		q = q.OrderDesc("p.pinned").OrderDesc("p.published_at")
	}
	if err := q.Limit(limit).Offset(offset).Scan(&out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// search runs the full-text (zhparser + GIN) variant of List, ranked by ts_rank.
// websearch_to_tsquery is bound once as a FROM-clause alias `q` so the same parsed
// query drives both the `@@` match and the ts_rank ordering without re-binding.
// Config `chinese_zh` + the search_vector generated column come from migration 0004.
func (p *PG) search(ctx context.Context, status string, f ListFilter, limit, offset int) ([]*model.Post, int, error) {
	conds := []string{"p.status = ?", "p.deleted_at IS NULL", "p.search_vector @@ q"}
	// arg order must match the ? order in the SQL text: FROM-clause query first, then status.
	args := []any{f.Q, status}
	if len(f.IDs) > 0 {
		ph := make([]string, len(f.IDs))
		for i := range f.IDs {
			ph[i] = "?"
			args = append(args, f.IDs[i])
		}
		conds = append(conds, "p.id IN ("+strings.Join(ph, ",")+")")
	}
	from := "posts p, websearch_to_tsquery('chinese_zh', ?) q"
	where := strings.Join(conds, " AND ")

	total, err := p.db.GetValue(ctx, "SELECT COUNT(*) FROM "+from+" WHERE "+where, args...)
	if err != nil {
		return nil, 0, err
	}
	var out []*model.Post
	rowsSQL := "SELECT p.* FROM " + from + " WHERE " + where +
		" ORDER BY ts_rank(p.search_vector, q) DESC, p.published_at DESC LIMIT ? OFFSET ?"
	if err := p.db.Ctx(ctx).Raw(rowsSQL, append(args, limit, offset)...).Scan(&out); err != nil {
		return nil, 0, err
	}
	return out, total.Int(), nil
}

// ListManage powers the manage console post list. authorID "" = all authors
// (admin all-posts view); otherwise scoped to that author. status "" = any;
// status "issues" is the computed incomplete-content view; q
// filters title/slug (ILIKE); taxonomyIDs narrows (AND) to posts carrying every
// given category/tag id. Newest first.
func (p *PG) ListManage(ctx context.Context, authorID, status, q string, taxonomyIDs []string, pinned, featured bool, sort, direction string, limit, offset int) ([]*model.Post, int, error) {
	m := p.db.Model(tPosts).Ctx(ctx).Where("deleted_at IS NULL")
	if authorID != "" {
		m = m.Where("author_id", authorID)
	}
	if status == "issues" {
		m = m.Where("(BTRIM(title) = '' OR BTRIM(content) = '')")
	} else if status != "" {
		m = m.Where("status", status)
	}
	if pinned {
		m = m.Where("pinned", true)
	}
	if featured {
		m = m.Where("featured", true)
	}
	if q != "" {
		like := "%" + q + "%"
		m = m.Where("(title ILIKE ? OR slug ILIKE ?)", like, like)
	}
	for _, tid := range taxonomyIDs {
		if tid != "" {
			m = m.Where(`id IN (
                SELECT post_id FROM blog_post_category_assignments WHERE category_id::text = ?
                UNION
                SELECT post_id FROM blog_post_tag_assignments WHERE tag_id::text = ?
            )`, tid, tid)
		}
	}
	total, err := m.Clone().Count()
	if err != nil {
		return nil, 0, err
	}
	var out []*model.Post
	m = m.Order(manageOrder(sort, direction))
	if err := m.OrderDesc("id").Limit(limit).Offset(offset).Scan(&out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func manageOrder(sort, direction string) string {
	orderDirection := "DESC"
	if direction == "asc" {
		orderDirection = "ASC"
	}
	switch sort {
	case "title":
		return "title " + orderDirection
	case "published":
		return "published_at " + orderDirection + " NULLS LAST"
	default:
		return "updated_at " + orderDirection
	}
}

// StatusCounts returns post counts keyed by status, plus an "all" total (drives
// the manage console's filter tabs). authorID "" = all authors. Soft-deleted
// excluded. Counts reflect the author scope only (not the taxonomy/search filter).
func (p *PG) StatusCounts(ctx context.Context, authorID string) (map[string]int, error) {
	m := p.db.Model(tPosts).Ctx(ctx).Where("deleted_at IS NULL")
	if authorID != "" {
		m = m.Where("author_id", authorID)
	}
	var rows []struct {
		Status string `orm:"status"`
		C      int    `orm:"c"`
	}
	if err := m.Fields("status, COUNT(*) AS c").Group("status").Scan(&rows); err != nil {
		return nil, err
	}
	out := map[string]int{}
	total := 0
	for _, r := range rows {
		out[r.Status] = r.C
		total += r.C
	}
	out["all"] = total
	issuesQuery := p.db.Model(tPosts).Ctx(ctx).
		Where("deleted_at IS NULL").
		Where("(BTRIM(title) = '' OR BTRIM(content) = '')")
	if authorID != "" {
		issuesQuery = issuesQuery.Where("author_id", authorID)
	}
	issues, err := issuesQuery.Count()
	if err != nil {
		return nil, err
	}
	out["issues"] = issues
	return out, nil
}

// Patch updates mutable fields for the author's post; returns rows affected
// (0 = absent / not author / already deleted).
func (p *PG) Patch(ctx context.Context, author, id string, data g.Map) (int64, error) {
	return p.PatchWithHook(ctx, author, id, data, nil)
}

func (p *PG) PatchWithHook(ctx context.Context, author, id string, data g.Map, hook TransactionHook) (int64, error) {
	var affected int64
	err := p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		data["updated_at"] = gtime.Now()
		r, err := tx.Model(tPosts).Ctx(ctx).
			Where("author_id", author).Where("id", id).Where("deleted_at IS NULL").
			Data(data).Update()
		if err != nil {
			return err
		}
		affected, err = r.RowsAffected()
		if err != nil || affected == 0 {
			return err
		}
		return runTransactionHook(ctx, tx, hook)
	})
	if err != nil {
		if isDupSlug(err) {
			return 0, ErrSlugTaken
		}
		return 0, err
	}
	return affected, nil
}

// PatchByID updates fields on a post by id without an author check (admin paths:
// editorial flags + batch ops; ownership is verified by the service first).
func (p *PG) PatchByID(ctx context.Context, id string, data g.Map) error {
	return p.PatchByIDWithHook(ctx, id, data, nil)
}

func (p *PG) PatchByIDWithHook(ctx context.Context, id string, data g.Map, hook TransactionHook) error {
	return p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		data["updated_at"] = gtime.Now()
		if _, err := tx.Model(tPosts).Ctx(ctx).
			Where("id", id).Where("deleted_at IS NULL").Data(data).Update(); err != nil {
			return err
		}
		return runTransactionHook(ctx, tx, hook)
	})
}

// SoftDeleteByID soft-deletes a post by id without an author check (admin/batch;
// ownership verified by the service first).
func (p *PG) SoftDeleteByID(ctx context.Context, id string) error {
	return p.SoftDeleteByIDWithHook(ctx, id, nil)
}

func (p *PG) SoftDeleteByIDWithHook(ctx context.Context, id string, hook TransactionHook) error {
	return p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Model(tPosts).Ctx(ctx).
			Where("id", id).Where("deleted_at IS NULL").
			Data(g.Map{"deleted_at": gtime.Now()}).Update(); err != nil {
			return err
		}
		return runTransactionHook(ctx, tx, hook)
	})
}

// SoftDelete sets deleted_at for the author's post; returns rows affected.
func (p *PG) SoftDelete(ctx context.Context, author, id string) (int64, error) {
	return p.SoftDeleteWithHook(ctx, author, id, nil)
}

func (p *PG) SoftDeleteWithHook(ctx context.Context, author, id string, hook TransactionHook) (int64, error) {
	var affected int64
	err := p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		result, err := tx.Model(tPosts).Ctx(ctx).
			Where("author_id", author).Where("id", id).Where("deleted_at IS NULL").
			Data(g.Map{"deleted_at": gtime.Now()}).Update()
		if err != nil {
			return err
		}
		affected, err = result.RowsAffected()
		if err != nil || affected == 0 {
			return err
		}
		return runTransactionHook(ctx, tx, hook)
	})
	return affected, err
}

func (p *PG) one(ctx context.Context, col, val string) (*model.Post, error) {
	var m *model.Post
	if err := p.db.Model(tPosts).Ctx(ctx).
		Where(col, val).Where("deleted_at IS NULL").Limit(1).Scan(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func isDupSlug(err error) bool {
	// lib/pq surfaces `duplicate key value violates unique constraint
	// "uq_posts_slug"` without the 23505 SQLSTATE — match the message text too.
	s := err.Error()
	dup := strings.Contains(s, "23505") || strings.Contains(s, "duplicate key")
	return dup && (strings.Contains(s, "slug") || strings.Contains(s, "uq_posts_slug"))
}
