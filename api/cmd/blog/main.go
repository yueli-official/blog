// Command blog is the blog site backend: article catalog + browse + authoring,
// layered on the asset service (covers via public delivery).
package main

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/yueli-official/foundation/go/traffic"
	trafficpostgres "github.com/yueli-official/foundation/go/traffic/postgres"

	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"

	"platform/gokit/authsetup"
	"platform/gokit/observability"
	"platform/gokit/openapiexport"
	"platform/products/blog/api/internal/appconfig"
	"platform/products/blog/api/internal/blogdiscovery"
	"platform/products/blog/api/internal/blogtraffic"
	"platform/products/blog/api/internal/blogurls"
	"platform/products/blog/api/internal/catalog"
	"platform/products/blog/api/internal/dao"
	"platform/products/blog/api/internal/identityclient"
	"platform/products/blog/api/internal/server"
)

func main() {
	ctx := gctx.New()
	shutdown, err := observability.StartFromEnvironment(ctx, "blog-api")
	if err != nil {
		panic(err)
	}
	defer observability.ShutdownWithTimeout(shutdown)

	// ── Catalog logic (DB + asset client for covers) ─────────────────────────
	store := dao.NewPG(g.DB())
	discoveryModule, discoveryCache, err := blogdiscovery.New(store, appconfig.DiscoveryConfig(ctx))
	if err != nil {
		panic(err)
	}
	legacyTraffic, err := blogtraffic.SnapshotLegacy(ctx, store)
	if err != nil {
		panic(err)
	}
	trafficDB, err := appconfig.OpenTrafficDB(ctx)
	if err != nil {
		panic(err)
	}
	defer trafficDB.Close()
	trafficCatalog, err := traffic.Compile(blogtraffic.Definition(appconfig.TrafficTimeZone(ctx)))
	if err != nil {
		panic(err)
	}
	trafficModule, err := trafficpostgres.New(ctx, trafficCatalog, trafficpostgres.Options{
		DB: trafficDB, InstanceKey: "blog:" + appconfig.SiteSlug(ctx),
		InitialBaselines: legacyTraffic.InitialBaselines,
	})
	if err != nil {
		panic(err)
	}
	if err := blogtraffic.Reconcile(ctx, trafficModule, store, legacyTraffic.Resources); err != nil {
		panic(err)
	}
	var urlLifecycle *blogurls.Lifecycle
	if openapiexport.Requested() {
		urlLifecycle, err = blogurls.NewMemory(appconfig.SiteURL(ctx))
		if err != nil {
			panic(err)
		}
	} else {
		urlLifecycle, err = blogurls.NewPostgres(
			ctx,
			trafficDB,
			"blog:"+appconfig.SiteSlug(ctx),
			appconfig.SiteURL(ctx),
		)
		if err != nil {
			panic(err)
		}
		urlRows, err := store.ListURLLifecycleClaims(ctx)
		if err != nil {
			panic(err)
		}
		urlClaims := make([]blogurls.Claim, 0, len(urlRows))
		for _, row := range urlRows {
			kind := blogurls.PostKind
			switch row.Kind {
			case "category":
				kind = blogurls.CategoryKind
			case "tag":
				kind = blogurls.TagKind
			}
			urlClaims = append(urlClaims, blogurls.Claim{State: blogurls.State{
				ID: row.ID, Kind: kind, Slug: row.Slug, Published: true,
			}})
		}
		if err := urlLifecycle.Reconcile(ctx, urlClaims); err != nil {
			panic(err)
		}
	}

	cat := catalog.New(store, appconfig.BuildAssetClient(ctx), appconfig.CoverCategory(ctx),
		appconfig.BuildMailer(ctx), appconfig.SiteURL(ctx), appconfig.LoadSpamPolicy(ctx))
	cat.SetTraffic(trafficModule)
	cat.SetURLLifecycle(urlLifecycle)
	// Author display data (name/avatar/cover/bio/social) is resolved from the IdP.
	cat.SetIdentityClient(identityclient.NewHTTP(
		g.Cfg().MustGet(ctx, "blog.identity.baseUrl", "http://localhost:8081").String()))

	// ── JWT verifier (IdP JWKS, lazy) ────────────────────────────────────────
	jw := appconfig.LoadJWKS(ctx)
	verifier, err := authsetup.NewRemoteVerifier(authsetup.RemoteVerifierConfig{
		JWKSURL: jw.URL, Issuer: jw.Issuer, Audience: jw.Audience,
		AllowLoopbackHTTP: jw.AllowLoopbackHTTP,
	})
	if err != nil {
		panic(err)
	}

	s := g.Server()
	server.Configure(s, server.Deps{
		Verifier: verifier, Catalog: cat,
		Discovery: discoveryModule, DiscoveryCache: discoveryCache,
		URLResolver: urlLifecycle.Resolver(),
	})
	if handled, err := openapiexport.ExportIfRequested(s); handled {
		if err != nil {
			panic(err)
		}
		return
	}
	g.Log().Info(ctx, "blog-service starting")
	s.Run()
}
