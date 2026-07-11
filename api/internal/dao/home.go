package dao

import (
	"context"

	"platform/products/blog/api/internal/model"
)

const tHomeConfig = "home_config"

func defaultHomeConfig() *model.HomeConfig {
	return &model.HomeConfig{
		Eyebrow:  "Editorial",
		Title:    "博客",
		Subtitle: "想法、笔记与记录, 关于技术、产品与日常的长短文。",
	}
}

func (p *PG) GetHomeConfig(ctx context.Context) (*model.HomeConfig, error) {
	var out *model.HomeConfig
	err := p.db.Model(tHomeConfig).Ctx(ctx).
		Fields("eyebrow", "title", "subtitle").
		Where("key", "default").
		Limit(1).
		Scan(&out)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return defaultHomeConfig(), nil
	}
	if out.Eyebrow == "" {
		out.Eyebrow = "Editorial"
	}
	if out.Title == "" {
		out.Title = "博客"
	}
	if out.Subtitle == "" {
		out.Subtitle = "想法、笔记与记录, 关于技术、产品与日常的长短文。"
	}
	return out, nil
}

func (p *PG) UpsertHomeConfig(ctx context.Context, cfg *model.HomeConfig) error {
	if cfg == nil {
		cfg = defaultHomeConfig()
	}
	_, err := p.db.Exec(ctx, `INSERT INTO home_config (key, eyebrow, title, subtitle, updated_at)
		VALUES ('default', ?, ?, ?, now())
		ON CONFLICT (key) DO UPDATE SET
			eyebrow = EXCLUDED.eyebrow,
			title = EXCLUDED.title,
			subtitle = EXCLUDED.subtitle,
			updated_at = now()`,
		cfg.Eyebrow, cfg.Title, cfg.Subtitle)
	return err
}
