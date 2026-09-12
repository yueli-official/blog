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
	{Key: "media.upload", Label: "上传文章图片", Description: "上传此博客的正文图片；上传封面还需选择编辑文章权限。"},
	{Key: "blog.post.read", Label: "读取文章", Description: "读取当前账号有权管理的文章，包含草稿。"},
	{Key: "blog.post.create", Label: "创建文章", Description: "以本人身份创建草稿。"},
	{Key: "blog.post.update", Label: "编辑文章", Description: "编辑正文、分类、标签、系列归属和封面，仍受当前账号的文章权限限制。"},
	{Key: "blog.post.publish", Label: "发布文章", Description: "发布当前账号有权发布的文章。"},
	{Key: "blog.post.archive", Label: "下架文章", Description: "下架当前账号有权管理的文章。"},
	{Key: "blog.post.delete", Label: "回收与恢复文章", Description: "将有权删除的文章移入回收站，或从回收站恢复。"},
	{Key: "blog.tag.create", Label: "创建标签", Description: "创建可供文章使用的标签。"},
	{Key: "blog.taxonomy.manage", Label: "管理分类与标签", Description: "新建分类、修改名称和层级、合并及删除分类或标签，仍需本站分类管理权限。"},
	{Key: "blog.series.create", Label: "创建系列", Description: "以本人身份创建文章系列。"},
	{Key: "blog.series.update", Label: "编辑系列", Description: "修改当前账号有权编辑的系列名称、路径和说明。"},
	{Key: "blog.series.delete", Label: "删除系列", Description: "删除当前账号有权删除的系列，文章保留并解除系列归属。"},
	{Key: "blog.post_flags.manage", Label: "设置文章推荐与置顶", Description: "设置文章推荐和置顶，仍需本站内容编排权限。"},
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
	available, err := c.availableCapabilities(userContext)
	if err != nil {
		return nil, mapAuthorizationError(err)
	}
	items := []foundationauth.PersonalPermission{}
	for _, item := range personalPermissions {
		if available[item.Key] || item.Key == "media.upload" && canUploadPersonalMedia(available) {
			items = append(items, item)
		}
	}
	return &v1.PersonalPermissionsRes{Site: c.site, UserKey: req.UserKey, Items: items}, nil
}

func (c *PersonalPermissions) availableCapabilities(ctx context.Context) (map[string]bool, error) {
	if c.service == nil || c.service.Runtime() == nil {
		return nil, blogerr.AuthorizationUnavailable()
	}
	if err := c.service.ReconcileSubject(ctx); err != nil {
		return nil, err
	}
	access, err := c.service.Runtime().EffectiveAccess(ctx, authorization.EffectiveAccessQuery{
		Subject: c.service.Subject(ctx), ScopeID: blogauthz.RootScopeID, IncludeDescendants: true,
	})
	if err != nil {
		return nil, err
	}
	available := make(map[string]bool, len(access.Capabilities))
	for _, capability := range access.Capabilities {
		available[string(capability)] = true
	}
	return available, nil
}

func canUploadPersonalMedia(available map[string]bool) bool {
	return available[string(blogauthz.CapabilityPostCreate)] || available[string(blogauthz.CapabilityPostUpdate)]
}

func (c *PersonalPermissions) AuthorizePersonalMedia(ctx context.Context, _ *v1.PersonalMediaAuthorizationReq) (*v1.PersonalMediaAuthorizationRes, error) {
	if c.service == nil || c.service.Runtime() == nil {
		return nil, blogerr.AuthorizationUnavailable()
	}
	p, _ := foundationauth.FromContext(ctx)
	if p == nil || !p.IsPersonalToken() || !p.HasScope("media.upload") || c.site == "" {
		return nil, blogerr.Forbidden()
	}
	// Media-only tokens may upload body images without being able to create
	// articles. Cover uploads also require an explicitly delegated update scope.
	available, err := c.availableCapabilities(ctx)
	if err != nil {
		return nil, mapAuthorizationError(err)
	}
	if !canUploadPersonalMedia(available) {
		return nil, blogerr.Forbidden()
	}
	scope, err := foundationauth.PersonalScope(c.site, "asset.profile.blog-post.upload")
	if err != nil {
		return nil, blogerr.Forbidden()
	}
	result := &v1.PersonalMediaAuthorizationRes{UserKey: p.Subject, Scopes: []string{scope}}
	if p.HasScope(string(blogauthz.CapabilityPostUpdate)) && available[string(blogauthz.CapabilityPostUpdate)] {
		coverScope, err := foundationauth.PersonalScope(c.site, "asset.profile.blog-cover.upload")
		if err != nil {
			return nil, blogerr.Forbidden()
		}
		result.Scopes = append(result.Scopes, coverScope)
	}
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
		if !personalTokenRouteAllowed(r.Method, r.URL.Path) {
			r.SetError(blogerr.Forbidden())
			return
		}
	}
	r.Middleware.Next()
}

func personalTokenRouteAllowed(method, path string) bool {
	if method == "GET" {
		return path == "/api/v1/posts/mine"
	}
	if method == "POST" {
		switch path {
		case "/api/v1/posts", "/api/v1/images", "/api/v1/images/finalize",
			"/api/v1/personal-token/media-authorization", "/api/v1/taxonomies", "/api/v1/series":
			return true
		}
	}
	parts := strings.Split(path, "/")
	if len(parts) < 5 || parts[0] != "" || parts[1] != "api" || parts[2] != "v1" || parts[4] == "" {
		return false
	}
	switch parts[3] {
	case "posts":
		if len(parts) == 5 {
			return method == "PATCH" || method == "DELETE"
		}
		if len(parts) == 6 {
			switch parts[5] {
			case "taxonomies", "series", "flags":
				return method == "PUT"
			case "cover", "restore":
				return method == "POST"
			}
		}
		return len(parts) == 7 && parts[5] == "cover" && parts[6] == "finalize" && method == "POST"
	case "taxonomies":
		return len(parts) == 5 && (method == "PATCH" || method == "DELETE") ||
			len(parts) == 6 && parts[5] == "merge" && method == "POST"
	case "series":
		return len(parts) == 5 && (method == "PATCH" || method == "DELETE")
	}
	return false
}
