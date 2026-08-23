// Package server wires the blog-site HTTP routes onto a GoFrame server.
// Shared by cmd/blog and integration tests so they exercise the same wiring.
package server

import (
	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/catalog"
	"github.com/yueli-official/blog/api/internal/controller"
	"github.com/yueli-official/blog/api/internal/runtime"
	foundationauth "github.com/yueli-official/foundation/go/auth"
	"github.com/yueli-official/foundation/go/discovery"
	"github.com/yueli-official/foundation/go/privacy"
	"github.com/yueli-official/foundation/go/urllifecycle"
)

// Deps are the wiring dependencies. Catalog may be nil for a minimal
// health-only server.
type Deps struct {
	Verifier       *foundationauth.Verifier
	Catalog        *catalog.Service
	Discovery      *discovery.Module
	DiscoveryCache *discovery.Cache
	URLResolver    urllifecycle.Resolver
	PrivacyOwner   privacy.OwnerHost
	PrivacyScope   string
	Authorization  *blogauthz.Service
}

// Configure mounts: public health, public browse/detail (optional auth in the
// handlers), and the JWT-protected author API.
func Configure(s *ghttp.Server, d Deps) {
	apiMiddleware := runtime.MustAPIMiddleware(runtime.MustRateLimiterFromEnvironment())
	s.Use(runtime.TraceRouteMiddleware)
	s.Group("/", func(grp *ghttp.RouterGroup) {
		grp.Middleware(apiMiddleware.Handle)
		grp.GET("/healthz", controller.Healthz)
		grp.GET("/readyz", runtime.ReadinessHandler(map[string]runtime.ReadinessCheck{
			"database": runtime.DatabaseReadiness,
		}))
	})

	if d.Catalog == nil {
		return
	}

	// Public browse/detail: enveloped, no mandatory auth (handlers verify the
	// token themselves when present, so an author can preview drafts). Comments
	// post with optional login — logged-in auto-approved, anonymous → pending.
	s.Group("/", func(grp *ghttp.RouterGroup) {
		grp.Middleware(apiMiddleware.Handle)
		grp.Bind(controller.NewPublicHome(d.Catalog))
		grp.Bind(controller.NewPublicPosts(d.Catalog, d.Verifier, d.Discovery))
		if d.DiscoveryCache != nil {
			grp.Bind(controller.NewPublicDiscovery(d.DiscoveryCache))
		}
		if d.URLResolver != nil {
			grp.Bind(controller.NewPublicURLLifecycle(d.URLResolver))
		}
		grp.Bind(controller.NewPublicComments(d.Catalog, d.Verifier))
		grp.Bind(controller.NewPublicSeries(d.Catalog))
		grp.Bind(controller.NewSubscribers(d.Catalog))
		if d.Authorization != nil {
			grp.Bind(controller.NewAuthorizationSetup(d.Authorization))
		}
	})

	// Author API: envelope first, then mandatory JWT.
	s.Group("/", func(grp *ghttp.RouterGroup) {
		grp.Middleware(apiMiddleware.Handle, runtime.RequiredAuth(d.Verifier), controller.AuthorizationMiddleware(d.Authorization))
		grp.Bind(controller.NewHome(d.Catalog))
		grp.Bind(controller.NewDashboard(d.Catalog))
		grp.Bind(controller.NewPosts(d.Catalog))
		grp.Bind(controller.NewSeries(d.Catalog))
		grp.Bind(controller.NewTaxonomy(d.Catalog))
		grp.Bind(controller.NewCover(d.Catalog))
		grp.Bind(controller.NewImages(d.Catalog))
		grp.Bind(controller.NewReactions(d.Catalog))
		grp.Bind(controller.NewComments(d.Catalog))
		grp.Bind(controller.NewAuthorization())
		if d.PrivacyOwner != nil {
			grp.POST("/api/internal/privacy/owner", runtime.OwnerHandler(d.PrivacyOwner, d.PrivacyScope))
		}
	})
}
