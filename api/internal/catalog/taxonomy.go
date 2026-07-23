package catalog

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gogf/gf/v2/frame/g"

	"platform/gokit/classification"
	"platform/products/blog/api/internal/blogerr"
	"platform/products/blog/api/internal/blogurls"
	"platform/products/blog/api/internal/dao"
	"platform/products/blog/api/internal/model"
)

const blogPostPolicyKey = "blog.post.default"

func (s *Service) CreateTaxonomy(ctx context.Context, name, kind, slug, parentID, description string) (*model.Taxonomy, error) {
	if kind != "category" && kind != "tag" {
		return nil, blogerr.InvalidInput("taxonomy must be category or tag")
	}
	if kind == "tag" && strings.TrimSpace(parentID) != "" {
		return nil, blogerr.InvalidInput("tags are flat and cannot have a parent")
	}
	source := slug
	if source == "" {
		source = name
	}
	slug = slugify(source)
	if slug == "" {
		return nil, blogerr.InvalidInput("name produces an empty slug")
	}
	if existing, err := s.dao.GetTaxonomyBySlug(ctx, kind, slug); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	if kind == "category" {
		if parentID != "" {
			parent, err := s.dao.GetTaxonomy(ctx, parentID)
			if err != nil {
				return nil, err
			}
			if parent == nil || parent.Taxonomy != "category" || parent.Status != string(classification.StatusActive) {
				return nil, blogerr.InvalidInput("category parent must be an active category")
			}
		}
		id, err := s.dao.CreateCategoryWithHook(
			ctx, name, slug, parentID, description,
			s.taxonomyCreateHook(blogurls.State{Kind: blogurls.CategoryKind}, slug),
		)
		if err != nil {
			return nil, err
		}
		return s.dao.GetTaxonomy(ctx, id)
	}
	lookup, err := s.classificationTagLookup(ctx, name)
	if err != nil {
		return nil, err
	}
	matches, _, err := s.dao.ClassificationTagMatches(ctx, []classification.TagLookupRequest{lookup})
	if err != nil {
		return nil, err
	}
	if len(matches) == 1 {
		switch matches[0].Kind {
		case classification.TagMatchCanonical, classification.TagMatchAlias, classification.TagMatchReplacement:
			return s.dao.GetTaxonomy(ctx, matches[0].TagID)
		case classification.TagMatchInactive:
			return nil, blogerr.InvalidState("tag is inactive")
		}
	}
	id, err := s.dao.CreateTagWithHook(
		ctx, name, slug, description, lookup.LookupKey,
		s.taxonomyCreateHook(blogurls.State{Kind: blogurls.TagKind}, slug),
	)
	if err != nil {
		return nil, err
	}
	return s.dao.GetTaxonomy(ctx, id)
}

func (s *Service) ListTaxonomies(ctx context.Context, kind string) ([]*model.Taxonomy, error) {
	items, _, _, _, err := s.ListTaxonomiesPage(ctx, kind, "", "name", "asc", 0, 0)
	return items, err
}

func (s *Service) ListTaxonomiesPage(ctx context.Context, kind, q, sortBy, direction string, page, size int) ([]*model.Taxonomy, int, int, int, error) {
	limit, offset := 0, 0
	if size > 0 {
		page, size = norm(page, size)
		limit, offset = size, (page-1)*size
	}
	items, total, err := s.dao.ListTaxonomiesPage(ctx, dao.TaxonomyListFilter{Kind: kind, Q: q, Sort: sortBy, Direction: direction}, limit, offset)
	return items, total, page, size, err
}

func (s *Service) GetPostTaxonomies(ctx context.Context, postID string) ([]*model.Taxonomy, error) {
	return s.dao.GetPostTaxonomies(ctx, postID)
}

func (s *Service) HydratePostTaxonomies(ctx context.Context, posts []*model.Post) error {
	ids := make([]string, 0, len(posts))
	for _, post := range posts {
		ids = append(ids, post.ID)
	}
	byPost, err := s.dao.GetPostTaxonomiesByPostIDs(ctx, ids)
	if err != nil {
		return err
	}
	for _, post := range posts {
		post.Taxonomies = byPost[post.ID]
	}
	return nil
}

