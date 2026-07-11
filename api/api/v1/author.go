package v1

import "github.com/gogf/gf/v2/frame/g"

// SocialLink is one labelled external link on an author profile.
type SocialLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// AuthorView is the outward projection of an author profile. DisplayName/AvatarURL
// may be empty (the UI falls back to the identity). PostCount is set on the public
// author page, omitted elsewhere.
type AuthorView struct {
	ID          string       `json:"id"`
	DisplayName string       `json:"displayName"`
	Bio         string       `json:"bio"`
	AvatarURL   string       `json:"avatarUrl"`
	BannerURL   string       `json:"bannerUrl"`
	Role        string       `json:"role"`
	Status      string       `json:"status,omitempty"` // pending | active | "" (no profile) — for /me/profile
	SocialLinks []SocialLink `json:"socialLinks"`
	PostCount   int          `json:"postCount,omitempty"`
	CreatedAt   string       `json:"createdAt,omitempty"` // profile creation = "joined" date (author page)
}

// ── public author page ───────────────────────────────────────────────────────

type GetAuthorReq struct {
	g.Meta `path:"/api/v1/authors/{id}" method:"get" tags:"blog" summary:"Public author profile + their posts"`
	ID     string `json:"id" in:"path" v:"required"`
	Page   int    `json:"page"`
	Size   int    `json:"size"`
}

type GetAuthorRes struct {
	Author     *AuthorView `json:"author"`
	Posts      []*PostView `json:"posts"`
	Total      int         `json:"total"`
	TotalViews int64       `json:"totalViews"` // sum of view_count across the author's published posts
	Page       int         `json:"page"`
	Size       int         `json:"size"`
}

// ── my profile (author JWT) ──────────────────────────────────────────────────

type GetMyProfileReq struct {
	g.Meta `path:"/api/v1/me/profile" method:"get" tags:"blog" summary:"Get my author profile"`
}

type GetMyProfileRes struct {
	Author  *AuthorView `json:"author"`
	IsOwner bool        `json:"isOwner"` // blog owner (operator) — the blog's own top role, not an IdP claim
}

// ── admin author management ──────────────────────────────────────────────────

// AdminAuthorView is one roster row in the admin author-management page.
type AdminAuthorView struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`   // author | contributor
	Status      string `json:"status"` // pending | active
	PostCount   int    `json:"postCount"`
	Owner       bool   `json:"owner"` // site operator (blog.operatorSubs) — role is config, not editable here
}

type AdminListAuthorsReq struct {
	g.Meta `path:"/api/v1/admin/authors" method:"get" tags:"blog" summary:"List authors + roles (admin)"`
}

type AdminListAuthorsRes struct {
	Authors []*AdminAuthorView `json:"authors"`
}

// AdminSetAuthorRoleReq sets an author's role 主笔(author)/客座(contributor). Owner only.
type AdminSetAuthorRoleReq struct {
	g.Meta `path:"/api/v1/admin/authors/{id}/role" method:"put" tags:"blog" summary:"Set an author's role (owner)"`
	ID     string `json:"id" in:"path" v:"required"`
	Role   string `json:"role" v:"required|in:author,contributor"`
}

type AdminSetAuthorRoleRes struct {
	Author *AuthorView `json:"author"`
}

// RequestAuthorReq records the caller's authorship request (creates a pending
// profile awaiting admin approval).
type RequestAuthorReq struct {
	g.Meta `path:"/api/v1/me/author-request" method:"post" tags:"blog" summary:"Request to become an author"`
}

type RequestAuthorRes struct {
	Author *AuthorView `json:"author"`
}

// AdminApproveAuthorReq approves a pending author request → active (admin).
type AdminApproveAuthorReq struct {
	g.Meta `path:"/api/v1/admin/authors/{id}/approve" method:"post" tags:"blog" summary:"Approve an author request (admin)"`
	ID     string `json:"id" in:"path" v:"required"`
}

type AdminApproveAuthorRes struct {
	Author *AuthorView `json:"author"`
}

// AdminRemoveAuthorReq rejects a request / revokes authorship (admin).
type AdminRemoveAuthorReq struct {
	g.Meta `path:"/api/v1/admin/authors/{id}" method:"delete" tags:"blog" summary:"Reject/revoke an author (admin)"`
	ID     string `json:"id" in:"path" v:"required"`
}

type AdminRemoveAuthorRes struct {
	Removed bool `json:"removed"`
}
