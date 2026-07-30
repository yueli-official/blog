package dao

import (
	"context"

	"github.com/gogf/gf/v2/os/gtime"

	"github.com/yueli-official/blog/api/internal/model"
)

// SiblingPosts returns the published posts adjacent to the given one by publish
// time: prev is the next-older post, next is the next-newer (article prev/next
// nav, M6). Either may be nil at the ends of the timeline.
func (p *PG) SiblingPosts(ctx context.Context, publishedAt *gtime.Time, id string) (prev, next *model.Post, err error) {
	pub := string(model.StatusPublished)
	if err = p.db.Model(tPosts).Ctx(ctx).
		Where("status", pub).Where("deleted_at IS NULL").Where("id != ?", id).
		Where("published_at < ?", publishedAt).
		OrderDesc("published_at").Limit(1).Scan(&prev); err != nil {
		return nil, nil, err
	}
	if err = p.db.Model(tPosts).Ctx(ctx).
		Where("status", pub).Where("deleted_at IS NULL").Where("id != ?", id).
		Where("published_at > ?", publishedAt).
		OrderAsc("published_at").Limit(1).Scan(&next); err != nil {
		return nil, nil, err
	}
	return prev, next, nil
}

// ListArchive returns one page of lightweight rows (id/slug/title/dates), newest
// first, plus the total published count, for the date-grouped archive page (M6).
// Paginated so the page loads more on demand instead of pulling the whole history.
func (p *PG) ListArchive(ctx context.Context, limit, offset int) ([]*model.Post, int, error) {
	m := p.db.Model(tPosts+" p").Ctx(ctx).
		Where("p.status", string(model.StatusPublished)).Where("p.deleted_at IS NULL")
	total, err := m.Clone().Count()
	if err != nil {
		return nil, 0, err
	}
	var out []*model.Post
	err = m.LeftJoin(tStats+" s", "s.post_id = p.id").
		Fields("p.id, p.slug, p.title, p.published_at, p.created_at, COALESCE(s.view_count, 0) AS view_count").
		OrderDesc("p.published_at").Limit(limit).Offset(offset).Scan(&out)
	if out == nil {
		out = []*model.Post{}
	}
	return out, total, err
}