func (s *Service) UpdateTaxonomy(ctx context.Context, id string, name, slug, description, parentID *string) (*model.Taxonomy, error) {
	current, err := s.dao.GetTaxonomy(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil || current.Status == string(classification.StatusReplaced) {
		return nil, blogerr.NotFound(id)
	}
	fields := g.Map{}
	lookupKey := ""
	if name != nil && strings.TrimSpace(*name) != "" {
		fields["current_name"] = strings.TrimSpace(*name)
		if current.Taxonomy == "tag" {
			lookup, err := s.classificationTagLookup(ctx, *name)
			if err != nil {
				return nil, err
			}
			matches, _, err := s.dao.ClassificationTagMatches(ctx, []classification.TagLookupRequest{lookup})
			if err != nil {
				return nil, err
			}
			if len(matches) == 1 && matches[0].Kind != classification.TagMatchNotFound && matches[0].TagID != current.ID {
				return nil, blogerr.InvalidInput("tag name already resolves to another canonical tag")
			}
			lookupKey = lookup.LookupKey
		}
	}
	if slug != nil {
		normalized := slugify(*slug)
		if normalized == "" {
			return nil, blogerr.InvalidInput("slug produces an empty value")
		}
		fields["current_slug"] = normalized
	}
	if description != nil {
		fields["description"] = *description
	}
	if parentID != nil {
		if current.Taxonomy == "tag" && *parentID != "" {
			return nil, blogerr.InvalidInput("tags are flat and cannot have a parent")
		}
		switch *parentID {
		case id:
			return nil, blogerr.InvalidInput("a category cannot be its own parent")
		case "":
			fields["parent_id"] = nil
		default:
			parent, err := s.dao.GetTaxonomy(ctx, *parentID)
			if err != nil {
				return nil, err
			}
			if parent == nil || parent.Taxonomy != "category" || parent.Status != string(classification.StatusActive) {
				return nil, blogerr.InvalidInput("category parent must be an active category")
			}
			if descendant, err := s.categoryDescendsFrom(ctx, *parentID, id); err != nil {
				return nil, err
			} else if descendant {
				return nil, blogerr.InvalidInput("category parent would create a cycle")
			}
			fields["parent_id"] = *parentID
		}
	}
	if len(fields) == 0 {
		return current, nil
	}
	beforeURL := taxonomyURLState(current)
	afterURL := beforeURL
	if normalized, ok := fields["current_slug"].(string); ok {
		afterURL.Slug = normalized
	}
	if err := s.dao.UpdateTaxonomyWithHook(ctx, current, fields, lookupKey, s.urlChangeHook(beforeURL, afterURL)); err != nil {
		return nil, err
	}
	return s.dao.GetTaxonomy(ctx, id)
}

func (s *Service) DeleteTaxonomy(ctx context.Context, id string) error {
	value, err := s.dao.GetTaxonomy(ctx, id)
	if err != nil {
		return err
	}
	if value == nil {
		return blogerr.NotFound(id)
	}
	if value.Taxonomy == "category" {
		count, err := s.dao.TaxonomyChildCount(ctx, id)
		if err != nil {
			return err
		}
		if count > 0 {
			return blogerr.InvalidState("category has children; reparent or remove them first")
		}
	}
	return s.dao.DeleteTaxonomyWithHook(ctx, value, s.urlDeleteHook(taxonomyURLState(value)))
}

func (s *Service) MergeTaxonomy(ctx context.Context, sourceID, targetID string) error {
	if sourceID == targetID {
		return blogerr.InvalidInput("cannot merge a taxonomy into itself")
	}
	source, err := s.dao.GetTaxonomy(ctx, sourceID)
	if err != nil {
		return err
	}
	target, err := s.dao.GetTaxonomy(ctx, targetID)
	if err != nil {
		return err
	}
	if source == nil || target == nil {
		return blogerr.NotFound(sourceID)
	}
	if source.Status == string(classification.StatusReplaced) {
		return blogerr.InvalidState("merge source is already replaced")
	}
	if source.Taxonomy != target.Taxonomy {
		return blogerr.InvalidInput("category and tag cannot be merged across kinds")
	}
	if target.Status != string(classification.StatusActive) {
		return blogerr.InvalidState("merge target must be active")
	}
	if source.Taxonomy == "category" {
		if descendant, err := s.categoryDescendsFrom(ctx, targetID, sourceID); err != nil {
			return err
		} else if descendant {
			return blogerr.InvalidInput("category cannot merge into its own subtree")
		}
	}
	return s.dao.MergeTaxonomyWithHook(
		ctx, source, target,
		s.taxonomyMergeHook(taxonomyURLState(source), taxonomyURLState(target)),
	)
}

func (s *Service) AssignTaxonomies(ctx context.Context, author, postID string, taxonomyIDs []string) error {
	post, err := s.dao.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil || post.AuthorID != author {
		return blogerr.NotFound(postID)
	}
	ids := uniqueTaxonomyIDs(taxonomyIDs)
	values, err := s.dao.TaxonomiesByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(values) != len(ids) {
		return blogerr.InvalidInput("one or more taxonomy ids do not exist")
	}
	categoryIDs := make([]string, 0, len(values))
	tags := make([]string, 0, len(values))
	for _, value := range values {
		switch value.Taxonomy {
		case "category":
			categoryIDs = append(categoryIDs, value.ID)
		case "tag":
			tags = append(tags, value.Name)
		}
	}
	catalog, err := s.classificationCatalog(ctx)
	if err != nil {
		return err
	}
	preparation := catalog.Classify(classification.ClassifyRequest{PolicyKey: blogPostPolicyKey, CategoryIDs: categoryIDs, Tags: tags})
	factRequest := preparation.FactRequest()
	matches, freshnessToken, err := s.dao.ClassificationTagMatches(ctx, factRequest.TagLookups)
	if err != nil {
		return err
	}
	result := preparation.Complete(classification.ClassifyFacts{
		CatalogRevision: factRequest.CatalogRevision,
		RequestToken:    factRequest.RequestToken,
		FreshnessToken:  freshnessToken,
		TagMatches:      matches,
	})
	if result.Outcome != classification.OutcomeAccepted {
		return blogClassificationError(result.Diagnostics)
	}
	if len(result.TagCreations) != 0 || len(result.TagProposals) != 0 {
		return blogerr.InvalidState("selected tags must resolve before assignment")
	}
	return s.dao.SetPostClassification(ctx, postID, result.Assignments)
}

func (s *Service) Related(ctx context.Context, viewer, slug string, limit int) ([]*model.Post, error) {
	post, err := s.GetBySlug(ctx, viewer, slug)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 12 {
		limit = 6
	}
	return s.dao.RelatedPosts(ctx, post.ID, limit)
}

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

func (s *Service) classificationCatalog(ctx context.Context) (*classification.Catalog, error) {
	snapshot, err := s.dao.ClassificationSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	compiled := classification.Compile(snapshot)
	if compiled.Outcome != classification.OutcomeAccepted || compiled.Catalog == nil {
		return nil, blogClassificationError(compiled.Diagnostics)
	}
	return compiled.Catalog, nil
}

func (s *Service) classificationTagLookup(ctx context.Context, name string) (classification.TagLookupRequest, error) {
	catalog, err := s.classificationCatalog(ctx)
	if err != nil {
		return classification.TagLookupRequest{}, err
	}
	request := catalog.Classify(classification.ClassifyRequest{PolicyKey: blogPostPolicyKey, Tags: []string{name}}).FactRequest()
	if len(request.TagLookups) != 1 {
		return classification.TagLookupRequest{}, blogerr.InvalidInput("tag name is empty")
	}
	return request.TagLookups[0], nil
}

func (s *Service) categoryDescendsFrom(ctx context.Context, categoryID, ancestorID string) (bool, error) {
	seen := map[string]bool{}
	current := categoryID
	for current != "" {
		if current == ancestorID {
			return true, nil
		}
		if seen[current] {
			return true, nil
		}
		seen[current] = true
		value, err := s.dao.GetTaxonomy(ctx, current)
		if err != nil {
			return false, err
		}
		if value == nil || value.Taxonomy != "category" {
			return false, nil
		}
		current = value.ParentID
	}
	return false, nil
}

func uniqueTaxonomyIDs(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, raw := range values {
		if value := strings.TrimSpace(raw); value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func blogClassificationError(diagnostics []classification.Diagnostic) error {
	if len(diagnostics) == 0 {
		return blogerr.InvalidState("classification rejected without diagnostics")
	}
	value := diagnostics[0]
	return blogerr.InvalidInput(fmt.Sprintf("%s: %s", value.Code, value.Reference))
}
