package v1

import "github.com/gogf/gf/v2/frame/g"

// ── view counter (public) ────────────────────────────────────────────────────

type IncrViewReq struct {
	g.Meta `path:"/api/v1/posts/{slug}/view" method:"post" tags:"blog" summary:"Increment a post's view count"`
	Slug   string `json:"slug" in:"path" v:"required"`
}

type IncrViewRes struct {
	Ok bool `json:"ok"`
}

// ── revisions (author JWT) ───────────────────────────────────────────────────

type RevisionView struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	RevNote   string `json:"revNote"`
	CreatedAt string `json:"createdAt"`
}

type ListRevisionsReq struct {
	g.Meta `path:"/api/v1/posts/{id}/revisions" method:"get" tags:"blog" summary:"List a post's revisions"`
	ID     string `json:"id" in:"path" v:"required"`
}

type ListRevisionsRes struct {
	Items []*RevisionView `json:"items"`
}

type RestoreRevisionReq struct {
	g.Meta `path:"/api/v1/posts/{id}/revisions/{revId}/restore" method:"post" tags:"blog" summary:"Restore a post to a revision"`
	ID     string `json:"id" in:"path" v:"required"`
	RevID  string `json:"revId" in:"path" v:"required"`
}

type RestoreRevisionRes struct {
	Post *PostView `json:"post"`
}
