package v1

import "github.com/gogf/gf/v2/frame/g"

type DashboardOverviewReq struct {
	g.Meta `path:"/api/v1/dashboard/overview" method:"get" tags:"blog" summary:"Get management dashboard analytics"`
	Days   int `json:"days"`
}

type DashboardTrafficPointView struct {
	Day               string `json:"day"`
	Views             int64  `json:"views"`
	UniqueVisitorDays int64  `json:"uniqueVisitorDays"`
}

type DashboardTopPostView struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	Slug              string `json:"slug"`
	Views             int64  `json:"views"`
	UniqueVisitorDays int64  `json:"uniqueVisitorDays"`
}

type DashboardTrafficSourceView struct {
	Source string `json:"source"`
	Views  int64  `json:"views"`
}

type DashboardOverviewRes struct {
	Days                      int                           `json:"days"`
	AllTimeViews              int64                         `json:"allTimeViews"`
	AllTimeUniqueVisitorDays  int64                         `json:"allTimeUniqueVisitorDays"`
	PeriodViews               int64                         `json:"periodViews"`
	PeriodUniqueVisitorDays   int64                         `json:"periodUniqueVisitorDays"`
	PreviousPeriodViews       int64                         `json:"previousPeriodViews"`
	PreviousUniqueVisitorDays int64                         `json:"previousUniqueVisitorDays"`
	Series                    []*DashboardTrafficPointView  `json:"series"`
	TopPosts                  []*DashboardTopPostView       `json:"topPosts"`
	TopSources                []*DashboardTrafficSourceView `json:"topSources"`
}
