package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	foundationauth "github.com/yueli-official/foundation/go/auth"
)

type PersonalPermissionsReq struct {
	g.Meta  `path:"/api/v1/internal/personal-token/permissions" method:"get" tags:"authorization" summary:"Read delegable permissions for an Identity-authenticated subject"`
	UserKey string `json:"userKey" v:"required|max-length:200"`
}

type PersonalPermissionsRes = foundationauth.PersonalPermissions

type PersonalMediaAuthorizationReq struct {
	g.Meta `path:"/api/v1/personal-token/media-authorization" method:"post" tags:"authorization" summary:"Authorize constrained media upload for the current personal token"`
}

type PersonalMediaAuthorizationRes struct {
	UserKey   string   `json:"userKey"`
	Scopes    []string `json:"scopes"`
	ExpiresAt string   `json:"expiresAt,omitempty"`
}
