// Package model holds the blog-catalog row structs. orm tags pin the column
// mapping for GoFrame gdb scan/insert.
package model

import "github.com/gogf/gf/v2/os/gtime"

// Status is the publish lifecycle of a post.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusPrivate   Status = "private"
	StatusArchived  Status = "archived"
	StatusTrash     Status = "trash"
)

// Post is one article (markdown content + metadata + a cover via public delivery).
type Post struct {
	ID            string      `json:"id" orm:"id"`
	AuthorID      string      `json:"authorId" orm:"author_id"`
	PostType      string      `json:"postType" orm:"post_type"`
	Title         string      `json:"title" orm:"title"`
	Slug          string      `json:"slug" orm:"slug"`
	Content       string      `json:"content" orm:"content"`
	Excerpt       string      `json:"excerpt" orm:"excerpt"`
	CoverAssetID  string      `json:"coverAssetId" orm:"cover_asset_id"`
	CoverURL      string      `json:"coverUrl" orm:"cover_url"`
	CommentStatus int         `json:"commentStatus" orm:"comment_status"`
	Status        Status      `json:"status" orm:"status"`
	Locale        string      `json:"locale" orm:"locale"`
	Pinned        bool        `json:"pinned" orm:"pinned"`
	Featured      bool        `json:"featured" orm:"featured"`
	SeriesID      string      `json:"seriesId" orm:"series_id"`
	SeriesOrder   int         `json:"seriesOrder" orm:"series_order"`
	PublishedAt   *gtime.Time `json:"publishedAt" orm:"published_at"`
	CreatedAt     *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt     *gtime.Time `json:"updatedAt" orm:"updated_at"`
	DeletedAt     *gtime.Time `json:"deletedAt" orm:"deleted_at"`
	ViewCount     int64       `json:"-" orm:"view_count"` // transient: joined from post_stats in List/Archive
	Taxonomies    []*Taxonomy `json:"-" orm:"-"`          // transient: batch-hydrated for management list chips
}

// Stats is a post's aggregate counters.
type Stats struct {
	PostID       string `json:"postId" orm:"post_id"`
	ViewCount    int64  `json:"viewCount" orm:"view_count"`
	LikeCount    int64  `json:"likeCount" orm:"like_count"`
	CommentCount int64  `json:"commentCount" orm:"comment_count"`
	ShareCount   int64  `json:"shareCount" orm:"share_count"`
}

// SEO is a post's SEO metadata (1:1 with the post).
type SEO struct {
	PostID       string `json:"postId" orm:"post_id"`
	MetaTitle    string `json:"metaTitle" orm:"meta_title"`
	MetaDesc     string `json:"metaDesc" orm:"meta_desc"`
	OgTitle      string `json:"ogTitle" orm:"og_title"`
	OgImage      string `json:"ogImage" orm:"og_image"`
	CanonicalURL string `json:"canonicalUrl" orm:"canonical_url"`
	Robots       string `json:"robots" orm:"robots"`
}

// Revision is a snapshot of a post's title/content taken before an edit.
type Revision struct {
	ID        string      `json:"id" orm:"id"`
	PostID    string      `json:"postId" orm:"post_id"`
	AuthorID  string      `json:"authorId" orm:"author_id"`
	Title     string      `json:"title" orm:"title"`
	Content   string      `json:"content" orm:"content"`
	RevNote   string      `json:"revNote" orm:"rev_note"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
}

// AuthorProfile is the blog-LOCAL author state, keyed by the IdP subject
// (Post.AuthorID). Only domain-specific fields (role / write-gate status) live
// here; display data (name / avatar / cover / bio / social) is owned by the
// identity service and overlaid at projection time (see identityclient), so it
// is deliberately absent.
type AuthorProfile struct {
	AuthorID  string      `json:"authorId" orm:"author_id"`
	Role      string      `json:"role" orm:"role"`
	Status    string      `json:"status" orm:"status"` // pending | active (write gate)
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at"`
}

// AuthorRoster is one row of the admin author-management roster: local author
// state plus the non-deleted post count (joined on read). Display name is
// resolved from the identity service at projection time.
type AuthorRoster struct {
	AuthorID  string      `orm:"author_id"`
	Role      string      `orm:"role"`
	Status    string      `orm:"status"`
	PostCount int         `orm:"post_count"`
	CreatedAt *gtime.Time `orm:"created_at"`
}

// Term is a reusable taxonomy label (shared across category/tag).
type Term struct {
	ID   string `json:"id" orm:"id"`
	Name string `json:"name" orm:"name"`
	Slug string `json:"slug" orm:"slug"`
}

// Series is a curated sequence of posts (专题/连载), authored by one author.
// PostCount is computed on read (published posts in the series).
type Series struct {
	ID           string      `json:"id" orm:"id"`
	Slug         string      `json:"slug" orm:"slug"`
	Name         string      `json:"name" orm:"name"`
	Description  string      `json:"description" orm:"description"`
	CoverAssetID string      `json:"coverAssetId" orm:"cover_asset_id"`
	CoverURL     string      `json:"coverUrl" orm:"cover_url"`
	AuthorID     string      `json:"authorId" orm:"author_id"`
	CreatedAt    *gtime.Time `json:"createdAt" orm:"created_at"`
	UpdatedAt    *gtime.Time `json:"updatedAt" orm:"updated_at"`
	PostCount    int         `json:"postCount"` // computed, not a column
	Recent       []*Post     `json:"-"`         // transient: recent posts (home series carousel)
}

// Taxonomy is a category or tag (a term applied under a taxonomy kind). Name/Slug
// are joined from terms for views and are not stored on this table.
type Taxonomy struct {
	ID          string `json:"id" orm:"id"`
	TermID      string `json:"termId" orm:"term_id"`
	Taxonomy    string `json:"taxonomy" orm:"taxonomy"`
	Description string `json:"description" orm:"description"`
	ParentID    string `json:"parentId" orm:"parent_id"`
	PostCount   int    `json:"postCount" orm:"post_count"`
	Name        string `json:"name"` // joined from terms (LeftJoin in ListTaxonomies)
	Slug        string `json:"slug"` // joined from terms
}
