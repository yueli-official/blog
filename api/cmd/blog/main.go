// Command blog is the blog site backend: article catalog + browse + authoring,
// layered on the asset service (covers via public delivery).
package main

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"

	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"

	"platform/gokit/authjwt"
	"platform/products/blog/api/internal/appconfig"
	"platform/products/blog/api/internal/catalog"
	"platform/products/blog/api/internal/dao"
	"platform/products/blog/api/internal/identityclient"
	"platform/products/blog/api/internal/server"
)

func main() {
	ctx := gctx.New()

	// ── Catalog logic (DB + asset client for covers) ─────────────────────────
	cat := catalog.New(dao.NewPG(g.DB()), appconfig.BuildAssetClient(ctx), appconfig.CoverCategory(ctx),
		appconfig.BuildMailer(ctx), appconfig.SiteURL(ctx), appconfig.LoadSpamPolicy(ctx))
	// Author display data (name/avatar/cover/bio/social) is resolved from the IdP.
	cat.SetIdentityClient(identityclient.NewHTTP(
		g.Cfg().MustGet(ctx, "blog.identity.baseUrl", "http://localhost:8081").String()))

	// ── JWT verifier (IdP JWKS, lazy) ────────────────────────────────────────
	jw := appconfig.LoadJWKS(ctx)
	verifier, err := authjwt.NewVerifier(authjwt.VerifierConfig{
		Keys: authjwt.NewRemoteKeySource(jw.URL), Issuer: jw.Issuer, Audience: jw.Audience,
	})
	if err != nil {
		panic(err)
	}

	s := g.Server()
	server.Configure(s, server.Deps{Verifier: verifier, Catalog: cat})
	g.Log().Info(ctx, "blog-service starting")
	s.Run()
}
