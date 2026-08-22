package appconfig

import (
	"context"
	"testing"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcfg"
)

func TestAssetSiteKeyIsIndependentFromDeploymentSlug(t *testing.T) {
	adapter, err := gcfg.NewAdapterContent("blog:\n  siteSlug: blog-main\n  assetSiteKey: blog\n")
	if err != nil {
		t.Fatal(err)
	}
	previous := g.Cfg().GetAdapter()
	g.Cfg().SetAdapter(adapter)
	t.Cleanup(func() { g.Cfg().SetAdapter(previous) })

	ctx := context.Background()
	if got := SiteSlug(ctx); got != "blog-main" {
		t.Fatalf("SiteSlug = %q, want blog-main", got)
	}
	if got := AssetSiteKey(ctx); got != "blog" {
		t.Fatalf("AssetSiteKey = %q, want blog", got)
	}
}

func TestAssetSiteKeyFallsBackToSiteSlug(t *testing.T) {
	adapter, err := gcfg.NewAdapterContent("blog:\n  siteSlug: legacy-blog\n")
	if err != nil {
		t.Fatal(err)
	}
	previous := g.Cfg().GetAdapter()
	g.Cfg().SetAdapter(adapter)
	t.Cleanup(func() { g.Cfg().SetAdapter(previous) })

	if got := AssetSiteKey(context.Background()); got != "legacy-blog" {
		t.Fatalf("AssetSiteKey = %q, want legacy-blog fallback", got)
	}
}
