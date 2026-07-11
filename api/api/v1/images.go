package v1

import "github.com/gogf/gf/v2/frame/g"

// ImageInitReq begins an inline content-image upload (editor E2). Unlike a cover,
// the image is standalone (not tied to a post) — it's embedded into markdown by
// its returned public URL.
type ImageInitReq struct {
	g.Meta   `path:"/api/v1/images" method:"post" tags:"blog" summary:"Begin an inline image upload"`
	Filename string `json:"filename" v:"required"`
	Mime     string `json:"mime"`
	Size     int64  `json:"size"`
}

type ImageInitRes struct {
	UploadURL     string            `json:"uploadUrl"`
	UploadToken   string            `json:"uploadToken"`
	UploadHeaders map[string]string `json:"uploadHeaders,omitempty"`
}

// ImageFinalizeReq finalizes the uploaded image and returns its public URL.
type ImageFinalizeReq struct {
	g.Meta      `path:"/api/v1/images/finalize" method:"post" tags:"blog" summary:"Finalize an inline image"`
	UploadToken string `json:"uploadToken" v:"required"`
}

type ImageFinalizeRes struct {
	URL string `json:"url"`
}
