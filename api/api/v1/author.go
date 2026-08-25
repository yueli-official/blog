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
	Handle      string       `json:"handle"`
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
	Author          *AuthorView `json:"author"`
	IsAdministrator bool        `json:"isAdministrator"`
	Capabilities    []string    `json:"capabilities"`
}
