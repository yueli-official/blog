// Package appconfig builds runtime objects from the GoFrame config
// (manifest/config/config.yaml + GF_* env overrides).
package appconfig

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"platform/gokit/mail"
	"platform/products/blog/api/internal/blogclient"
	"platform/products/blog/api/internal/catalog"
)

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
