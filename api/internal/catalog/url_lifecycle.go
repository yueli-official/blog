package catalog

import (
	"context"
	"database/sql"

	"platform/products/blog/api/internal/blogurls"
	"platform/products/blog/api/internal/dao"
	"platform/products/blog/api/internal/model"
)

// SetURLLifecycle enables atomic public-address history. It is optional only
// for isolated catalog tests; production wires it during startup.
func (s *Service) SetURLLifecycle(lifecycle *blogurls.Lifecycle) {
	s.urls = lifecycle
}

func postURLState(post *model.Post) blogurls.State {
	if post == nil {
		return blogurls.State{}
	}
	return blogurls.State{
		ID: post.ID, Kind: blogurls.PostKind, Slug: post.Slug,
		Published: post.PublishedAt != nil,
	}
}

func taxonomyURLState(value *model.Taxonomy) blogurls.State {
	if value == nil {
		return blogurls.State{}
	}
	kind := blogurls.CategoryKind
	if value.Taxonomy == "tag" {
		kind = blogurls.TagKind
	}
	return blogurls.State{ID: value.ID, Kind: kind, Slug: value.Slug, Published: true}
}

func (s *Service) urlChangeHook(before, after blogurls.State) dao.TransactionHook {
	if s.urls == nil {
		return nil
	}
	return func(ctx context.Context, tx *sql.Tx) error {
		return s.urls.Change(ctx, tx, before, after)
	}
}

func (s *Service) urlDeleteHook(state blogurls.State) dao.TransactionHook {
	if s.urls == nil {
		return nil
	}
	return func(ctx context.Context, tx *sql.Tx) error {
		return s.urls.Delete(ctx, tx, state)
	}
}

func (s *Service) taxonomyCreateHook(kind blogurls.State, slug string) dao.CreateTransactionHook {
	if s.urls == nil {
		return nil
	}
	return func(ctx context.Context, tx *sql.Tx, id string) error {
		kind.ID = id
		kind.Slug = slug
		kind.Published = true
		return s.urls.Change(ctx, tx, blogurls.State{ID: id, Kind: kind.Kind}, kind)
	}
}

func (s *Service) taxonomyMergeHook(source, target blogurls.State) dao.TransactionHook {
	if s.urls == nil {
		return nil
	}
	return func(ctx context.Context, tx *sql.Tx) error {
		return s.urls.Merge(ctx, tx, source, target)
	}
}
