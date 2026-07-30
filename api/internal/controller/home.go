package controller

import (
	"context"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/catalog"
	"github.com/yueli-official/blog/api/internal/model"
	"github.com/yueli-official/foundation/go/authorization"
)

type PublicHome struct{ svc *catalog.Service }

func NewPublicHome(svc *catalog.Service) *PublicHome { return &PublicHome{svc: svc} }

func (c *PublicHome) GetHomeConfig(ctx context.Context, _ *v1.GetHomeConfigReq) (*v1.GetHomeConfigRes, error) {
	cfg, err := c.svc.GetHomeConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetHomeConfigRes{Config: homeConfigView(cfg)}, nil
}

type Home struct{ svc *catalog.Service }

func NewHome(svc *catalog.Service) *Home { return &Home{svc: svc} }

func (c *Home) UpdateHomeConfig(ctx context.Context, req *v1.UpdateHomeConfigReq) (*v1.UpdateHomeConfigRes, error) {
	if err := requireCapability(
		ctx, blogauthz.CapabilitySiteSettingsManage, blogauthz.RootScopeID,
		authorization.ResourceFacts{},
	); err != nil {
		return nil, err
	}
	cfg, err := c.svc.UpdateHomeConfig(ctx, &model.HomeConfig{
		Eyebrow:         req.Eyebrow,
		Title:           req.Title,
		Subtitle:        req.Subtitle,
		SiteTitle:       req.SiteTitle,
		SiteDescription: req.SiteDescription,
		SupportEmail:    req.SupportEmail,
		FooterTagline:   req.FooterTagline,
		FooterCopyright: req.FooterCopyright,
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateHomeConfigRes{Config: homeConfigView(cfg)}, nil
}

func homeConfigView(cfg *model.HomeConfig) *v1.HomeConfigView {
	if cfg == nil {
		return nil
	}
	return &v1.HomeConfigView{
		Eyebrow:         cfg.Eyebrow,
		Title:           cfg.Title,
		Subtitle:        cfg.Subtitle,
		SiteTitle:       cfg.SiteTitle,
		SiteDescription: cfg.SiteDescription,
		SupportEmail:    cfg.SupportEmail,
		FooterTagline:   cfg.FooterTagline,
		FooterCopyright: cfg.FooterCopyright,
	}
}
