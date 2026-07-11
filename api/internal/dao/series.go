package dao

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"

	"platform/products/blog/api/internal/model"
)

const tSeries = "series"

// InsertSeries writes a new series row.
func (p *PG) InsertSeries(ctx context.Context, m *model.Series) error {
	_, err := p.db.Model(tSeries).Ctx(ctx).Data(g.Map{
		"id": m.ID, "slug": m.Slug, "name": m.Name, "description": m.Description,
		"author_id": m.AuthorID,
	}).Insert()
	return err
}

func (p *PG) GetSeriesByID(ctx context.Context, id string) (*model.Series, error) {
	var s *model.Series
	err := p.db.Model(tSeries).Ctx(ctx).Where("id", id).Limit(1).Scan(&s)
	return s, err
}

func (p *PG) GetSeriesBySlug(ctx context.Context, slug string) (*model.Series, error) {
	var s *model.Series
	err := p.db.Model(tSeries).Ctx(ctx).Where("slug", slug).Limit(1).Scan(&s)
	return s, err
}

// ListSeries returns every series, newest first. Never returns nil.
func (p *PG) ListSeries(ctx context.Context) ([]*model.Series, error) {
	var out []*model.Series
	err := p.db.Model(tSeries).Ctx(ctx).OrderDesc("created_at").Scan(&out)
	if out == nil {
		out = []*model.Series{}
	}
	return out, err
}

// SeriesPostCounts returns published-post counts keyed by series id.
func (p *PG) SeriesPostCounts(ctx context.Context) (map[string]int, error) {
	var rows []struct {
		SeriesID string `orm:"series_id"`
		C        int    `orm:"c"`
	}
	err := p.db.Model(tPosts).Ctx(ctx).
		Where("series_id IS NOT NULL").
		Where("status", string(model.StatusPublished)).
		Where("deleted_at IS NULL").
		Fields("series_id, COUNT(*) AS c").
		Group("series_id").Scan(&rows)
	if err != nil {
		return nil, err
	}
	m := make(map[string]int, len(rows))
	for _, r := range rows {
		m[r.SeriesID] = r.C
	}
	return m, nil
}

// UpdateSeries applies the given fields to a series (touch updated_at).
func (p *PG) UpdateSeries(ctx context.Context, id string, fields g.Map) error {
	fields["updated_at"] = gdb.Raw("now()")
	_, err := p.db.Model(tSeries).Ctx(ctx).Where("id", id).Data(fields).Update()
	return err
}

// DeleteSeries removes a series; member posts have series_id set NULL via FK.
func (p *PG) DeleteSeries(ctx context.Context, id string) error {
	_, err := p.db.Model(tSeries).Ctx(ctx).Where("id", id).Delete()
	return err
}

// PostsBySeries returns the published posts in a series, ordered by series_order
// then publish time. Never returns nil.
func (p *PG) PostsBySeries(ctx context.Context, seriesID string) ([]*model.Post, error) {
	var out []*model.Post
	err := p.db.Model(tPosts).Ctx(ctx).
		Where("series_id", seriesID).
		Where("status", string(model.StatusPublished)).
		Where("deleted_at IS NULL").
		OrderAsc("series_order").OrderAsc("published_at").Scan(&out)
	if out == nil {
		out = []*model.Post{}
	}
	return out, err
}

// RecentPostsBySeries returns up to `limit` published posts of a series, in
// reading order (for the home series carousel preview). Never returns nil.
func (p *PG) RecentPostsBySeries(ctx context.Context, seriesID string, limit int) ([]*model.Post, error) {
	var out []*model.Post
	err := p.db.Model(tPosts).Ctx(ctx).
		Where("series_id", seriesID).
		Where("status", string(model.StatusPublished)).
		Where("deleted_at IS NULL").
		OrderAsc("series_order").OrderAsc("published_at").Limit(limit).Scan(&out)
	if out == nil {
		out = []*model.Post{}
	}
	return out, err
}

// SetPostSeries assigns (or clears, when seriesID is empty) a post's series and
// its order within it. Author ownership is checked by the caller.
func (p *PG) SetPostSeries(ctx context.Context, postID, seriesID string, order int) error {
	data := g.Map{"series_order": order, "updated_at": gdb.Raw("now()")}
	if seriesID == "" {
		data["series_id"] = nil
	} else {
		data["series_id"] = seriesID
	}
	_, err := p.db.Model(tPosts).Ctx(ctx).Where("id", postID).Data(data).Update()
	return err
}
