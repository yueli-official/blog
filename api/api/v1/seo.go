package v1

import "github.com/gogf/gf/v2/frame/g"

// SEOView is the outward projection of a post's SEO metadata.
type SEOView struct {
	MetaTitle    string `json:"metaTitle"`
	MetaDesc     string `json:"metaDesc"`
	OgTitle      string `json:"ogTitle"`
	OgImage      string `json:"ogImage"`
	CanonicalURL string `json:"canonicalUrl"`
	Robots       string `json:"robots"`
}

// PutSEOReq sets a post's SEO metadata (author JWT). Pointer fields distinguish
// "omitted" from "set to empty".
type PutSEOReq struct {
	g.Meta       `path:"/api/v1/posts/{id}/seo" method:"put" tags:"blog" summary:"Set a post's SEO metadata"`
	ID           string  `json:"id" in:"path" v:"required"`
	MetaTitle    *string `json:"metaTitle"`
	MetaDesc     *string `json:"metaDesc"`
	OgTitle      *string `json:"ogTitle"`
	OgImage      *string `json:"ogImage"`
	CanonicalURL *string `json:"canonicalUrl"`
	Robots       *string `json:"robots"`
}

type PutSEORes struct {
	SEO *SEOView `json:"seo"`
}
