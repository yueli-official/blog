package catalog

import (
	"context"
	"strings"

	"platform/products/blog/api/internal/model"
)

const (
	defaultHomeEyebrow  = "Editorial"
	defaultHomeTitle    = "博客"
	defaultHomeSubtitle = "想法、笔记与记录, 关于技术、产品与日常的长短文。"
)

func (s *Service) GetHomeConfig(ctx context.Context) (*model.HomeConfig, error) {
	return s.dao.GetHomeConfig(ctx)
}

func (s *Service) UpdateHomeConfig(ctx context.Context, cfg *model.HomeConfig) (*model.HomeConfig, error) {
	next := normalizeHomeConfig(cfg)
	if err := s.dao.UpsertHomeConfig(ctx, next); err != nil {
		return nil, err
	}
	return s.dao.GetHomeConfig(ctx)
}

func normalizeHomeConfig(in *model.HomeConfig) *model.HomeConfig {
	out := &model.HomeConfig{
		Eyebrow:         defaultHomeEyebrow,
		Title:           defaultHomeTitle,
		Subtitle:        defaultHomeSubtitle,
		SiteDescription: "想法、笔记与记录",
		FooterTagline:   "想法、笔记与记录",
	}
	if in == nil {
		return out
	}
	if v := strings.TrimSpace(in.Eyebrow); v != "" {
		out.Eyebrow = v
	}
	if v := strings.TrimSpace(in.Title); v != "" {
		out.Title = v
	}
	if v := strings.TrimSpace(in.Subtitle); v != "" {
		out.Subtitle = v
	}
	out.SiteTitle = strings.TrimSpace(in.SiteTitle)
	out.SiteDescription = strings.TrimSpace(in.SiteDescription)
	out.SupportEmail = strings.TrimSpace(in.SupportEmail)
	out.FooterTagline = strings.TrimSpace(in.FooterTagline)
	out.FooterCopyright = strings.TrimSpace(in.FooterCopyright)
	return out
}
