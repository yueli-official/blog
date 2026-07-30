// Command blog is the blog site backend: article catalog + browse + authoring,
// layered on the asset service (covers via public delivery).
package main

import (
	"context"
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

	"github.com/yueli-official/blog/api/internal/appconfig"
	"github.com/yueli-official/blog/api/internal/blogabuse"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/blogclient"
	"github.com/yueli-official/blog/api/internal/blogdiscovery"
	"github.com/yueli-official/blog/api/internal/blogprivacy"
	"github.com/yueli-official/blog/api/internal/blogsearch"
	"github.com/yueli-official/blog/api/internal/blogtraffic"
	"github.com/yueli-official/blog/api/internal/blogurls"
	"github.com/yueli-official/blog/api/internal/catalog"
	"github.com/yueli-official/blog/api/internal/dao"
	"github.com/yueli-official/blog/api/internal/identityclient"
	"github.com/yueli-official/blog/api/internal/runtime"
	"github.com/yueli-official/blog/api/internal/server"
)

func main() {
	if err := runtime.EnableEnvironmentConfig(); err != nil {
		panic(err)
	}
	ctx := gctx.New()
	if runtime.OpenAPIRequested() {
		exportOpenAPI(ctx)
		return
	}
	shutdown, err := runtime.StartTelemetry(ctx, "blog-api")
	if err != nil {
		panic(err)
	}
	defer runtime.ShutdownTelemetry(shutdown)

	// ── Catalog logic (DB + asset client for covers) ─────────────────────────
	store := dao.NewPG(g.DB())
	discoveryModule, discoveryCache, err := blogdiscovery.New(store, appconfig.DiscoveryConfig(ctx))
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
	})
	if err != nil {
		panic(err)
	}
	if err := blogtraffic.ReconcileProjections(ctx, trafficModule, store); err != nil {
		panic(err)
	}
	urlLifecycle, err := blogurls.NewPostgres(
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

	spamPolicy := appconfig.LoadSpamPolicy(ctx)
	cat := catalog.New(store, appconfig.BuildAssetClient(ctx), appconfig.CoverCategory(ctx),
		appconfig.BuildMailer(ctx), appconfig.SiteURL(ctx), spamPolicy)
	cat.SetTraffic(trafficModule)
	privacyService, err := blogprivacy.NewPostgres(
		ctx, trafficDB, "blog:"+appconfig.SiteSlug(ctx),
		privacy.OwnerKey("site."+appconfig.SiteSlug(ctx)),
	)
	if err != nil {
		panic(err)
	}
	cat.SetPrivacy(privacyService)
	privacyOwner := privacyService.OwnerHost()
	var (
		abuseChallenge *foundationabuse.ChallengeDefinition
		abuseVerifiers map[foundationabuse.ChallengeKind]foundationabuse.ChallengeVerifier
	)
	if secret := g.Cfg().MustGet(ctx, "blog.abuse.turnstile.secret").String(); secret != "" {
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
	abuseModule, err := foundationabuse.NewPostgres(ctx, abuseCatalog, foundationabuse.PostgresOptions{
		DB: trafficDB, InstanceKey: "blog:" + appconfig.SiteSlug(ctx),
		Verifiers: abuseVerifiers,
	})
	if err != nil {
		panic(err)
	}
	cat.SetAbuse(abuseModule)
	cat.SetURLLifecycle(urlLifecycle)
	searchIndex, err := blogsearch.NewPostgres(ctx, trafficDB, appconfig.SiteSlug(ctx))
	if err != nil {
		panic(err)
	}
	if err := searchIndex.Reconcile(ctx, trafficDB); err != nil {
		panic(err)
	}
	cat.SetSearch(searchIndex)
	// Author display data (name/avatar/cover/bio/social) is resolved from the IdP.
	cat.SetIdentityClient(identityclient.NewHTTP(
		g.Cfg().MustGet(ctx, "blog.identity.baseUrl", "http://localhost:8081").String()))

	definition, err := authorization.Compile(blogauthz.Definition())
	if err != nil {
		panic(err)
	}
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
	authorizationService := blogauthz.New(authz, trafficDB)

	// ── JWT verifier (IdP JWKS, lazy) ────────────────────────────────────────
	jw := appconfig.LoadJWKS(ctx)
	verifier, err := runtime.NewRemoteVerifier(runtime.RemoteVerifierConfig{
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
	if handled, err := runtime.ExportOpenAPIIfRequested(s); handled {
		if err != nil {
			panic(err)
		}
		return
	}
	g.Log().Info(ctx, "blog-service starting")
	s.Run()
}

func exportOpenAPI(ctx context.Context) {
	discoveryModule, discoveryCache, err := blogdiscovery.New(nil, blogdiscovery.Config{
		Origin: "https://blog.example.test", Name: "Blog",
		Description: "OpenAPI export", Locale: "zh-CN", TTL: 5 * time.Minute, Clock: time.Now,
	})
	if err != nil {
		panic(err)
	}
	trafficCatalog, err := traffic.Compile(blogtraffic.Definition("UTC"))
	if err != nil {
		panic(err)
	}
	trafficModule, err := traffic.NewMemory(trafficCatalog, traffic.MemoryOptions{
		Clock: time.Now, Secret: []byte("blog-openapi-traffic-secret-32-bytes"),
	})
	if err != nil {
		panic(err)
	}
	urlLifecycle, err := blogurls.NewMemory("https://blog.example.test")
	if err != nil {
		panic(err)
	}
	cat := catalog.New(
		nil, blogclient.NewFake(), "blog-cover", nil,
		"https://blog.example.test", catalog.SpamPolicy{},
	)
	cat.SetTraffic(trafficModule)
	cat.SetURLLifecycle(urlLifecycle)
	cat.SetSearch(blogsearch.NewMemory())
	cat.SetIdentityClient(identityclient.NewHTTP("https://identity.example.test"))

	abuseCatalog := foundationabuse.MustCompile(blogabuse.Definition(blogabuse.Policy{
		AnonymousCapacity: 5, Window: time.Minute,
	}))
	abuseModule, err := foundationabuse.NewMemory(abuseCatalog, foundationabuse.MemoryOptions{
		Secret: []byte("blog-openapi-abuse-memory-secret"),
	})
	if err != nil {
		panic(err)
	}
	cat.SetAbuse(abuseModule)

	definition, err := authorization.Compile(blogauthz.Definition())
	if err != nil {
		panic(err)
	}
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
	authorizationService := blogauthz.New(authz, nil)

	s := g.Server()
	server.Configure(s, server.Deps{
		Catalog: cat, Authorization: authorizationService,
		Discovery: discoveryModule, DiscoveryCache: discoveryCache,
		URLResolver: urlLifecycle.Resolver(),
	})
	handled, err := runtime.ExportOpenAPIIfRequested(s)
	if err != nil {
		panic(err)
	}
	if !handled {
		panic("BLOG_OPENAPI_OUTPUT is required")
	}
}
