package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/yueli-official/foundation/go/traffic"

	"github.com/yueli-official/blog/api/internal/blogtraffic"
	"github.com/yueli-official/blog/api/internal/model"
)

type DashboardTopPost struct {
	Post   *model.Post
	Totals traffic.Totals
}

type DashboardTraffic struct {
	AllTime    traffic.Totals
	Current    traffic.Totals
	Previous   traffic.Totals
	Series     []traffic.SeriesPoint
	Top        []DashboardTopPost
	TopSources []DashboardTrafficSource
}

type DashboardWindow struct {
	Current     traffic.DateRange
	Previous    traffic.DateRange
	CurrentFrom time.Time
	To          time.Time
}

type DashboardTrafficSource struct {
	Source string
	Views  int64
}

func (s *Service) DashboardTraffic(
	ctx context.Context,
	window DashboardWindow,
) (DashboardTraffic, error) {
	if s.traffic == nil {
		return DashboardTraffic{}, errors.New("blog traffic module is not configured")
	}
	allTime, err := s.traffic.Summary(ctx, traffic.SummaryQuery{Scope: traffic.InstanceScope()})
	if err != nil {
		return DashboardTraffic{}, err
	}
	currentSummary, err := s.traffic.Summary(ctx, traffic.SummaryQuery{
		Scope: traffic.InstanceScope(), Range: &window.Current,
	})
	if err != nil {
		return DashboardTraffic{}, err
	}
	previousSummary, err := s.traffic.Summary(ctx, traffic.SummaryQuery{
		Scope: traffic.InstanceScope(), Range: &window.Previous,
	})
	if err != nil {
		return DashboardTraffic{}, err
	}
	series, err := s.traffic.Series(ctx, traffic.SeriesQuery{
		Scope: traffic.InstanceScope(), Range: window.Current,
	})
	if err != nil {
		return DashboardTraffic{}, err
	}
	topEntries, err := s.traffic.Top(ctx, traffic.TopQuery{
		ResourceKind: blogtraffic.ResourcePost,
		Range:        &window.Current,
		Metric:       traffic.RankViews,
		Limit:        5,
	})
	if err != nil {
		return DashboardTraffic{}, err
	}
	ids := make([]string, 0, len(topEntries))
	for _, entry := range topEntries {
		ids = append(ids, entry.Resource.ID)
	}
	posts, err := s.dao.ListPublishedByIDs(ctx, ids)
	if err != nil {
		return DashboardTraffic{}, err
	}
	byID := make(map[string]*model.Post, len(posts))
	for _, post := range posts {
		byID[post.ID] = post
	}
	top := make([]DashboardTopPost, 0, len(topEntries))
	for _, entry := range topEntries {
		if post := byID[entry.Resource.ID]; post != nil {
			top = append(top, DashboardTopPost{Post: post, Totals: entry.Totals})
		}
	}
	sourceRows, err := s.dao.DashboardTrafficSources(
		ctx,
		window.CurrentFrom.Format(time.DateOnly),
		window.To.Format(time.DateOnly),
		5,
	)
	if err != nil {
		return DashboardTraffic{}, err
	}
	topSources := make([]DashboardTrafficSource, 0, len(sourceRows))
	for _, source := range sourceRows {
		topSources = append(topSources, DashboardTrafficSource{Source: source.Source, Views: source.Views})
	}
	return DashboardTraffic{
		AllTime: allTime.Totals, Current: currentSummary.Totals,
		Previous: previousSummary.Totals, Series: series, Top: top,
		TopSources: topSources,
	}, nil
}
