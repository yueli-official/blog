// Package blogtraffic declares the Blog consumer's traffic vocabulary and
// maintains the catalog's read projection from Foundation Traffic truth.
package blogtraffic

import (
	"context"
	"fmt"
	"strings"

	"github.com/yueli-official/foundation/go/traffic"

	"github.com/yueli-official/blog/api/internal/dao"
)

const (
	ResourcePost traffic.ResourceKind = "post"
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

func ReconcileProjections(ctx context.Context, module traffic.Module, store *dao.PG) error {
	ids, err := store.ListPostIDs(ctx)
	if err != nil {
		return fmt.Errorf("list blog posts for traffic projection: %w", err)
	}
	if len(ids) == 0 {
		return nil
	}
	resources := make([]traffic.Resource, 0, len(ids))
	for _, id := range ids {
		resources = append(resources, traffic.Resource{Kind: ResourcePost, ID: id})
	}
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
