package dao

import (
	"context"

	"platform/products/blog/api/internal/model"
)

// TotalViewsByAuthor sums view_count across the author's published,
// non-deleted posts. The author identity is the post's immutable author_id;
// role and application state live exclusively in Authorization.
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
