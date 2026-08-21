package catalog

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/errors/gerror"

	"github.com/yueli-official/blog/api/internal/model"
)

func (s *Service) GetHomeConfig(ctx context.Context) (*model.HomeConfig, error) {
	return s.dao.GetHomeConfig(ctx)
}

func (s *Service) UpdateHomeConfig(ctx context.Context, cfg *model.HomeConfig) (*model.HomeConfig, error) {
	next, err := normalizeHomeConfig(cfg)
	if err != nil {
		return nil, err
	}
	if err := s.dao.UpsertHomeConfig(ctx, next); err != nil {
		return nil, err
	}
	return s.dao.GetHomeConfig(ctx)
}

func normalizeHomeConfig(in *model.HomeConfig) (*model.HomeConfig, error) {
	if in == nil {
		return nil, gerror.New("blog site configuration is required")
	}
	out := &model.HomeConfig{
		Eyebrow: strings.TrimSpace(in.Eyebrow), Title: strings.TrimSpace(in.Title), Subtitle: strings.TrimSpace(in.Subtitle),
		SiteTitle: strings.TrimSpace(in.SiteTitle), SiteDescription: strings.TrimSpace(in.SiteDescription),
		SupportEmail: strings.TrimSpace(in.SupportEmail), FooterTagline: strings.TrimSpace(in.FooterTagline),
		FooterCopyright:  strings.TrimSpace(in.FooterCopyright),
		CoverAspectWidth: in.CoverAspectWidth, CoverAspectHeight: in.CoverAspectHeight,
	}
	if out.Eyebrow == "" || out.Title == "" || out.Subtitle == "" || out.SiteTitle == "" || out.SiteDescription == "" || out.FooterTagline == "" {
		return nil, gerror.New("blog homepage, site, and footer content must be configured")
	}
	if out.CoverAspectWidth == 0 && out.CoverAspectHeight == 0 {
		out.CoverAspectWidth, out.CoverAspectHeight = 3, 2
	}
	if out.CoverAspectWidth < 1 || out.CoverAspectHeight < 1 || out.CoverAspectWidth > 100 || out.CoverAspectHeight > 100 {
		return nil, gerror.New("blog cover aspect ratio must use width and height between 1 and 100")
	}
	divisor := greatestCommonDivisor(out.CoverAspectWidth, out.CoverAspectHeight)
	out.CoverAspectWidth /= divisor
	out.CoverAspectHeight /= divisor
	return out, nil
}

func greatestCommonDivisor(left, right int) int {
	for right != 0 {
		left, right = right, left%right
	}
	return left
}
