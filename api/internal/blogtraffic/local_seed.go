package blogtraffic

import (
	"context"
	"fmt"
	"time"

	"github.com/yueli-official/foundation/go/traffic"
)

var localWeightedPostIDs = []string{
	"019c52f0-1000-7000-8000-000000000401",
	"019c52f0-1000-7000-8000-000000000401",
	"019c52f0-1000-7000-8000-000000000401",
	"019c52f0-1000-7000-8000-000000000402",
	"019c52f0-1000-7000-8000-000000000402",
	"019c52f0-1000-7000-8000-000000000403",
	"019c52f0-1000-7000-8000-000000000403",
	"019c52f0-1000-7000-8000-000000000404",
	"019c52f0-1000-7000-8000-000000000405",
	"019c52f0-1000-7000-8000-000000000406",
	"019c52f0-1000-7000-8000-000000000407",
	"019c52f0-1000-7000-8000-000000000408",
}

var localDailyViews = []int{10, 7, 12, 6, 9, 5, 13}

const localSeedBatchSize = 100

var localSeedEpoch = time.Date(2026, time.August, 14, 0, 0, 0, 0, time.UTC)

func localSeedDayOffset(day time.Time) int {
	civilDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	return int(civilDay.Sub(localSeedEpoch) / (24 * time.Hour))
}

func localSeedIndex(offset, length int) int {
	index := offset % length
	if index < 0 {
		index += length
	}
	return index
}

func SeedLocal(ctx context.Context, module traffic.Module, now time.Time) error {
	if module == nil {
		return nil
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	observations := make([]traffic.Observation, 0, 120)
	for windowOffset := range localDailyViews {
		dayStart := today.AddDate(0, 0, windowOffset-(len(localDailyViews)-1))
		stableOffset := localSeedDayOffset(dayStart)
		count := localDailyViews[localSeedIndex(stableOffset, len(localDailyViews))]
		occurredAt := dayStart
		step := time.Duration(0)
		// The first local sample window (2026-08-14 through 2026-08-20) was
		// already persisted at noon with minute spacing. Preserve that payload
		// forever; newer days use midnight so morning restarts cannot create
		// future observations and repeated prepare/up remains idempotent.
		if stableOffset >= 0 && stableOffset < len(localDailyViews) {
			occurredAt = dayStart.Add(12 * time.Hour)
			step = time.Minute
		}
		day := occurredAt.Format(time.DateOnly)
		for index := 0; index < count; index++ {
			observationTime := occurredAt.Add(time.Duration(index) * step)
			visitor, err := module.TokenizeVisitor(
				ctx, observationTime,
				[]byte(fmt.Sprintf("blog-local-visitor-%d", index%3)),
			)
			if err != nil {
				return err
			}
			observations = append(observations, traffic.Observation{
				EventID: traffic.EventID(fmt.Sprintf("blog-local-traffic:%s:%02d", day, index)),
				Resource: traffic.Resource{
					Kind: ResourcePost,
					ID: localWeightedPostIDs[localSeedIndex(
						stableOffset+index,
						len(localWeightedPostIDs),
					)],
				},
				OccurredAt: observationTime,
				Class:      traffic.VisitHuman,
				HasVisitor: true, VisitorToken: visitor,
			})
		}
	}
	for start := 0; start < len(observations); start += localSeedBatchSize {
		end := min(start+localSeedBatchSize, len(observations))
		if _, err := module.RecordBatch(ctx, observations[start:end]); err != nil {
			return err
		}
	}
	return nil
}
