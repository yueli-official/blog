package v1

import "github.com/gogf/gf/v2/frame/g"

type HomeConfigView struct {
	Eyebrow  string `json:"eyebrow"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

type GetHomeConfigReq struct {
	g.Meta `path:"/api/v1/home" method:"get" tags:"blog" summary:"Get homepage configuration"`
}
type GetHomeConfigRes struct {
	Config *HomeConfigView `json:"config"`
}

type UpdateHomeConfigReq struct {
	g.Meta   `path:"/api/v1/home" method:"patch" tags:"blog" summary:"Update homepage configuration"`
	Eyebrow  string `json:"eyebrow"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}
type UpdateHomeConfigRes struct {
	Config *HomeConfigView `json:"config"`
}
