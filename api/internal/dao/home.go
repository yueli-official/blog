package dao

import (
	"context"
	"encoding/json"

	"github.com/gogf/gf/v2/errors/gerror"

	"github.com/yueli-official/blog/api/internal/model"
)

const tHomeConfig = "home_config"

func (p *PG) GetHomeConfig(ctx context.Context) (*model.HomeConfig, error) {
	var out *model.HomeConfig
	err := p.db.Model(tHomeConfig).Ctx(ctx).
		Fields("eyebrow", "title", "subtitle", "site_title", "site_description", "support_email", "footer_tagline", "footer_copyright", "friend_links", "contact_links").
		Where("key", "default").
		Limit(1).
		Scan(&out)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, gerror.New("blog site configuration is not seeded")
	}
	if out.FriendLinksJSON == "" {
		out.FriendLinksJSON = "[]"
	}
	if err := json.Unmarshal([]byte(out.FriendLinksJSON), &out.FriendLinks); err != nil {
		return nil, gerror.Wrap(err, "decode blog friend links")
	}
	if out.ContactLinksJSON == "" {
		out.ContactLinksJSON = "[]"
	}
	if err := json.Unmarshal([]byte(out.ContactLinksJSON), &out.ContactLinks); err != nil {
		return nil, gerror.Wrap(err, "decode blog contact links")
	}
	return out, nil
}

func (p *PG) UpsertHomeConfig(ctx context.Context, cfg *model.HomeConfig) error {
	if cfg == nil {
		return gerror.New("blog site configuration is required")
	}
	friendLinks, err := json.Marshal(cfg.FriendLinks)
	if err != nil {
		return gerror.Wrap(err, "encode blog friend links")
	}
	contactLinks, err := json.Marshal(cfg.ContactLinks)
	if err != nil {
		return gerror.Wrap(err, "encode blog contact links")
	}
	_, err = p.db.Exec(ctx, `INSERT INTO home_config (key, eyebrow, title, subtitle, site_title, site_description, support_email, footer_tagline, footer_copyright, friend_links, contact_links, updated_at)
		VALUES ('default', ?, ?, ?, ?, ?, ?, ?, ?, CAST(? AS jsonb), CAST(? AS jsonb), now())
		ON CONFLICT (key) DO UPDATE SET
			eyebrow = EXCLUDED.eyebrow,
			title = EXCLUDED.title,
			subtitle = EXCLUDED.subtitle,
			site_title = EXCLUDED.site_title,
			site_description = EXCLUDED.site_description,
			support_email = EXCLUDED.support_email,
			footer_tagline = EXCLUDED.footer_tagline,
			footer_copyright = EXCLUDED.footer_copyright,
			friend_links = EXCLUDED.friend_links,
			contact_links = EXCLUDED.contact_links,
			updated_at = now()`,
		cfg.Eyebrow, cfg.Title, cfg.Subtitle, cfg.SiteTitle, cfg.SiteDescription, cfg.SupportEmail, cfg.FooterTagline, cfg.FooterCopyright, string(friendLinks), string(contactLinks))
	return err
}
