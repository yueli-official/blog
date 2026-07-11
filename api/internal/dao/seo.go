package dao

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"platform/products/blog/api/internal/model"
)

const tSEO = "post_seo"

// UpsertSEO ensures the post's SEO row exists, then applies the given fields.
func (p *PG) UpsertSEO(ctx context.Context, postID string, fields g.Map) error {
	if _, err := p.db.Exec(ctx, "INSERT INTO post_seo (post_id) VALUES (?) ON CONFLICT (post_id) DO NOTHING", postID); err != nil {
		return err
	}
	if len(fields) == 0 {
		return nil
	}
	_, err := p.db.Model(tSEO).Ctx(ctx).Where("post_id", postID).Data(fields).Update()
	return err
}

// GetSEO returns the post's SEO metadata, or (nil, nil) when absent.
func (p *PG) GetSEO(ctx context.Context, postID string) (*model.SEO, error) {
	var s *model.SEO
	if err := p.db.Model(tSEO).Ctx(ctx).Where("post_id", postID).Limit(1).Scan(&s); err != nil {
		return nil, err
	}
	return s, nil
}
