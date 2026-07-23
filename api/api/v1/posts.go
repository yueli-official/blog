// Package v1 holds the blog-site typed-handler request/response contracts
// (g.Meta drives GoFrame's auto OpenAPI).
package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/yueli-official/foundation/go/discovery"
)

// PostView is the outward projection of a post.
type PostView struct {
	ID            string          `json:"id"`
	AuthorID      string          `json:"authorId"`
	Title         string          `json:"title"`
	Slug          string          `json:"slug"`
	Content       string          `json:"content"`
	Excerpt       string          `json:"excerpt"`
	CoverAssetID  string          `json:"coverAssetId,omitempty"`
	CoverURL      string          `json:"coverUrl,omitempty"`
	Status        string          `json:"status"`
	CommentStatus int             `json:"commentStatus"`
	ViewCount     int64           `json:"viewCount"`
	Pinned        bool            `json:"pinned"`
	Featured      bool            `json:"featured"`
	SeriesID      string          `json:"seriesId,omitempty"`
	SeriesOrder   int             `json:"seriesOrder,omitempty"`
	PublishedAt   string          `json:"publishedAt,omitempty"`
	CreatedAt     string          `json:"createdAt"`
	UpdatedAt     string          `json:"updatedAt"`
	Taxonomies    []*TaxonomyView `json:"taxonomies,omitempty"`
}

// ── browse (public) ──────────────────────────────────────────────────────────

type ListPostsReq struct {
	g.Meta   `path:"/api/v1/posts" method:"get" tags:"blog" summary:"Browse / search published posts"`
	Taxonomy string `json:"taxonomy"` // optional category/tag slug filter
	Q        string `json:"q"`        // optional full-text search query (site search)
	Featured bool   `json:"featured"` // only featured posts (home carousel)
	Pinned   bool   `json:"pinned"`   // only pinned posts
	Sort     string `json:"sort"`     // "" (newest) | "popular" | "random" (discovery widgets)
	Page     int    `json:"page"`
	Size     int    `json:"size"`
}

