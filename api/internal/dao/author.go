package dao

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	"platform/products/blog/api/internal/model"
)

const tAuthorProfiles = "author_profiles"

// GetAuthorProfile returns the author's profile, or (nil, nil) when none exists
// yet (the author has never edited it).
func (p *PG) GetAuthorProfile(ctx context.Context, authorID string) (*model.AuthorProfile, error) {
	var m *model.AuthorProfile
	if err := p.db.Model(tAuthorProfiles).Ctx(ctx).Where("author_id", authorID).Limit(1).Scan(&m); err != nil {
		return nil, err
	}
	return m, nil
}

// EnsureAuthorProfile creates a bare profile row (role 'contributor', status
// 'pending' per migrations 0010/0009) when the author has none yet — i.e. an
// authorship request awaiting owner approval.
func (p *PG) EnsureAuthorProfile(ctx context.Context, authorID string) error {
	_, err := p.db.Exec(ctx, "INSERT INTO author_profiles (author_id) VALUES (?) ON CONFLICT (author_id) DO NOTHING", authorID)
	return err
}

// IsActiveAuthor reports whether the author has an approved (active) profile.
func (p *PG) IsActiveAuthor(ctx context.Context, authorID string) (bool, error) {
	n, err := p.db.Model(tAuthorProfiles).Ctx(ctx).Where("author_id", authorID).Where("status", "active").Count()
	return n > 0, err
}

// DeleteAuthorProfile removes a profile row (admin reject/revoke). The author's
// existing posts are untouched (byline falls back to the id).
func (p *PG) DeleteAuthorProfile(ctx context.Context, authorID string) error {
	_, err := p.db.Model(tAuthorProfiles).Ctx(ctx).Where("author_id", authorID).Delete()
	return err
}

// ListAuthorRoster returns every author profile with their non-deleted post
// count, most-active first (admin author management).
func (p *PG) ListAuthorRoster(ctx context.Context) ([]*model.AuthorRoster, error) {
	var out []*model.AuthorRoster
	err := p.db.Model(tAuthorProfiles+" ap").Ctx(ctx).
		LeftJoin(tPosts+" p", "p.author_id = ap.author_id AND p.deleted_at IS NULL").
		Fields("ap.author_id, ap.role, ap.status, ap.created_at, COUNT(p.id) AS post_count").
		Group("ap.author_id, ap.role, ap.status, ap.created_at").
		OrderAsc("ap.status").OrderDesc("post_count").Scan(&out)
	if out == nil {
		out = []*model.AuthorRoster{}
	}
	return out, err
}

// TotalViewsByAuthor sums view_count across the author's published, non-deleted
// posts (author page stat strip). Returns 0 when there are none.
func (p *PG) TotalViewsByAuthor(ctx context.Context, authorID string) (int64, error) {
	v, err := p.db.Model(tPosts+" p").Ctx(ctx).
		LeftJoin(tStats+" s", "s.post_id = p.id").
		Where("p.author_id", authorID).
		Where("p.status", string(model.StatusPublished)).
		Where("p.deleted_at IS NULL").
		Fields("COALESCE(SUM(s.view_count), 0)").Value()
	if err != nil {
		return 0, err
	}
	return v.Int64(), nil
}

// UpsertAuthorProfile inserts or updates the given fields for the author's
// profile (self-service edit, or admin role change). fields holds mapped columns.
func (p *PG) UpsertAuthorProfile(ctx context.Context, authorID string, fields g.Map) error {
	fields["updated_at"] = gtime.Now()
	n, err := p.db.Model(tAuthorProfiles).Ctx(ctx).Where("author_id", authorID).Count()
	if err != nil {
		return err
	}
	if n == 0 {
		fields["author_id"] = authorID
		_, err = p.db.Model(tAuthorProfiles).Ctx(ctx).Data(fields).Insert()
		return err
	}
	_, err = p.db.Model(tAuthorProfiles).Ctx(ctx).Where("author_id", authorID).Data(fields).Update()
	return err
}
