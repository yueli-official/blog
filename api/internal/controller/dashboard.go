package controller

import (
	"context"
	"time"

	"github.com/yueli-official/foundation/go/traffic"

	v1 "github.com/yueli-official/blog/api/api/v1"
	"github.com/yueli-official/blog/api/internal/appconfig"
	"github.com/yueli-official/blog/api/internal/blogerr"
	"github.com/yueli-official/blog/api/internal/catalog"
)

type Dashboard struct{ svc *catalog.Service }

func NewDashboard(svc *catalog.Service) *Dashboard { return &Dashboard{svc: svc} }

func (c *Dashboard) Overview(ctx context.Context, req *v1.DashboardOverviewReq) (*v1.DashboardOverviewRes, error) {
	if _, err := authorizationService(ctx).ManagePostOwner(ctx); err != nil {
		return nil, mapAuthorizationError(err)
	}
	days := req.Days
	if days == 0 {
		days = 14
	}
	if days != 7 && days != 14 && days != 30 {
		return nil, blogerr.InvalidInput("dashboard_range_invalid")
	}
	location, err := time.LoadLocation(appconfig.TrafficTimeZone(ctx))
	if err != nil {
		return nil, err
	}
	now := time.Now().In(location)
	toTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location).AddDate(0, 0, 1)
	fromTime := toTime.AddDate(0, 0, -days)
	previousFrom := fromTime.AddDate(0, 0, -days)
	current := traffic.DateRange{
		From: traffic.MustParseDay(fromTime.Format(time.DateOnly)),
		To:   traffic.MustParseDay(toTime.Format(time.DateOnly)),
	}
	previous := traffic.DateRange{
		From: traffic.MustParseDay(previousFrom.Format(time.DateOnly)),
		To:   current.From,
	}
	overview, err := c.svc.DashboardTraffic(ctx, catalog.DashboardWindow{
		Current: current, Previous: previous,
		CurrentFrom: fromTime, To: toTime,
	})
	if err != nil {
		return nil, err
	}
	series := make([]*v1.DashboardTrafficPointView, 0, len(overview.Series))
	for _, point := range overview.Series {
		series = append(series, &v1.DashboardTrafficPointView{
			Day: point.Day.String(), Views: point.Totals.Views,
			UniqueVisitorDays: point.Totals.UniqueVisitorDays,
		})
	}
	topPosts := make([]*v1.DashboardTopPostView, 0, len(overview.Top))
	for _, item := range overview.Top {
		topPosts = append(topPosts, &v1.DashboardTopPostView{
			ID: item.Post.ID, Title: item.Post.Title, Slug: item.Post.Slug,
			Views: item.Totals.Views, UniqueVisitorDays: item.Totals.UniqueVisitorDays,
		})
	}
	topSources := make([]*v1.DashboardTrafficSourceView, 0, len(overview.TopSources))
	for _, source := range overview.TopSources {
		topSources = append(topSources, &v1.DashboardTrafficSourceView{Source: source.Source, Views: source.Views})
	}
	return &v1.DashboardOverviewRes{
		Days:                      days,
		AllTimeViews:              overview.AllTime.Views,
		AllTimeUniqueVisitorDays:  overview.AllTime.UniqueVisitorDays,
		PeriodViews:               overview.Current.Views,
		PeriodUniqueVisitorDays:   overview.Current.UniqueVisitorDays,
		PreviousPeriodViews:       overview.Previous.Views,
		PreviousUniqueVisitorDays: overview.Previous.UniqueVisitorDays,
		Series:                    series,
		TopPosts:                  topPosts,
		TopSources:                topSources,
	}, nil
}
