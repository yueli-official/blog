// Package blogtraffic declares the Blog consumer's traffic vocabulary and
// migration bridge from the legacy post_stats projection.
package blogtraffic

import (
	"context"
	"fmt"
	"strings"

	"github.com/yueli-official/foundation/go/traffic"

	"platform/products/blog/api/internal/dao"
)

const (
	ResourcePost   traffic.ResourceKind = "post"
	baselineSource                      = "blog.post_stats.view_count"
)

func Definition(timeZone string) traffic.Definition {
	return traffic.Definition{
		Version:  traffic.DefinitionVersion,
		TimeZone: strings.TrimSpace(timeZone),
		ResourceKinds: []traffic.ResourceKindDefinition{
			{Key: ResourcePost},
		},
	}
}

type LegacySnapshot struct {
	InitialBaselines []traffic.BaselineImport
	Resources        []traffic.Resource
}

// SnapshotLegacy reads the old projection before the traffic instance is
// created so it can be imported atomically with instance creation.
func SnapshotLegacy(ctx context.Context, store *dao.PG) (LegacySnapshot, error) {
	rows, err := store.ListViewProjections(ctx)
	if err != nil {
		return LegacySnapshot{}, fmt.Errorf("list blog traffic projections: %w", err)
	}
	snapshot := LegacySnapshot{
		InitialBaselines: make([]traffic.BaselineImport, 0, len(rows)),
		Resources:        make([]traffic.Resource, 0, len(rows)),
	}
	for _, row := range rows {
		resource := traffic.Resource{Kind: ResourcePost, ID: row.PostID}
		snapshot.Resources = append(snapshot.Resources, resource)
		snapshot.InitialBaselines = append(snapshot.InitialBaselines, traffic.BaselineImport{
			Source: baselineSource, Resource: resource, Views: row.Views,
		})
	}
	return snapshot, nil
}

// Reconcile repairs the consumer-owned read projection from module truth. It is
// safe at startup, before the HTTP server accepts concurrent view writes.
func Reconcile(ctx context.Context, module traffic.Module, store *dao.PG, resources []traffic.Resource) error {
	totals, err := module.Totals(ctx, resources)
	if err != nil {
		return fmt.Errorf("read blog traffic totals: %w", err)
	}
	for _, item := range totals {
		if err := store.ReplaceViewProjection(ctx, item.Resource.ID, item.Totals.Views); err != nil {
			return fmt.Errorf("reconcile blog view projection for post %s: %w", item.Resource.ID, err)
		}
	}
	return nil
}