type ListPostsRes struct {
	Items []*PostView `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

type GetPostReq struct {
	g.Meta `path:"/api/v1/posts/{slug}" method:"get" tags:"blog" summary:"Get a published post by slug"`
	Slug   string `json:"slug" in:"path" v:"required"`
}

type GetPostRes struct {
	Post       *PostView                 `json:"post"`
	SEO        *SEOView                  `json:"seo,omitempty"`
	Discovery  *discovery.PageProjection `json:"discovery,omitempty"`
	Taxonomies []*TaxonomyView           `json:"taxonomies"`       // the post's categories + tags
	Series     *SeriesView               `json:"series,omitempty"` // the series this post belongs to (if any)
	Author     *AuthorView               `json:"author,omitempty"` // the post author's profile (byline)
	Liked      bool                      `json:"liked"`
	Bookmarked bool                      `json:"bookmarked"`
}

// SiblingsReq fetches the published posts adjacent to one by publish time
// (article prev/next nav, M6).
type SiblingsReq struct {
	g.Meta `path:"/api/v1/posts/{slug}/siblings" method:"get" tags:"blog" summary:"Prev/next published posts"`
	Slug   string `json:"slug" in:"path" v:"required"`
}

type SiblingsRes struct {
	Prev *PostView `json:"prev,omitempty"`
	Next *PostView `json:"next,omitempty"`
}

// ArchiveReq returns published posts (lightweight), newest first, paginated for
// the date archive page (M6). The page loads more on demand rather than fetching
// the whole history at once.
type ArchiveReq struct {
	g.Meta `path:"/api/v1/archive" method:"get" tags:"blog" summary:"Published posts (date archive, paginated)"`
	Page   int `json:"page"`
	Size   int `json:"size"`
}

type ArchiveRes struct {
	Items []*PostView `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// RelatedPostsReq fetches posts sharing taxonomies with the given post (the
// detail page's "related reading" rail). Public, optional login.
type RelatedPostsReq struct {
	g.Meta `path:"/api/v1/posts/{slug}/related" method:"get" tags:"blog" summary:"Related posts sharing taxonomies"`
	Slug   string `json:"slug" in:"path" v:"required"`
	Limit  int    `json:"limit"`
}

type RelatedPostsRes struct {
	Items []*PostView `json:"items"`
}

// ── manage (author JWT) ──────────────────────────────────────────────────────

// ListMineReq lists the caller's own posts of any status (draft/published/...)
// for the manage console — distinct from the public browse list.
type ListMineReq struct {
	g.Meta      `path:"/api/v1/posts/mine" method:"get" tags:"blog" summary:"List manage-console posts (status/search/taxonomy/author filtered)"`
	Status      string   `json:"status"`      // draft|published|private|archived|issues (computed); "" = all
	Q           string   `json:"q"`           // title/slug search
	TaxonomyIds []string `json:"taxonomyIds"` // AND filter by category/tag ids
	Pinned      bool     `json:"pinned"`      // only pinned posts
	Featured    bool     `json:"featured"`    // only featured posts
	AuthorID    string   `json:"authorId"`    // admin only: scope to one author
	All         bool     `json:"all"`         // admin only: all authors' posts
	Sort        string   `json:"sort"`        // updated|title|published
	Direction   string   `json:"direction"`   // asc|desc
	Page        int      `json:"page"`
	Size        int      `json:"size"`
}

type ListMineRes struct {
	Items      []*PostView    `json:"items"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	Size       int            `json:"size"`
	Counts     map[string]int `json:"counts"`     // per-status counts + "all" (filter tabs)
	TotalViews int64          `json:"totalViews"` // sum of view_count across all my posts (dashboard)
}

type CreatePostReq struct {
	g.Meta  `path:"/api/v1/posts" method:"post" tags:"blog" summary:"Create a draft post"`
	Title   string `json:"title" v:"required"`
	Content string `json:"content"`
	Excerpt string `json:"excerpt"`
}

type CreatePostRes struct {
	Post *PostView `json:"post"`
}

// PatchPostReq updates mutable fields; setting status to "published" promotes
// the draft (publish constraints enforced server-side). Pointer fields
// distinguish "omitted" from "set to empty".
type PatchPostReq struct {
	g.Meta  `path:"/api/v1/posts/{id}" method:"patch" tags:"blog" summary:"Update a post (status drives publish)"`
	ID      string  `json:"id" in:"path" v:"required"`
	Title   *string `json:"title"`
	Slug    *string `json:"slug"` // custom URL slug (slugified server-side; must stay unique)
	Content *string `json:"content"`
	Excerpt *string `json:"excerpt"`
	Status  *string `json:"status"`
}

type PatchPostRes struct {
	Post *PostView `json:"post"`
}

type DeletePostReq struct {
	g.Meta `path:"/api/v1/posts/{id}" method:"delete" tags:"blog" summary:"Delete a post"`
	ID     string `json:"id" in:"path" v:"required"`
}

type DeletePostRes struct {
	Deleted bool `json:"deleted"`
}

// SetFlagsReq sets editorial flags (superadmin). Pointer fields distinguish
// "omitted" from "set".
type SetFlagsReq struct {
	g.Meta   `path:"/api/v1/posts/{id}/flags" method:"put" tags:"blog" summary:"Set pinned/featured (admin)"`
	ID       string `json:"id" in:"path" v:"required"`
	Pinned   *bool  `json:"pinned"`
	Featured *bool  `json:"featured"`
}

type SetFlagsRes struct {
	Post *PostView `json:"post"`
}

// BatchReq applies a lifecycle action to many of the caller's posts at once.
type BatchReq struct {
	g.Meta `path:"/api/v1/posts/batch" method:"post" tags:"blog" summary:"Batch publish/draft/archive/delete my posts"`
	IDs    []string `json:"ids" v:"required"`
	Action string   `json:"action" v:"required|in:publish,draft,archive,delete"`
}

type BatchRes struct {
	Changed  int             `json:"changed"`
	Failures []*BatchFailure `json:"failures"`
}

type BatchFailure struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
