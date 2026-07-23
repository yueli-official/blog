// Package appconfig builds runtime objects from the GoFrame config
// (manifest/config/config.yaml + GF_* env overrides).
package appconfig

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	_ "github.com/lib/pq"

	"platform/gokit/mail"
	"platform/products/blog/api/internal/blogclient"
	"platform/products/blog/api/internal/blogdiscovery"
	"platform/products/blog/api/internal/catalog"
)

// OpenTrafficDB opens the standard-library PostgreSQL handle required by the
// Foundation Traffic Adapter. It points at the same consumer-owned database as
// GoFrame; traffic remains instance-local and has no central service.
func OpenTrafficDB(ctx context.Context) (*sql.DB, error) {
	host := g.Cfg().MustGet(ctx, "database.default.host").String()
	port := g.Cfg().MustGet(ctx, "database.default.port", "5432").String()
	name := g.Cfg().MustGet(ctx, "database.default.name").String()
	user := g.Cfg().MustGet(ctx, "database.default.user").String()
	password := g.Cfg().MustGet(ctx, "database.default.pass").String()
	if host == "" || name == "" || user == "" {
		return nil, fmt.Errorf("database.default host, name, and user are required")
	}
	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   name,
	}
	query := dsn.Query()
	query.Set("sslmode", g.Cfg().MustGet(ctx, "database.default.sslmode", "disable").String())
	dsn.RawQuery = query.Encode()
	db, err := sql.Open("postgres", dsn.String())
	if err != nil {
		return nil, fmt.Errorf("open traffic database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping traffic database: %w", err)
	}
	return db, nil
}

// TrafficTimeZone is immutable after the traffic instance is initialized.
func TrafficTimeZone(ctx context.Context) string {
	return g.Cfg().MustGet(ctx, "blog.traffic.timeZone", "Asia/Shanghai").String()
}

// BuildAssetClient constructs the HTTP asset-service client (covers via public
// delivery; no service token needed since covers are unsigned).
func BuildAssetClient(ctx context.Context) blogclient.Client {
	base := g.Cfg().MustGet(ctx, "blog.assetService.baseUrl").String()
	return blogclient.NewHTTP(base, SiteSlug(ctx), AssetSpace(ctx))
}

// SiteSlug is the stable deployment-instance identity used for shared-service
// isolation. It comes from trusted server config, never from browser input.
func SiteSlug(ctx context.Context) string {
	return g.Cfg().MustGet(ctx, "blog.siteSlug", "blog").String()
}

// AssetSpace is the physical asset-pool key assigned to this deployment.
func AssetSpace(ctx context.Context) string {
	return g.Cfg().MustGet(ctx, "blog.assetSpace", "default").String()
}

// CoverCategory returns the asset-service category for cover images.
func CoverCategory(ctx context.Context) string {
	return g.Cfg().MustGet(ctx, "blog.coverCategory", "blog-cover").String()
}

// SiteURL is the public base for newsletter confirm/unsubscribe links.
func SiteURL(ctx context.Context) string {
	return g.Cfg().MustGet(ctx, "blog.siteUrl", "http://localhost:3002").String()
}

func DiscoveryConfig(ctx context.Context) blogdiscovery.Config {
	origin := strings.TrimRight(SiteURL(ctx), "/")
	return blogdiscovery.Config{
		Origin:      origin,
		Name:        g.Cfg().MustGet(ctx, "blog.siteBrand", "博客").String(),
		Description: g.Cfg().MustGet(ctx, "blog.siteDescription", "想法、笔记与记录").String(),
		Locale:      g.Cfg().MustGet(ctx, "blog.locale", "zh-CN").String(),
		TTL:         g.Cfg().MustGet(ctx, "blog.discovery.ttl", 5*time.Minute).Duration(),
		Clock:       time.Now,
	}
}

// BuildMailer returns the newsletter transport: SMTP when blog.mailer.mode is
// "smtp", else a dev logger (default; offline-safe).
func BuildMailer(ctx context.Context) mail.Sender {
	if g.Cfg().MustGet(ctx, "blog.mailer.mode").String() == "smtp" {
		s := func(k string) string { return g.Cfg().MustGet(ctx, "blog.mailer.smtp."+k).String() }
		return mail.NewSMTP(s("host"), s("port"), s("username"), s("password"), s("from"), s("fromName"))
	}
	return mail.NewDev()
}

// LoadSpamPolicy builds the comment anti-spam policy from blog.commentGuard.*
// (code-level defaults; no runtime admin). Defaults are lenient enough not to
// trip legitimate commenters: 3 links forces moderation, 5 comments/60s per IP.
func LoadSpamPolicy(ctx context.Context) catalog.SpamPolicy {
	cfg := g.Cfg()
	return catalog.SpamPolicy{
		Blacklist:         cfg.MustGet(ctx, "blog.commentGuard.blacklist").Strings(),
		MaxLinks:          cfg.MustGet(ctx, "blog.commentGuard.maxLinks", 3).Int(),
		RatePerWindow:     cfg.MustGet(ctx, "blog.commentGuard.ratePerWindow", 5).Int(),
		RateWindowSeconds: cfg.MustGet(ctx, "blog.commentGuard.rateWindowSeconds", 60).Int(),
	}
}

// JWKS is the IdP key/issuer config for the Foundation auth verifier.
type JWKS struct {
	URL               string
	Issuer            string
	Audience          string
	AllowLoopbackHTTP bool
}

func LoadJWKS(ctx context.Context) JWKS {
	return JWKS{
		URL:               g.Cfg().MustGet(ctx, "blog.jwks.url").String(),
		Issuer:            g.Cfg().MustGet(ctx, "blog.jwks.issuer").String(),
		Audience:          g.Cfg().MustGet(ctx, "blog.jwks.audience").String(),
		AllowLoopbackHTTP: g.Cfg().MustGet(ctx, "blog.jwks.allowLoopbackHttp", false).Bool(),
	}
}
