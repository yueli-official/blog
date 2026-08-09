// Package blogurls adapts the Foundation URL lifecycle module to Blog's
// public resource identities. Product rows remain authoritative; this package
// only owns public address history and resolution.
package blogurls

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/yueli-official/foundation/go/identifier"
	"github.com/yueli-official/foundation/go/urllifecycle"
)

const (
	PostKind     urllifecycle.ResourceKind = "blog.post"
	CategoryKind urllifecycle.ResourceKind = "blog.category"
	TagKind      urllifecycle.ResourceKind = "blog.tag"
)

type State struct {
	ID        string
	Kind      urllifecycle.ResourceKind
	Slug      string
	Published bool
}

type Claim struct {
	State State
}

type Lifecycle struct {
	module   urllifecycle.Module
	postgres *urllifecycle.PostgresAdapter
}

func Definition(origin string) urllifecycle.Definition {
	return urllifecycle.Definition{
		Version:       urllifecycle.DefinitionVersion,
		TrustedOrigin: strings.TrimRight(origin, "/"),
		ResourceKinds: []urllifecycle.ResourceKindDefinition{
			{Key: PostKind},
			{Key: CategoryKind},
			{Key: TagKind},
		},
		Namespaces: []urllifecycle.NamespaceDefinition{
			{Key: "blog.posts", PathPrefix: "/posts"},
			{Key: "blog.categories", PathPrefix: "/category"},
			{Key: "blog.tags", PathPrefix: "/tags"},
		},
	}
}

func NewPostgres(ctx context.Context, db *sql.DB, instanceKey, origin string) (*Lifecycle, error) {
	catalog, err := urllifecycle.Compile(Definition(origin))
	if err != nil {
		return nil, err
	}
	adapter, err := urllifecycle.NewPostgres(ctx, catalog, urllifecycle.PostgresOptions{
		DB: db, InstanceKey: instanceKey,
	})
	if err != nil {
		return nil, err
	}
	return &Lifecycle{module: adapter, postgres: adapter}, nil
}

func NewMemory(origin string) (*Lifecycle, error) {
	catalog, err := urllifecycle.Compile(Definition(origin))
	if err != nil {
		return nil, err
	}
	module, err := urllifecycle.NewMemory(catalog, urllifecycle.MemoryOptions{})
	if err != nil {
		return nil, err
	}
	return &Lifecycle{module: module}, nil
}

func (l *Lifecycle) Resolver() urllifecycle.Resolver {
	if l == nil {
		return nil
	}
	return l.module
}

func (l *Lifecycle) Reconcile(ctx context.Context, claims []Claim) error {
	for _, claim := range claims {
		if !claim.State.Published {
			continue
		}
		if err := l.ensure(ctx, l.module, claim.State, "blog startup reconciliation"); err != nil {
			return err
		}
	}
	return nil
}

func (l *Lifecycle) Change(ctx context.Context, tx *sql.Tx, before, after State) error {
	module, err := l.bound(tx)
	if err != nil {
		return err
	}
	if !before.Published && !after.Published {
		return nil
	}
	if !before.Published && after.Published {
		return l.ensure(ctx, module, after, "blog resource published")
	}
	inspection, found, err := inspect(ctx, module, route(before))
	if err != nil {
		return err
	}
	if !found {
		return l.ensure(ctx, module, after, "blog lifecycle repair")
	}
	if before.Slug == after.Slug {
		return nil
	}
	change := urllifecycle.Rename(
		meta("blog resource renamed"),
		route(after),
		inspection.Revision,
		*inspection.Active,
		ref(after),
		urllifecycle.DefaultPermanentRedirect(),
	)
	_, err = module.Apply(ctx, change, urllifecycle.ApplyOptions{})
	return err
}

