package catalog

import (
	"context"
	"net/url"
	"strings"
	"unicode/utf8"

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
		FooterCopyright: strings.TrimSpace(in.FooterCopyright),
		FriendLinks:     make([]model.FriendLink, 0, len(in.FriendLinks)),
		ContactLinks:    make([]model.ContactLink, 0, len(in.ContactLinks)),
	}
	if out.Eyebrow == "" || out.Title == "" || out.Subtitle == "" || out.SiteTitle == "" || out.SiteDescription == "" || out.FooterTagline == "" {
		return nil, gerror.New("blog homepage, site, and footer content must be configured")
	}
	if len(in.FriendLinks) > 24 {
		return nil, gerror.New("blog footer supports at most 24 friend links")
	}
	seenURLs := make(map[string]struct{}, len(in.FriendLinks))
	for _, item := range in.FriendLinks {
		link := model.FriendLink{
			Label: strings.TrimSpace(item.Label),
			URL:   strings.TrimSpace(item.URL),
		}
		if link.Label == "" || utf8.RuneCountInString(link.Label) > 40 {
			return nil, gerror.New("friend link labels must use 1 to 40 characters")
		}
		parsed, err := url.Parse(link.URL)
		if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, gerror.New("friend link URLs must use a public HTTP or HTTPS address")
		}
		parsed.Scheme = strings.ToLower(parsed.Scheme)
		parsed.Host = strings.ToLower(parsed.Host)
		link.URL = parsed.String()
		duplicateKey := strings.TrimSuffix(link.URL, "/")
		if _, exists := seenURLs[duplicateKey]; exists {
			return nil, gerror.New("friend link URLs must be unique")
		}
		seenURLs[duplicateKey] = struct{}{}
		out.FriendLinks = append(out.FriendLinks, link)
	}
	if len(in.ContactLinks) > 12 {
		return nil, gerror.New("blog footer supports at most 12 contact links")
	}
	for _, item := range in.ContactLinks {
		link := model.ContactLink{
			Value: strings.TrimSpace(item.Value),
			URL:   strings.TrimSpace(item.URL),
		}
		if link.Value == "" || utf8.RuneCountInString(link.Value) > 80 {
			return nil, gerror.New("contact values must use 1 to 80 characters")
		}
		if utf8.RuneCountInString(link.URL) > 2048 {
			return nil, gerror.New("contact URLs must use at most 2048 characters")
		}
		if link.URL != "" {
			parsed, err := url.Parse(link.URL)
			validWeb := err == nil && parsed.Host != "" && parsed.User == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
			validEmail := err == nil && parsed.Scheme == "mailto" && (parsed.Opaque != "" || parsed.Path != "")
			if !validWeb && !validEmail {
				return nil, gerror.New("contact URLs must use HTTP, HTTPS, or mailto")
			}
			parsed.Scheme = strings.ToLower(parsed.Scheme)
			parsed.Host = strings.ToLower(parsed.Host)
			link.URL = parsed.String()
		}
		out.ContactLinks = append(out.ContactLinks, link)
	}
	return out, nil
}
