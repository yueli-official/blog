package v1

import "github.com/gogf/gf/v2/frame/g"

// CoverInitReq begins a cover-image upload for a post (public delivery).
type CoverInitReq struct {
	g.Meta   `path:"/api/v1/posts/{id}/cover" method:"post" tags:"blog" summary:"Begin a cover image upload"`
	ID       string `json:"id" in:"path" v:"required"`
	Filename string `json:"filename" v:"required"`
	Mime     string `json:"mime"`
	Size     int64  `json:"size"`
}

type CoverInitRes struct {
	g.Meta        `status:"201"`
	UploadURL     string            `json:"uploadUrl"`
	UploadToken   string            `json:"uploadToken"`
	UploadHeaders map[string]string `json:"uploadHeaders,omitempty"`
}

// CoverFinalizeReq finalizes the uploaded cover and snapshots it onto the post.
type CoverFinalizeReq struct {
	g.Meta      `path:"/api/v1/posts/{id}/cover/finalize" method:"post" tags:"blog" summary:"Finalize a cover image"`
	ID          string `json:"id" in:"path" v:"required"`
	UploadToken string `json:"uploadToken" v:"required"`
}

type CoverFinalizeRes struct {
	CoverAssetID string `json:"coverAssetId"`
	CoverURL     string `json:"coverUrl"`
}
