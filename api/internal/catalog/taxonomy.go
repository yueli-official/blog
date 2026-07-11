package catalog

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/model"
)

// CreateTaxonomy upserts a term (name→slug) and its taxonomy (category/tag). An
// explicit slug overrides the name-derived one (still normalized via slugify).
func (s *Service) CreateTaxonomy(ctx context.Context, name, kind, slug, parentID, description string) (*model.Taxonomy, error) {
	source := slug
	if source == "" {
		source = name
	}
	slug = slugify(source)
	if slug == "" {
		return nil, blogerr.InvalidInput("name produces an empty slug")
	}
	termID, err := s.dao.UpsertTerm(ctx, name, slug)
	if err != nil {
		return nil, err
	}
	id, err := s.dao.UpsertTaxonomy(ctx, termID, kind, description, parentID)
	if err != nil {
		return nil, err
	}
	return s.dao.GetTaxonomy(ctx, id)
}

// ListTaxonomies returns taxonomies (optionally filtered by kind), each carrying
// its published-post count (for tag-cloud sizing + governance display).
func (s *Service) ListTaxonomies(ctx context.Context, kind string) ([]*model.Taxonomy, error) {
	items, err := s.dao.ListTaxonomies(ctx, kind)
	if err != nil {
		return nil, err
	}
	counts, err := s.dao.TaxonomyPostCounts(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range items {
		t.PostCount = counts[t.ID]
	}
	return items, nil
}

// GetPostTaxonomies returns a post's assigned taxonomies (detail page tags +
// editor selection state).
func (s *Service) GetPostTaxonomies(ctx context.Context, postID string) ([]*model.Taxonomy, error) {
	return s.dao.GetPostTaxonomies(ctx, postID)
}

// UpdateTaxonomy renames / re-slugs / re-describes / re-parents a taxonomy (admin
// governance). Nil pointers are left unchanged; a non-nil empty parentId clears
// the parent. Self-parenting is rejected (a deeper cycle is admin-only, low risk).
func (s *Service) UpdateTaxonomy(ctx context.Context, id string, name, slug, description, parentID *string) (*model.Taxonomy, error) {
	cur, err := s.dao.GetTaxonomy(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, blogerr.NotFound(id)
	}
	termFields := g.Map{}
	if name != nil && *name != "" {
		termFields["name"] = *name
	}
	if slug != nil {
		sl := slugify(*slug)
		if sl == "" {
			return nil, blogerr.InvalidInput("slug produces an empty value")
		}
		termFields["slug"] = sl
	}
	taxFields := g.Map{}
	if description != nil {
		taxFields["description"] = *description
	}
	if parentID != nil {
		switch *parentID {
		case id:
			return nil, blogerr.InvalidInput("a taxonomy cannot be its own parent")
		case "":
			taxFields["parent_id"] = nil
		default:
			taxFields["parent_id"] = *parentID
		}
	}
	if len(termFields) == 0 && len(taxFields) == 0 {
		return cur, nil
	}
	if err := s.dao.UpdateTaxonomy(ctx, id, cur.TermID, termFields, taxFields); err != nil {
		return nil, err
	}
	return s.dao.GetTaxonomy(ctx, id)
}

// DeleteTaxonomy removes a taxonomy; refused while it still has child taxonomies
// (reparent or remove them first) to avoid orphaning a subtree.
func (s *Service) DeleteTaxonomy(ctx context.Context, id string) error {
	n, err := s.dao.TaxonomyChildCount(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return blogerr.InvalidState("taxonomy has child taxonomies; reparent or remove them first")
	}
	return s.dao.DeleteTaxonomy(ctx, id)
}

// MergeTaxonomy folds source into target: posts move to target, source's children
// re-parent to target, then source is deleted. Both must exist and differ.
func (s *Service) MergeTaxonomy(ctx context.Context, sourceID, targetID string) error {
	if sourceID == targetID {
		return blogerr.InvalidInput("cannot merge a taxonomy into itself")
	}
	n, err := s.dao.CountTaxonomiesByIDs(ctx, []string{sourceID, targetID})
	if err != nil {
		return err
	}
	if n != 2 {
		return blogerr.NotFound(sourceID)
	}
	return s.dao.MergeTaxonomy(ctx, sourceID, targetID)
}

// AssignTaxonomies replaces a post's taxonomy assignments (author-gated). Unknown
// taxonomy ids are rejected up front (otherwise the FK violation surfaces as 500).
func (s *Service) AssignTaxonomies(ctx context.Context, author, postID string, taxIDs []string) error {
	p, err := s.dao.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if p == nil || p.AuthorID != author {
		return blogerr.NotFound(postID)
	}
	if len(taxIDs) > 0 {
		n, err := s.dao.CountTaxonomiesByIDs(ctx, taxIDs)
		if err != nil {
			return err
		}
		if n != len(taxIDs) {
			return blogerr.InvalidInput("one or more taxonomy ids do not exist")
		}
	}
	return s.dao.SetPostTaxonomies(ctx, postID, taxIDs)
}

// Related returns up to `limit` published posts sharing taxonomies with the
// post identified by slug (most overlap first), for the "you might also like"
// section. The slug must resolve to a post visible to the viewer.
func (s *Service) Related(ctx context.Context, viewer, slug string, limit int) ([]*model.Post, error) {
	p, err := s.GetBySlug(ctx, viewer, slug)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 12 {
		limit = 6
	}
	return s.dao.RelatedPosts(ctx, p.ID, limit)
}

// PostIDsByTaxonomy resolves a taxonomy slug to the matching post ids (archive
// filter). Returns a non-nil (possibly empty) slice.
func (s *Service) PostIDsByTaxonomy(ctx context.Context, slug string) ([]string, error) {
	ids, err := s.dao.PostIDsByTaxonomySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}
