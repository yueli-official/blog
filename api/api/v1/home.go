package v1

import "github.com/gogf/gf/v2/frame/g"

type FriendLinkView struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type ContactLinkView struct {
	Value string `json:"value"`
	URL   string `json:"url"`
}

type HomeConfigView struct {
	Eyebrow           string            `json:"eyebrow"`
	Title             string            `json:"title"`
	Subtitle          string            `json:"subtitle"`
	SiteTitle         string            `json:"siteTitle"`
	SiteDescription   string            `json:"siteDescription"`
	SupportEmail      string            `json:"supportEmail"`
	FooterTagline     string            `json:"footerTagline"`
	FooterCopyright   string            `json:"footerCopyright"`
	FriendLinks       []FriendLinkView  `json:"friendLinks"`
	ContactLinks      []ContactLinkView `json:"contactLinks"`
	CoverAspectWidth  int               `json:"coverAspectWidth"`
	CoverAspectHeight int               `json:"coverAspectHeight"`
}

type GetHomeConfigReq struct {
	g.Meta `path:"/api/v1/home" method:"get" tags:"blog" summary:"Get homepage configuration"`
}
type GetHomeConfigRes struct {
	Config *HomeConfigView `json:"config"`
}

type UpdateHomeConfigReq struct {
	g.Meta            `path:"/api/v1/home" method:"patch" tags:"blog" summary:"Update homepage configuration"`
	Eyebrow           string             `json:"eyebrow"`
	Title             string             `json:"title"`
	Subtitle          string             `json:"subtitle"`
	SiteTitle         string             `json:"siteTitle"`
	SiteDescription   string             `json:"siteDescription"`
	SupportEmail      string             `json:"supportEmail"`
	FooterTagline     string             `json:"footerTagline"`
	FooterCopyright   string             `json:"footerCopyright"`
	FriendLinks       *[]FriendLinkView  `json:"friendLinks"`
	ContactLinks      *[]ContactLinkView `json:"contactLinks"`
	CoverAspectWidth  int                `json:"coverAspectWidth"`
	CoverAspectHeight int                `json:"coverAspectHeight"`
}
type UpdateHomeConfigRes struct {
	Config *HomeConfigView `json:"config"`
}
