// Package server wires the blog-site HTTP routes onto a GoFrame server.
// Shared by cmd/blog and integration tests so they exercise the same wiring.
package server

import (
	"github.com/gogf/gf/v2/net/ghttp"

	"platform/gokit/authjwt"
	"platform/gokit/ghttpx"
	"platform/gokit/healthcheck"
	"platform/products/blog/api/internal/catalog"
	"platform/products/blog/api/internal/controller"
)

// Deps are the wiring dependencies. Catalog may be nil for a minimal
// health-only server.
type Deps struct {
	Verifier *authjwt.Verifier
	Catalog  *catalog.Service
}

// Configure mounts: public health, public browse/detail (optional auth in the
// handlers), and the JWT-protected author API.
func Configure(s *ghttp.Server, d Deps) {
	s.Use(ghttpx.TraceRouteMiddleware)
	s.Group("/", func(grp *ghttp.RouterGroup) {
		grp.Middleware(ghttpx.Middleware)
		grp.GET("/healthz", controller.Healthz)
		grp.GET("/readyz", healthcheck.Handler(map[string]healthcheck.Check{"database": healthcheck.Database}))
	})

	if d.Catalog == nil {
		return
	}

	// Public browse/detail: enveloped, no mandatory auth (handlers verify the
	// token themselves when present, so an author can preview drafts). Comments
	// post with optional login — logged-in auto-approved, anonymous → pending.
	s.Group("/", func(grp *ghttp.RouterGroup) {
		grp.Middleware(ghttpx.Middleware)
		grp.Bind(controller.NewPublicHome(d.Catalog))
		grp.Bind(controller.NewPublicPosts(d.Catalog, d.Verifier))
		grp.Bind(controller.NewPublicComments(d.Catalog, d.Verifier))
		grp.Bind(controller.NewPublicSeries(d.Catalog))
		grp.Bind(controller.NewSubscribers(d.Catalog))
	})

	// Author API: envelope first, then mandatory JWT.
	s.Group("/", func(grp *ghttp.RouterGroup) {
		grp.Middleware(ghttpx.Middleware, authjwt.Middleware(d.Verifier))
		grp.Bind(controller.NewHome(d.Catalog))
		grp.Bind(controller.NewPosts(d.Catalog))
		grp.Bind(controller.NewSeries(d.Catalog))
		grp.Bind(controller.NewTaxonomy(d.Catalog))
		grp.Bind(controller.NewCover(d.Catalog))
		grp.Bind(controller.NewImages(d.Catalog))
		grp.Bind(controller.NewReactions(d.Catalog))
		grp.Bind(controller.NewComments(d.Catalog))
	})
}
