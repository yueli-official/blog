// Command blog is the blog site backend: article catalog + browse + authoring,
// layered on the asset service (covers via public delivery).
package main

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	foundationabuse "github.com/yueli-official/foundation/go/abuse"
	"github.com/yueli-official/foundation/go/abuse/turnstile"
	"github.com/yueli-official/foundation/go/authorization"
	authorizationpostgres "github.com/yueli-official/foundation/go/authorization/postgres"
	"github.com/yueli-official/foundation/go/privacy"
	"github.com/yueli-official/foundation/go/traffic"
	trafficpostgres "github.com/yueli-official/foundation/go/traffic/postgres"

	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"

	"platform/gokit/authsetup"
	"platform/gokit/observability"
	"platform/gokit/openapiexport"
	"platform/products/blog/api/internal/appconfig"
	"platform/products/blog/api/internal/blogabuse"
	"platform/products/blog/api/internal/blogauthz"
	"platform/products/blog/api/internal/blogdiscovery"
	"platform/products/blog/api/internal/blogprivacy"
	"platform/products/blog/api/internal/blogsearch"
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

	spamPolicy := appconfig.LoadSpamPolicy(ctx)
	cat := catalog.New(store, appconfig.BuildAssetClient(ctx), appconfig.CoverCategory(ctx),
		appconfig.BuildMailer(ctx), appconfig.SiteURL(ctx), spamPolicy)
	cat.SetTraffic(trafficModule)
	var privacyOwner privacy.OwnerHost
	if !openapiexport.Requested() {
		privacyService, err := blogprivacy.NewPostgres(
			ctx, trafficDB, "blog:"+appconfig.SiteSlug(ctx),
			privacy.OwnerKey("site."+appconfig.SiteSlug(ctx)),
		)
		if err != nil {
			panic(err)
		}
		if err := privacyService.ReconcileNewsletter(ctx); err != nil {
			panic(err)
		}
		cat.SetPrivacy(privacyService)
		privacyOwner = privacyService.OwnerHost()
	}
	var (
		abuseChallenge *foundationabuse.ChallengeDefinition
		abuseVerifiers map[foundationabuse.ChallengeKind]foundationabuse.ChallengeVerifier
	)
	if secret := g.Cfg().MustGet(ctx, "blog.abuse.turnstile.secret").String(); secret != "" && !openapiexport.Requested() {
		hostnames := g.Cfg().MustGet(ctx, "blog.abuse.turnstile.hostnames").Strings()
		if len(hostnames) == 0 {
			panic("blog.abuse.turnstile.hostnames is required when Turnstile is enabled")
		}
		challengeVerifier, err := turnstile.New(turnstile.Options{
			Secret:   secret,
			Endpoint: g.Cfg().MustGet(ctx, "blog.abuse.turnstile.endpoint").String(),
		})
		if err != nil {
			panic(err)
		}
		abuseChallenge = &foundationabuse.ChallengeDefinition{
			Kind: "turnstile", ExpectedAction: "blog-comment",
			AllowedHosts: hostnames,
		}
		abuseVerifiers = map[foundationabuse.ChallengeKind]foundationabuse.ChallengeVerifier{
			"turnstile": challengeVerifier,
		}
	}
	abuseCatalog := foundationabuse.MustCompile(blogabuse.Definition(blogabuse.Policy{
		AnonymousCapacity: int64(spamPolicy.RatePerWindow),
		Window:            time.Duration(spamPolicy.RateWindowSeconds) * time.Second,
		Challenge:         abuseChallenge,
	}))
	var abuseModule foundationabuse.Module
	if openapiexport.Requested() {
		abuseModule, err = foundationabuse.NewMemory(abuseCatalog, foundationabuse.MemoryOptions{
			Secret:    []byte("blog-openapi-abuse-memory-secret"),
			Verifiers: abuseVerifiers,
		})
	} else {
		abuseModule, err = foundationabuse.NewPostgres(ctx, abuseCatalog, foundationabuse.PostgresOptions{
			DB: trafficDB, InstanceKey: "blog:" + appconfig.SiteSlug(ctx),
			Verifiers: abuseVerifiers,
		})
	}
	if err != nil {
		panic(err)
	}
	cat.SetAbuse(abuseModule)
	cat.SetURLLifecycle(urlLifecycle)
	var searchIndex *blogsearch.Index
	if openapiexport.Requested() {
		searchIndex = blogsearch.NewMemory()
	} else {
		searchIndex, err = blogsearch.NewPostgres(ctx, trafficDB, appconfig.SiteSlug(ctx))
		if err != nil {
			panic(err)
		}
		if err := searchIndex.Reconcile(ctx, trafficDB); err != nil {
			panic(err)
		}
	}
	cat.SetSearch(searchIndex)
	// Author display data (name/avatar/cover/bio/social) is resolved from the IdP.
	cat.SetIdentityClient(identityclient.NewHTTP(
		g.Cfg().MustGet(ctx, "blog.identity.baseUrl", "http://localhost:8081").String()))

	definition, err := authorization.Compile(blogauthz.Definition())
	if err != nil {
		panic(err)
	}
	var authorizationService *blogauthz.Service
	if openapiexport.Requested() {
		authz, err := authorization.NewMemory(definition, authorization.MemoryOptions{
			RootScopeID: blogauthz.RootScopeID,
			ProtectedSubjects: []authorization.SubjectRef{{
				Kind: authorization.SubjectUser, ID: "openapi-export-admin",
			}},
			Constraints: blogauthz.ConstraintEvaluators(),
			Predicates:  blogauthz.PredicateEvaluators(),
		})
		if err != nil {
			panic(err)
		}
		authorizationService = blogauthz.New(authz, nil)
	} else {
		bootstrapSubs := appconfig.BootstrapAdministratorSubs(ctx)
		protected := make([]authorization.SubjectRef, 0, len(bootstrapSubs))
		for _, sub := range bootstrapSubs {
			if sub != "" {
				protected = append(protected, authorization.SubjectRef{
					Kind: authorization.SubjectUser, ID: sub,
				})
			}
		}
		authz, err := authorizationpostgres.New(ctx, definition, authorizationpostgres.Options{
			DB: trafficDB, InstanceKey: "blog:" + appconfig.SiteSlug(ctx),
			Memory: authorization.MemoryOptions{
				RootScopeID: blogauthz.RootScopeID, ProtectedSubjects: protected,
				Constraints: blogauthz.ConstraintEvaluators(),
				Predicates:  blogauthz.PredicateEvaluators(),
			},
		})
		if err != nil {
			panic(err)
		}
		if authz.InstanceWasCreated() {
			if len(protected) == 0 {
				panic("blog authorization bootstrap requires at least one administrator subject")
			}
			if err := blogauthz.SyncResourceScopes(ctx, trafficDB, authz); err != nil {
				panic(err)
			}
		}
		authorizationService = blogauthz.New(authz, trafficDB)
	}

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
		Verifier: verifier, Catalog: cat, Authorization: authorizationService,
		Discovery: discoveryModule, DiscoveryCache: discoveryCache,
		URLResolver:  urlLifecycle.Resolver(),
		PrivacyOwner: privacyOwner, PrivacyScope: "privacy:owner",
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
