package dao

import (
	"context"

	"github.com/gogf/gf/v2/errors/gerror"

	"github.com/yueli-official/blog/api/internal/model"
)

const tHomeConfig = "home_config"

func (p *PG) GetHomeConfig(ctx context.Context) (*model.HomeConfig, error) {
	var out *model.HomeConfig
	err := p.db.Model(tHomeConfig).Ctx(ctx).
		Fields("eyebrow", "title", "subtitle", "site_title", "site_description", "support_email", "footer_tagline", "footer_copyright").
		Where("key", "default").
		Limit(1).
		Scan(&out)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, gerror.New("blog site configuration is not seeded")
	}
	return out, nil
}

func (p *PG) UpsertHomeConfig(ctx context.Context, cfg *model.HomeConfig) error {
	if cfg == nil {
		return gerror.New("blog site configuration is required")
	}
	_, err := p.db.Exec(ctx, `INSERT INTO home_config (key, eyebrow, title, subtitle, site_title, site_description, support_email, footer_tagline, footer_copyright, updated_at)
		VALUES ('default', ?, ?, ?, ?, ?, ?, ?, ?, now())
		ON CONFLICT (key) DO UPDATE SET
			eyebrow = EXCLUDED.eyebrow,
			title = EXCLUDED.title,
			subtitle = EXCLUDED.subtitle,
			site_title = EXCLUDED.site_title,
			site_description = EXCLUDED.site_description,
			support_email = EXCLUDED.support_email,
			footer_tagline = EXCLUDED.footer_tagline,
			footer_copyright = EXCLUDED.footer_copyright,
			updated_at = now()`,
		cfg.Eyebrow, cfg.Title, cfg.Subtitle, cfg.SiteTitle, cfg.SiteDescription, cfg.SupportEmail, cfg.FooterTagline, cfg.FooterCopyright)
	return err
}
