package controller

import (
	"context"
	"strings"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/blogauthz"
	"github.com/yueli-official/blog/api/internal/blogerr"
	foundationauth "github.com/yueli-official/foundation/go/auth"
	"github.com/yueli-official/foundation/go/authorization"
)

type PersonalPermissions struct {
	site    string
	service *blogauthz.Service
}

func NewPersonalPermissions(site string, service *blogauthz.Service) *PersonalPermissions {
	return &PersonalPermissions{site: site, service: service}
}

var personalPermissions = []foundationauth.PersonalPermission{
	{Key: "media.upload", Label: "上传文章图片", Description: "上传此博客的正文图片，仍需具备本站创作权限。"},
	{Key: "blog.post.read", Label: "读取文章", Description: "读取当前账号有权管理的文章，包含草稿。"},
	{Key: "blog.post.create", Label: "创建文章", Description: "以本人身份创建草稿。"},
	{Key: "blog.post.update", Label: "编辑文章", Description: "编辑当前账号有权修改的文章，仍受作者归属限制。"},
	{Key: "blog.post.publish", Label: "发布文章", Description: "发布当前账号有权发布的文章。"},
	{Key: "blog.post.archive", Label: "下架文章", Description: "下架当前账号有权管理的文章。"},
}

func (c *PersonalPermissions) GetPersonalPermissions(ctx context.Context, req *v1.PersonalPermissionsReq) (*v1.PersonalPermissionsRes, error) {
	p, _ := foundationauth.FromContext(ctx)
	if p == nil || p.SubjectKind != foundationauth.SubjectClient || p.ClientID != "identity-svc" || !p.HasScope(foundationauth.PersonalPermissionsScope) || c.site == "" {
		return nil, blogerr.Forbidden()
	}
	if c.service == nil {
		return nil, blogerr.AuthorizationUnavailable()
	}
	userContext := foundationauth.NewContext(ctx, &foundationauth.Principal{Subject: req.UserKey, SubjectKind: foundationauth.SubjectUser})
	if err := c.service.ReconcileSubject(userContext); err != nil {
		return nil, mapAuthorizationError(err)
	}
	access, err := c.service.Runtime().EffectiveAccess(userContext, authorization.EffectiveAccessQuery{
		Subject: c.service.Subject(userContext), ScopeID: blogauthz.RootScopeID, IncludeDescendants: true,
	})
	if err != nil {
		return nil, mapAuthorizationError(err)
	}
	available := map[string]bool{}
	for _, capability := range access.Capabilities {
		available[string(capability)] = true
	}
	items := []foundationauth.PersonalPermission{}
	for _, item := range personalPermissions {
		if available[item.Key] || item.Key == "media.upload" && available[string(blogauthz.CapabilityPostCreate)] {
			items = append(items, item)
		}
	}
	return &v1.PersonalPermissionsRes{Site: c.site, UserKey: req.UserKey, Items: items}, nil
}

func (c *PersonalPermissions) AuthorizePersonalMedia(ctx context.Context, _ *v1.PersonalMediaAuthorizationReq) (*v1.PersonalMediaAuthorizationRes, error) {
	if c.service == nil || c.service.Runtime() == nil {
		return nil, blogerr.AuthorizationUnavailable()
	}
	p, _ := foundationauth.FromContext(ctx)
	if p == nil || !p.IsPersonalToken() || !p.HasScope("media.upload") || c.site == "" {
		return nil, blogerr.Forbidden()
	}
	// Evaluate current product rights without requiring the caller to also grant
	// article creation to a token used only for image uploads.
	decision, err := c.service.Runtime().Decide(ctx, authorization.DecisionRequest{
		Subject: c.service.Subject(ctx), Capability: blogauthz.CapabilityPostCreate, ScopeID: blogauthz.RootScopeID,
	})
	if err != nil {
		return nil, mapAuthorizationError(err)
	}
	if !decision.Allowed {
		return nil, blogerr.Forbidden()
	}
	scope, err := foundationauth.PersonalScope(c.site, "asset.profile.blog-post.upload")
	if err != nil {
		return nil, blogerr.Forbidden()
	}
	result := &v1.PersonalMediaAuthorizationRes{UserKey: p.Subject, Scopes: []string{scope}}
	if !p.ExpiresAt.IsZero() {
		result.ExpiresAt = p.ExpiresAt.Format(time.RFC3339)
	}
	return result, nil
}

// Unreviewed endpoints do not acquire PAT support merely by accepting a user
// principal. Each admitted handler must also check its business capabilities.
func PersonalTokenRoutes(r *ghttp.Request) {
	p, _ := foundationauth.FromContext(r.Context())
	if p != nil && p.IsPersonalToken() {
		path := r.URL.Path
		allowed := r.Method == "POST" && path == "/api/v1/posts" || r.Method == "GET" && path == "/api/v1/posts/mine"
		if r.Method == "POST" && (path == "/api/v1/images" || path == "/api/v1/images/finalize" || path == "/api/v1/personal-token/media-authorization") {
			allowed = true
		}
		if r.Method == "PATCH" && strings.HasPrefix(path, "/api/v1/posts/") {
			id := strings.TrimPrefix(path, "/api/v1/posts/")
			allowed = id != "" && !strings.Contains(id, "/")
		}
		if !allowed {
			r.SetError(blogerr.Forbidden())
			return
		}
	}
	r.Middleware.Next()
}
