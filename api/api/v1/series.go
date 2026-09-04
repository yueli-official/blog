package v1

import "github.com/gogf/gf/v2/frame/g"

// SeriesView is the outward projection of a series (专题/连载).
type SeriesView struct {
	ID          string      `json:"id"`
	Slug        string      `json:"slug"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	CoverURL    string      `json:"coverUrl,omitempty"`
	AuthorID    string      `json:"authorId"`
	PostCount   int         `json:"postCount"`
	RecentPosts []*PostView `json:"recentPosts,omitempty"` // preview for the home carousel
}

// ── browse (public) ──────────────────────────────────────────────────────────

type ListSeriesReq struct {
	g.Meta `path:"/api/v1/series" method:"get" tags:"blog" summary:"List all series"`
}

type ListSeriesRes struct {
	Items []*SeriesView `json:"items"`
}

type GetSeriesReq struct {
	g.Meta `path:"/api/v1/series/{slug}" method:"get" tags:"blog" summary:"Get a series with its ordered posts"`
	Slug   string `json:"slug" in:"path" v:"required"`
}

type GetSeriesRes struct {
	Series *SeriesView `json:"series"`
	Posts  []*PostView `json:"posts"`
}

// ── manage (author JWT) ──────────────────────────────────────────────────────

type CreateSeriesReq struct {
	g.Meta      `path:"/api/v1/series" method:"post" tags:"blog" summary:"Create a series"`
	Name        string `json:"name" v:"required"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type CreateSeriesRes struct {
	g.Meta `status:"201"`
	Series *SeriesView `json:"series"`
}

type UpdateSeriesReq struct {
	g.Meta      `path:"/api/v1/series/{id}" method:"patch" tags:"blog" summary:"Update a series (owner/admin)"`
	ID          string  `json:"id" in:"path" v:"required"`
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
}

type UpdateSeriesRes struct {
	Series *SeriesView `json:"series"`
}

type DeleteSeriesReq struct {
	g.Meta `path:"/api/v1/series/{id}" method:"delete" tags:"blog" summary:"Delete a series (owner/admin)"`
	ID     string `json:"id" in:"path" v:"required"`
}

type DeleteSeriesRes struct {
	g.Meta `status:"204"`
}

// SetPostSeriesReq assigns a post to a series at a given order; an empty seriesId
// clears the assignment.
type SetPostSeriesReq struct {
	g.Meta      `path:"/api/v1/posts/{id}/series" method:"put" tags:"blog" summary:"Set a post's series + order"`
	ID          string `json:"id" in:"path" v:"required"`
	SeriesID    string `json:"seriesId"`
	SeriesOrder int    `json:"seriesOrder"`
}

type SetPostSeriesRes struct {
	g.Meta `status:"204"`
}
