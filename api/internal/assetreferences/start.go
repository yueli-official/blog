package assetreferences

import (
	"context"
	"database/sql"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/yueli-official/asset/referencesync"
	"os"
)

func Start(ctx context.Context, db *sql.DB) (context.CancelFunc, error) {
	cfg := referencesync.Config{
		BaseURL:      g.Cfg().MustGet(ctx, "blog.assetService.baseUrl").String(),
		TokenURL:     g.Cfg().MustGet(ctx, "blog.assetService.tokenUrl").String(),
		ClientID:     g.Cfg().MustGet(ctx, "blog.assetService.clientId", "blog-asset-svc").String(),
		ClientSecret: g.Cfg().MustGet(ctx, "blog.assetService.clientSecret").String(),
	}.WithEnvironment()
	return referencesync.Start(ctx, db, "blog:asset-references", Source(g.Cfg().MustGet(ctx, "blog.siteUrl").String(), os.Getenv("ASSET_PUBLIC_ORIGIN")), cfg, func(err error) { g.Log().Warning(ctx, "asset reference reconciliation:", err) })
}