func (l *Lifecycle) Delete(ctx context.Context, tx *sql.Tx, state State) error {
	if !state.Published {
		return nil
	}
	module, err := l.bound(tx)
	if err != nil {
		return err
	}
	inspection, found, err := inspect(ctx, module, route(state))
	if err != nil || !found {
		return err
	}
	change := urllifecycle.Retire(
		meta("blog resource deleted"),
		urllifecycle.RetireGone(route(state), inspection.Revision),
	)
	_, err = module.Apply(ctx, change, urllifecycle.ApplyOptions{})
	return err
}

func (l *Lifecycle) Merge(ctx context.Context, tx *sql.Tx, source, target State) error {
	module, err := l.bound(tx)
	if err != nil {
		return err
	}
	if err := l.ensure(ctx, module, target, "blog taxonomy merge target repair"); err != nil {
		return err
	}
	if err := l.ensure(ctx, module, source, "blog taxonomy merge source repair"); err != nil {
		return err
	}
	sourceInspection, _, err := inspect(ctx, module, route(source))
	if err != nil {
		return err
	}
	change := urllifecycle.Merge(
		meta("blog taxonomy merged"),
		route(source),
		sourceInspection.Revision,
		route(target),
		urllifecycle.DefaultPermanentRedirect(),
	)
	_, err = module.Apply(ctx, change, urllifecycle.ApplyOptions{})
	return err
}

func (l *Lifecycle) ensure(ctx context.Context, module urllifecycle.Module, state State, reason string) error {
	inspection, found, err := inspect(ctx, module, route(state))
	if err != nil {
		return err
	}
	if !found {
		change := urllifecycle.Claim(meta(reason), urllifecycle.ClaimSpec{
			Route: route(state), Active: urllifecycle.ActiveRoute{Canonical: ref(state)},
		})
		_, err = module.Apply(ctx, change, urllifecycle.ApplyOptions{})
		return err
	}
	if inspection.Active == nil || inspection.Active.Canonical.Path == ref(state).Path {
		return nil
	}
	change := urllifecycle.Rename(
		meta(reason),
		route(state),
		inspection.Revision,
		*inspection.Active,
		ref(state),
		urllifecycle.DefaultPermanentRedirect(),
	)
	_, err = module.Apply(ctx, change, urllifecycle.ApplyOptions{})
	return err
}

func (l *Lifecycle) bound(tx *sql.Tx) (urllifecycle.Module, error) {
	if l == nil || l.module == nil {
		return nil, fmt.Errorf("blog URL lifecycle is not configured")
	}
	if l.postgres == nil {
		return l.module, nil
	}
	if tx == nil {
		return nil, fmt.Errorf("blog URL lifecycle requires the product transaction")
	}
	return l.postgres.Bind(tx)
}

func inspect(ctx context.Context, module urllifecycle.Reader, key urllifecycle.RouteKey) (urllifecycle.Inspection, bool, error) {
	value, err := module.Inspect(ctx, urllifecycle.InspectQuery{Route: &key})
	if err == nil {
		return value, true, nil
	}
	var lifecycleErr *urllifecycle.Error
	if errors.As(err, &lifecycleErr) && lifecycleErr.Kind == urllifecycle.ErrorNotFound {
		return urllifecycle.Inspection{}, false, nil
	}
	return urllifecycle.Inspection{}, false, err
}

func meta(reason string) urllifecycle.MutationMeta {
	return urllifecycle.MutationMeta{
		CommandID: urllifecycle.CommandID(identifier.MustNew().String()),
		Actor:     urllifecycle.ActorRef{Kind: "system", ID: "blog"},
		Reason:    reason,
	}
}

func route(state State) urllifecycle.RouteKey {
	return urllifecycle.RouteKey{
		Resource: urllifecycle.ResourceKey{Kind: state.Kind, ID: state.ID},
	}
}

func ref(state State) urllifecycle.LocalRef {
	prefix := "/posts/"
	switch state.Kind {
	case CategoryKind:
		prefix = "/category/"
	case TagKind:
		prefix = "/tags/"
	}
	return urllifecycle.LocalRef{Path: prefix + state.Slug}
}
