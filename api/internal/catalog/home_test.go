package catalog

import (
	"testing"

	"platform/products/blog/api/internal/model"
)

func TestNormalizeHomeConfigIncludesSiteAndFooter(t *testing.T) {
	got := normalizeHomeConfig(&model.HomeConfig{
		Eyebrow:         "  Notes  ",
		Title:           "  工程博客  ",
		Subtitle:        "  长文章与短记录  ",
		SiteTitle:       "  Yueli Blog  ",
		SiteDescription: "  技术与产品  ",
		SupportEmail:    "  blog@example.com  ",
		FooterTagline:   "  持续记录  ",
		FooterCopyright: "  © 2026 Yueli  ",
	})

	if got.SiteTitle != "Yueli Blog" || got.SupportEmail != "blog@example.com" {
		t.Fatalf("site settings were not normalized: %#v", got)
	}
	if got.FooterTagline != "持续记录" || got.FooterCopyright != "© 2026 Yueli" {
		t.Fatalf("footer settings were not normalized: %#v", got)
	}
}
