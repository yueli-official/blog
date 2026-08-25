package catalog

import (
	"fmt"
	"testing"

	"github.com/yueli-official/blog/api/internal/model"
)

func TestNormalizeHomeConfigIncludesSiteAndFooter(t *testing.T) {
	got, err := normalizeHomeConfig(&model.HomeConfig{
		Eyebrow:         "  Notes  ",
		Title:           "  工程博客  ",
		Subtitle:        "  长文章与短记录  ",
		SiteTitle:       "  Yueli Blog  ",
		SiteDescription: "  技术与产品  ",
		SupportEmail:    "  blog@example.com  ",
		FooterTagline:   "  持续记录  ",
		FooterCopyright: "  © 2026 Yueli  ",
		FriendLinks: []model.FriendLink{
			{Label: " 月离官网 ", URL: " HTTPS://YUELI.EXAMPLE.COM/ "},
			{Label: "设计系统", URL: "https://design.example.com"},
		},
		ContactLinks: []model.ContactLink{
			{Value: " blog@example.com ", URL: " MAILTO:blog@example.com "},
			{Value: "123456789", URL: " HTTPS://QM.QQ.COM/example "},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got.SiteTitle != "Yueli Blog" || got.SupportEmail != "blog@example.com" {
		t.Fatalf("site settings were not normalized: %#v", got)
	}
	if got.FooterTagline != "持续记录" || got.FooterCopyright != "© 2026 Yueli" {
		t.Fatalf("footer settings were not normalized: %#v", got)
	}
	if len(got.FriendLinks) != 2 || got.FriendLinks[0].Label != "月离官网" || got.FriendLinks[0].URL != "https://yueli.example.com/" {
		t.Fatalf("friend links were not normalized in display order: %#v", got.FriendLinks)
	}
	if len(got.ContactLinks) != 2 || got.ContactLinks[0].Value != "blog@example.com" || got.ContactLinks[0].URL != "mailto:blog@example.com" || got.ContactLinks[1].URL != "https://qm.qq.com/example" {
		t.Fatalf("contact links were not normalized in display order: %#v", got.ContactLinks)
	}
}

func TestNormalizeHomeConfigRejectsInvalidContactLinks(t *testing.T) {
	base := func() *model.HomeConfig {
		return &model.HomeConfig{
			Eyebrow: "Notes", Title: "工程博客", Subtitle: "长文章与短记录",
			SiteTitle: "Yueli Blog", SiteDescription: "技术与产品",
			FooterTagline: "持续记录", FooterCopyright: "© 2026 Yueli",
		}
	}

	cases := []struct {
		name  string
		links []model.ContactLink
	}{
		{name: "empty value", links: []model.ContactLink{{URL: "https://example.com"}}},
		{name: "non contact URL", links: []model.ContactLink{{Value: "点击", URL: "javascript:alert(1)"}}},
		{name: "credential URL", links: []model.ContactLink{{Value: "点击", URL: "https://user:pass@example.com"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := base()
			input.ContactLinks = tc.links
			if _, err := normalizeHomeConfig(input); err == nil {
				t.Fatalf("expected contact links to be rejected: %#v", tc.links)
			}
		})
	}

	tooMany := base()
	for range 13 {
		tooMany.ContactLinks = append(tooMany.ContactLinks, model.ContactLink{
			Value: fmt.Sprintf("%d", len(tooMany.ContactLinks)),
		})
	}
	if _, err := normalizeHomeConfig(tooMany); err == nil {
		t.Fatal("expected more than 12 contact links to be rejected")
	}
}

func TestNormalizeHomeConfigRejectsInvalidFriendLinks(t *testing.T) {
	base := func() *model.HomeConfig {
		return &model.HomeConfig{
			Eyebrow: "Notes", Title: "工程博客", Subtitle: "长文章与短记录",
			SiteTitle: "Yueli Blog", SiteDescription: "技术与产品",
			FooterTagline: "持续记录", FooterCopyright: "© 2026 Yueli",
		}
	}

	cases := []struct {
		name  string
		links []model.FriendLink
	}{
		{name: "empty label", links: []model.FriendLink{{URL: "https://example.com"}}},
		{name: "non web URL", links: []model.FriendLink{{Label: "危险", URL: "javascript:alert(1)"}}},
		{name: "credential URL", links: []model.FriendLink{{Label: "私密", URL: "https://user:pass@example.com"}}},
		{name: "duplicate URL", links: []model.FriendLink{
			{Label: "A", URL: "https://example.com"},
			{Label: "B", URL: "https://example.com/"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := base()
			input.FriendLinks = tc.links
			if _, err := normalizeHomeConfig(input); err == nil {
				t.Fatalf("expected friend links to be rejected: %#v", tc.links)
			}
		})
	}

	tooMany := base()
	for range 25 {
		index := len(tooMany.FriendLinks)
		tooMany.FriendLinks = append(tooMany.FriendLinks, model.FriendLink{
			Label: "站点", URL: fmt.Sprintf("https://example.com/%d", index),
		})
	}
	if _, err := normalizeHomeConfig(tooMany); err == nil {
		t.Fatal("expected more than 24 friend links to be rejected")
	}
}

func TestNormalizeHomeConfigAllowsOptionalFooterCopyright(t *testing.T) {
	base := &model.HomeConfig{
		Eyebrow: "Notes", Title: "工程博客", Subtitle: "长文章与短记录",
		SiteTitle: "Yueli Blog", SiteDescription: "技术与产品",
		FooterTagline: "持续记录", FooterCopyright: "© 2026 Yueli",
	}
	base.FooterCopyright = ""
	if _, err := normalizeHomeConfig(base); err != nil {
		t.Fatalf("optional footer copyright was rejected: %v", err)
	}
}
