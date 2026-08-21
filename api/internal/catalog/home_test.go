package catalog

import (
	"testing"

	"github.com/yueli-official/blog/api/internal/model"
)

func TestNormalizeHomeConfigIncludesSiteAndFooter(t *testing.T) {
	got, err := normalizeHomeConfig(&model.HomeConfig{
		Eyebrow:           "  Notes  ",
		Title:             "  工程博客  ",
		Subtitle:          "  长文章与短记录  ",
		SiteTitle:         "  Yueli Blog  ",
		SiteDescription:   "  技术与产品  ",
		SupportEmail:      "  blog@example.com  ",
		FooterTagline:     "  持续记录  ",
		FooterCopyright:   "  © 2026 Yueli  ",
		CoverAspectWidth:  12,
		CoverAspectHeight: 8,
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
	if got.CoverAspectWidth != 3 || got.CoverAspectHeight != 2 {
		t.Fatalf("cover ratio was not normalized: %#v", got)
	}
}

func TestNormalizeHomeConfigDefaultsAndValidatesCoverAspect(t *testing.T) {
	base := &model.HomeConfig{
		Eyebrow: "Notes", Title: "工程博客", Subtitle: "长文章与短记录",
		SiteTitle: "Yueli Blog", SiteDescription: "技术与产品",
		FooterTagline: "持续记录", FooterCopyright: "© 2026 Yueli",
	}
	got, err := normalizeHomeConfig(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.CoverAspectWidth != 3 || got.CoverAspectHeight != 2 {
		t.Fatalf("cover ratio default = %d:%d, want 3:2", got.CoverAspectWidth, got.CoverAspectHeight)
	}
	base.FooterCopyright = ""
	if _, err := normalizeHomeConfig(base); err != nil {
		t.Fatalf("optional footer copyright was rejected: %v", err)
	}

	base.CoverAspectWidth = 3
	base.CoverAspectHeight = 0
	if _, err := normalizeHomeConfig(base); err == nil {
		t.Fatal("expected incomplete cover ratio to be rejected")
	}
}
