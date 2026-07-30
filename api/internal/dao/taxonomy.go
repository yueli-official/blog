package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/yueli-official/blog/api/internal/model"
	"github.com/yueli-official/foundation/go/classification"
)

type TaxonomyListFilter struct {
	Kind      string
	Q         string
	Sort      string
	Direction string
}

const taxonomyProjectionSQL = `
SELECT category.id::text AS id, 'category'::text AS taxonomy,
       category.current_name AS name, category.current_slug AS slug,
       category.description, COALESCE(category.parent_id::text, '') AS parent_id,
       category.status,
       (SELECT COUNT(DISTINCT assignment.post_id) FROM blog_post_category_assignments assignment
        JOIN posts post ON post.id = assignment.post_id
        WHERE assignment.category_id IN (
            WITH RECURSIVE descendants AS (
                SELECT category.id
                UNION ALL
                SELECT child.id FROM blog_categories child
                JOIN descendants parent ON child.parent_id = parent.id
            )
            SELECT id FROM descendants
        )
          AND post.status = 'published' AND post.deleted_at IS NULL)::bigint AS post_count
FROM blog_categories category
UNION ALL
SELECT tag.id::text AS id, 'tag'::text AS taxonomy,
       tag.current_name AS name, tag.current_slug AS slug,
       tag.description, ''::text AS parent_id, tag.status,
       (SELECT COUNT(*) FROM blog_post_tag_assignments assignment
        JOIN posts post ON post.id = assignment.post_id
        WHERE assignment.tag_id = tag.id
          AND post.status = 'published' AND post.deleted_at IS NULL)::bigint AS post_count
FROM blog_tags tag`

func (p *PG) CreateCategory(ctx context.Context, name, slug, parentID, description string) (string, error) {
	return p.CreateCategoryWithHook(ctx, name, slug, parentID, description, nil)
}

func (p *PG) CreateCategoryWithHook(ctx context.Context, name, slug, parentID, description string, hook CreateTransactionHook) (string, error) {
	id := uuid.Must(uuid.NewV7()).String()
	err := p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		catalogID, err := blogCatalogID(ctx, tx)
		if err != nil {
			return err
		}
		var rows []struct {
			ID string `orm:"id"`
		}
		if err := tx.Ctx(ctx).Raw(`
INSERT INTO blog_categories (id, catalog_id, parent_id, current_name, current_slug, description, status, first_activated_at)
VALUES (?::uuid, ?::uuid, NULLIF(?, '')::uuid, ?, ?, ?, 'active', NOW())
RETURNING id::text AS id`, id, catalogID, parentID, name, slug, description).Scan(&rows); err != nil {
			return gerror.Wrap(err, "create blog category")
		}
		id = rows[0].ID
		if err := bumpBlogClassificationRevision(ctx, tx, catalogID); err != nil {
			return err
		}
		return runCreateTransactionHook(ctx, tx, id, hook)
	})
	return id, err
}

func (p *PG) CreateTag(ctx context.Context, name, slug, description, lookupKey string) (string, error) {
	return p.CreateTagWithHook(ctx, name, slug, description, lookupKey, nil)
}

func (p *PG) CreateTagWithHook(ctx context.Context, name, slug, description, lookupKey string, hook CreateTransactionHook) (string, error) {
	id := uuid.Must(uuid.NewV7()).String()
	err := p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		catalogID, err := blogCatalogID(ctx, tx)
		if err != nil {
			return err
		}
		var rows []struct {
			ID string `orm:"id"`
		}
		if err := tx.Ctx(ctx).Raw(`
INSERT INTO blog_tags (id, catalog_id, current_name, current_slug, description, status, first_activated_at)
VALUES (?::uuid, ?::uuid, ?, ?, ?, 'active', NOW())
RETURNING id::text AS id`, id, catalogID, name, slug, description).Scan(&rows); err != nil {
			return gerror.Wrap(err, "create blog tag")
		}
		id = rows[0].ID
		if _, err := tx.Ctx(ctx).Exec(`
INSERT INTO blog_tag_lookup_entries (catalog_id, lookup_key, target_tag_id, kind, display_value)
VALUES (?::uuid, ?, ?::uuid, 'canonical', ?)`, catalogID, lookupKey, id, name); err != nil {
			return gerror.Wrap(err, "register blog canonical tag")
		}
		if err := bumpBlogClassificationRevision(ctx, tx, catalogID); err != nil {
			return err
		}
		return runCreateTransactionHook(ctx, tx, id, hook)
	})
	return id, err
}

func (p *PG) GetTaxonomy(ctx context.Context, id string) (*model.Taxonomy, error) {
	var value *model.Taxonomy
	err := p.db.Ctx(ctx).Raw(`SELECT * FROM (`+taxonomyProjectionSQL+`) projection WHERE id = ?`, id).Scan(&value)
	return value, err
}

func (p *PG) GetTaxonomyBySlug(ctx context.Context, kind, slug string) (*model.Taxonomy, error) {
	var value *model.Taxonomy
	err := p.db.Ctx(ctx).Raw(`SELECT * FROM (`+taxonomyProjectionSQL+`) projection
WHERE taxonomy = ? AND slug = ? AND status <> 'replaced'`, kind, slug).Scan(&value)
	return value, err
}

func (p *PG) ListTaxonomies(ctx context.Context, kind string) ([]*model.Taxonomy, error) {
	items, _, err := p.ListTaxonomiesPage(ctx, TaxonomyListFilter{Kind: kind}, 0, 0)
	return items, err
}

func (p *PG) ListTaxonomiesPage(ctx context.Context, filter TaxonomyListFilter, limit, offset int) ([]*model.Taxonomy, int, error) {
	where := []string{"status <> 'replaced'"}
	args := make([]any, 0, 5)
	if filter.Kind != "" {
		where = append(where, "taxonomy = ?")
		args = append(args, filter.Kind)
	}
	if keyword := strings.TrimSpace(filter.Q); keyword != "" {
		where = append(where, "(name ILIKE ? OR slug ILIKE ? OR description ILIKE ?)")
		like := "%" + keyword + "%"
		args = append(args, like, like, like)
	}
	base := ` FROM (` + taxonomyProjectionSQL + `) projection WHERE ` + strings.Join(where, " AND ")
	count, err := p.db.GetValue(ctx, `SELECT COUNT(*)`+base, args...)
	if err != nil {
		return nil, 0, err
	}
	query := `SELECT *` + base + ` ORDER BY ` + taxonomyListOrder(filter.Sort, filter.Direction)
	rowArgs := append([]any(nil), args...)
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		rowArgs = append(rowArgs, limit, offset)
	}
	var values []*model.Taxonomy
	if err := p.db.Ctx(ctx).Raw(query, rowArgs...).Scan(&values); err != nil {
		return nil, 0, err
	}
	if values == nil {
		values = []*model.Taxonomy{}
	}
	return values, count.Int(), nil
}

func taxonomyListOrder(sortBy, direction string) string {
	dir := "ASC"
	if strings.EqualFold(direction, "desc") {
		dir = "DESC"
	}
	switch strings.TrimSpace(sortBy) {
	case "postCount", "count":
		return "post_count " + dir + ", name ASC, id ASC"
	case "slug":
		return "slug " + dir + ", name ASC, id ASC"
	default:
		return "name " + dir + ", id ASC"
	}
}

func (p *PG) TaxonomiesByIDs(ctx context.Context, ids []string) ([]*model.Taxonomy, error) {
	if len(ids) == 0 {
		return []*model.Taxonomy{}, nil
	}
	var values []*model.Taxonomy
	if err := p.db.Ctx(ctx).Raw(`SELECT * FROM (`+taxonomyProjectionSQL+`) projection WHERE id = ANY(?::text[])`, uuidArrayLiteral(ids)).Scan(&values); err != nil {
		return nil, err
	}
	if values == nil {
		values = []*model.Taxonomy{}
	}
	return values, nil
}

func (p *PG) CountTaxonomiesByIDs(ctx context.Context, ids []string) (int, error) {
	values, err := p.TaxonomiesByIDs(ctx, ids)
	return len(values), err
}

func (p *PG) SetPostClassification(ctx context.Context, postID string, assignments classification.ClassificationAssignments) error {
	return p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Model("blog_post_category_assignments").Ctx(ctx).Where("post_id", postID).Delete(); err != nil {
			return err
		}
		if _, err := tx.Model("blog_post_tag_assignments").Ctx(ctx).Where("post_id", postID).Delete(); err != nil {
			return err
		}
		for _, value := range assignments.Categories {
			if _, err := tx.Model("blog_post_category_assignments").Ctx(ctx).Data(g.Map{"post_id": postID, "category_id": value.CategoryID}).Insert(); err != nil {
				return err
			}
		}
		for _, value := range assignments.Tags {
			if _, err := tx.Model("blog_post_tag_assignments").Ctx(ctx).Data(g.Map{"post_id": postID, "tag_id": value.TagID}).Insert(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (p *PG) RelatedPosts(ctx context.Context, postID string, limit int) ([]*model.Post, error) {
	var values []*model.Post
	err := p.db.Ctx(ctx).Raw(`
WITH source_taxonomies AS (
    SELECT category_id AS taxonomy_id, 'category'::text AS kind
    FROM blog_post_category_assignments WHERE post_id = ?::uuid
    UNION ALL
    SELECT tag_id, 'tag'::text FROM blog_post_tag_assignments WHERE post_id = ?::uuid
), candidate_taxonomies AS (
    SELECT post_id, category_id AS taxonomy_id, 'category'::text AS kind FROM blog_post_category_assignments
    UNION ALL
    SELECT post_id, tag_id, 'tag'::text FROM blog_post_tag_assignments
)
SELECT post.*, COUNT(*) AS overlap
FROM posts post
JOIN candidate_taxonomies candidate ON candidate.post_id = post.id
JOIN source_taxonomies source ON source.taxonomy_id = candidate.taxonomy_id AND source.kind = candidate.kind
WHERE post.id <> ?::uuid AND post.status = 'published' AND post.deleted_at IS NULL
GROUP BY post.id
ORDER BY overlap DESC, post.published_at DESC
LIMIT ?`, postID, postID, postID, limit).Scan(&values)
	if values == nil {
		values = []*model.Post{}
	}
	return values, err
}

func (p *PG) PostIDsByTaxonomySlug(ctx context.Context, slug string) ([]string, error) {
	var rows []struct {
		ID string `orm:"id"`
	}
	err := p.db.Ctx(ctx).Raw(`
WITH RECURSIVE matching_categories AS (
    SELECT id FROM blog_categories WHERE current_slug = ? AND status = 'active'
    UNION ALL
    SELECT child.id FROM blog_categories child
    JOIN matching_categories parent ON child.parent_id = parent.id
    WHERE child.status = 'active'
), matching_tags AS (
    SELECT id FROM blog_tags WHERE current_slug = ? AND status = 'active'
), matching_posts AS (
    SELECT assignment.post_id AS id FROM blog_post_category_assignments assignment
    WHERE assignment.category_id IN (SELECT id FROM matching_categories)
    UNION
    SELECT assignment.post_id FROM blog_post_tag_assignments assignment
    WHERE assignment.tag_id IN (SELECT id FROM matching_tags)
)
SELECT id::text AS id FROM matching_posts ORDER BY id`, slug, slug).Scan(&rows)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids, nil
}

func (p *PG) GetPostTaxonomies(ctx context.Context, postID string) ([]*model.Taxonomy, error) {
	values, err := p.GetPostTaxonomiesByPostIDs(ctx, []string{postID})
	return values[postID], err
}

func (p *PG) GetPostTaxonomiesByPostIDs(ctx context.Context, postIDs []string) (map[string][]*model.Taxonomy, error) {
	result := make(map[string][]*model.Taxonomy, len(postIDs))
	for _, id := range postIDs {
		result[id] = []*model.Taxonomy{}
	}
	if len(postIDs) == 0 {
		return result, nil
	}
	var rows []struct {
		PostID      string `orm:"post_id"`
		ID          string `orm:"id"`
		Taxonomy    string `orm:"taxonomy"`
		Name        string `orm:"name"`
		Slug        string `orm:"slug"`
		Description string `orm:"description"`
		ParentID    string `orm:"parent_id"`
		Status      string `orm:"status"`
	}
	if err := p.db.Ctx(ctx).Raw(`
SELECT assignment.post_id::text AS post_id, category.id::text AS id, 'category'::text AS taxonomy,
       category.current_name AS name, category.current_slug AS slug, category.description,
       COALESCE(category.parent_id::text, '') AS parent_id, category.status
FROM blog_post_category_assignments assignment
JOIN blog_categories category ON category.id = assignment.category_id
WHERE assignment.post_id = ANY(?::uuid[])
UNION ALL
SELECT assignment.post_id::text, tag.id::text, 'tag'::text,
       tag.current_name, tag.current_slug, tag.description, ''::text, tag.status
FROM blog_post_tag_assignments assignment
JOIN blog_tags tag ON tag.id = assignment.tag_id
WHERE assignment.post_id = ANY(?::uuid[])
ORDER BY post_id, taxonomy, name, id`, uuidArrayLiteral(postIDs), uuidArrayLiteral(postIDs)).Scan(&rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.PostID] = append(result[row.PostID], &model.Taxonomy{
			ID: row.ID, Taxonomy: row.Taxonomy, Name: row.Name, Slug: row.Slug,
			Description: row.Description, ParentID: row.ParentID, Status: row.Status,
		})
	}
	return result, nil
}

func (p *PG) TaxonomyChildCount(ctx context.Context, id string) (int, error) {
	return p.db.Model("blog_categories").Ctx(ctx).Where("parent_id", id).Count()
}

func (p *PG) UpdateTaxonomy(ctx context.Context, value *model.Taxonomy, fields g.Map, lookupKey string) error {
	return p.UpdateTaxonomyWithHook(ctx, value, fields, lookupKey, nil)
}

func (p *PG) UpdateTaxonomyWithHook(ctx context.Context, value *model.Taxonomy, fields g.Map, lookupKey string, hook TransactionHook) error {
	table := "blog_categories"
	if value.Taxonomy == "tag" {
		table = "blog_tags"
		delete(fields, "parent_id")
	}
	return p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		catalogID, err := blogCatalogID(ctx, tx)
		if err != nil {
			return err
		}
		if len(fields) != 0 {
			fields["updated_at"] = gdb.Raw("NOW()")
			if _, err := tx.Model(table).Ctx(ctx).Where("id", value.ID).Data(fields).Update(); err != nil {
				return err
			}
		}
		if value.Taxonomy == "tag" && lookupKey != "" {
			if _, err := tx.Ctx(ctx).Exec(`UPDATE blog_tag_lookup_entries SET kind = 'alias' WHERE catalog_id = ?::uuid AND target_tag_id = ?::uuid AND kind = 'canonical'`, catalogID, value.ID); err != nil {
				return err
			}
			name := value.Name
			if raw, ok := fields["current_name"].(string); ok {
				name = raw
			}
			if _, err := tx.Ctx(ctx).Exec(`
INSERT INTO blog_tag_lookup_entries (catalog_id, lookup_key, target_tag_id, kind, display_value)
VALUES (?::uuid, ?, ?::uuid, 'canonical', ?)
ON CONFLICT (catalog_id, lookup_key) DO UPDATE
SET kind = 'canonical', display_value = EXCLUDED.display_value
WHERE blog_tag_lookup_entries.target_tag_id = EXCLUDED.target_tag_id`, catalogID, lookupKey, value.ID, name); err != nil {
				return err
			}
		}
		if err := bumpBlogClassificationRevision(ctx, tx, catalogID); err != nil {
			return err
		}
		return runTransactionHook(ctx, tx, hook)
	})
}

func (p *PG) DeleteTaxonomy(ctx context.Context, value *model.Taxonomy) error {
	return p.DeleteTaxonomyWithHook(ctx, value, nil)
}

func (p *PG) DeleteTaxonomyWithHook(ctx context.Context, value *model.Taxonomy, hook TransactionHook) error {
	return p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		catalogID, err := blogCatalogID(ctx, tx)
		if err != nil {
			return err
		}
		if value.Taxonomy == "category" {
			if _, err := tx.Model("blog_post_category_assignments").Ctx(ctx).Where("category_id", value.ID).Delete(); err != nil {
				return err
			}
			if _, err := tx.Model("blog_categories").Ctx(ctx).Where("id", value.ID).Delete(); err != nil {
				return err
			}
		} else {
			if _, err := tx.Model("blog_post_tag_assignments").Ctx(ctx).Where("tag_id", value.ID).Delete(); err != nil {
				return err
			}
			if _, err := tx.Model("blog_tag_lookup_entries").Ctx(ctx).Where("target_tag_id", value.ID).Delete(); err != nil {
				return err
			}
			if _, err := tx.Model("blog_tags").Ctx(ctx).Where("id", value.ID).Delete(); err != nil {
				return err
			}
		}
		if err := bumpBlogClassificationRevision(ctx, tx, catalogID); err != nil {
			return err
		}
		return runTransactionHook(ctx, tx, hook)
	})
}

func (p *PG) MergeTaxonomy(ctx context.Context, source, target *model.Taxonomy) error {
	return p.MergeTaxonomyWithHook(ctx, source, target, nil)
}

func (p *PG) MergeTaxonomyWithHook(ctx context.Context, source, target *model.Taxonomy, hook TransactionHook) error {
	return p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		catalogID, err := blogCatalogID(ctx, tx)
		if err != nil {
			return err
		}
		if source.Taxonomy == "category" {
			if _, err := tx.Ctx(ctx).Exec(`
INSERT INTO blog_post_category_assignments (post_id, category_id)
SELECT post_id, ?::uuid FROM blog_post_category_assignments WHERE category_id = ?::uuid
ON CONFLICT DO NOTHING`, target.ID, source.ID); err != nil {
				return err
			}
			if _, err := tx.Model("blog_post_category_assignments").Ctx(ctx).Where("category_id", source.ID).Delete(); err != nil {
				return err
			}
			if _, err := tx.Model("blog_categories").Ctx(ctx).Where("parent_id", source.ID).Data(g.Map{"parent_id": target.ID}).Update(); err != nil {
				return err
			}
			if _, err := tx.Model("blog_categories").Ctx(ctx).Where("id", source.ID).Data(g.Map{"parent_id": nil, "status": "replaced", "replacement_id": target.ID, "updated_at": gdb.Raw("NOW()")}).Update(); err != nil {
				return err
			}
		} else {
			if _, err := tx.Ctx(ctx).Exec(`
INSERT INTO blog_post_tag_assignments (post_id, tag_id)
SELECT post_id, ?::uuid FROM blog_post_tag_assignments WHERE tag_id = ?::uuid
ON CONFLICT DO NOTHING`, target.ID, source.ID); err != nil {
				return err
			}
			if _, err := tx.Model("blog_post_tag_assignments").Ctx(ctx).Where("tag_id", source.ID).Delete(); err != nil {
				return err
			}
			if _, err := tx.Model("blog_tag_lookup_entries").Ctx(ctx).Where("target_tag_id", source.ID).Data(g.Map{"target_tag_id": target.ID, "kind": "alias"}).Update(); err != nil {
				return err
			}
			if _, err := tx.Model("blog_tags").Ctx(ctx).Where("id", source.ID).Data(g.Map{"status": "replaced", "replacement_id": target.ID, "updated_at": gdb.Raw("NOW()")}).Update(); err != nil {
				return err
			}
		}
		if err := bumpBlogClassificationRevision(ctx, tx, catalogID); err != nil {
			return err
		}
		return runTransactionHook(ctx, tx, hook)
	})
}

func (p *PG) ClassificationSnapshot(ctx context.Context) (classification.Snapshot, error) {
	type catalogRow struct {
		ID       string `orm:"id"`
		Revision uint64 `orm:"revision"`
	}
	type policyRow struct {
		Key             string `orm:"policy_key"`
		SchemaVersion   uint16 `orm:"schema_version"`
		PolicyRevision  uint64 `orm:"policy_revision"`
		CategoryPolicy  string `orm:"category_policy"`
		FacetPolicies   string `orm:"facet_policies"`
		TagPolicy       string `orm:"tag_policy"`
		DiscoveryPolicy string `orm:"discovery_policy"`
	}
	var snapshot classification.Snapshot
	err := p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		scoped := tx.Ctx(ctx)
		if _, err := scoped.Exec(`SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
			return err
		}
		var catalog *catalogRow
		if err := scoped.Raw(`SELECT id::text AS id, revision FROM blog_classification_catalogs WHERE catalog_key = 'blog'`).Scan(&catalog); err != nil {
			return err
		}
		if catalog == nil {
			return fmt.Errorf("blog classification catalog is not seeded")
		}
		snapshot = classification.Snapshot{CatalogID: catalog.ID, Revision: catalog.Revision, Facets: []classification.Facet{}, FacetValues: []classification.FacetValue{}}
		if err := scoped.Raw(`
SELECT id::text AS id, COALESCE(parent_id::text, '') AS parent_id,
       current_slug AS slug, current_name AS name, status, editorial_position,
       COALESCE(replacement_id::text, '') AS replacement_id
FROM blog_categories WHERE catalog_id = ?::uuid ORDER BY id`, catalog.ID).Scan(&snapshot.Categories); err != nil {
			return err
		}
		var rows []policyRow
		if err := scoped.Raw(`
SELECT policy_key, schema_version, policy_revision,
       category_policy::text AS category_policy, facet_policies::text AS facet_policies,
       tag_policy::text AS tag_policy, discovery_policy::text AS discovery_policy
FROM blog_classification_policy_profiles WHERE catalog_id = ?::uuid ORDER BY policy_key`, catalog.ID).Scan(&rows); err != nil {
			return err
		}
		for _, row := range rows {
			policy := classification.PolicyProfile{Key: row.Key, SchemaVersion: row.SchemaVersion, PolicyRevision: row.PolicyRevision}
			if err := json.Unmarshal([]byte(row.CategoryPolicy), &policy.Category); err != nil {
				return err
			}
			if err := json.Unmarshal([]byte(row.FacetPolicies), &policy.Facets); err != nil {
				return err
			}
			if err := json.Unmarshal([]byte(row.TagPolicy), &policy.Tags); err != nil {
				return err
			}
			if err := json.Unmarshal([]byte(row.DiscoveryPolicy), &policy.Discovery); err != nil {
				return err
			}
			snapshot.Policies = append(snapshot.Policies, policy)
		}
		return nil
	})
	return snapshot, err
}

func (p *PG) ClassificationTagMatches(ctx context.Context, lookups []classification.TagLookupRequest) ([]classification.TagMatch, string, error) {
	if len(lookups) == 0 {
		return []classification.TagMatch{}, "", nil
	}
	keys := make([]string, 0, len(lookups))
	for _, lookup := range lookups {
		keys = append(keys, lookup.LookupKey)
	}
	keyArray, err := pq.StringArray(keys).Value()
	if err != nil {
		return nil, "", err
	}
	var rows []struct {
		LookupKey string `orm:"lookup_key"`
		Kind      string `orm:"kind"`
		TagID     string `orm:"tag_id"`
		Revision  uint64 `orm:"revision"`
	}
	if err := p.db.Ctx(ctx).Raw(`
WITH requested AS (
    SELECT lookup_key, ordinal FROM unnest(?::text[]) WITH ORDINALITY AS value(lookup_key, ordinal)
), catalog AS (
    SELECT id, revision FROM blog_classification_catalogs WHERE catalog_key = 'blog'
)
SELECT requested.lookup_key,
       CASE
           WHEN target.id IS NULL THEN 'not_found'
           WHEN target.status = 'inactive' THEN 'inactive'
           WHEN target.status = 'replaced' THEN 'replacement'
           ELSE entry.kind
       END AS kind,
       COALESCE(CASE WHEN target.status = 'replaced' THEN target.replacement_id ELSE target.id END::text, '') AS tag_id,
       catalog.revision
FROM requested CROSS JOIN catalog
LEFT JOIN blog_tag_lookup_entries entry ON entry.catalog_id = catalog.id AND entry.lookup_key = requested.lookup_key
LEFT JOIN blog_tags target ON target.id = entry.target_tag_id
ORDER BY requested.ordinal`, keyArray).Scan(&rows); err != nil {
		return nil, "", err
	}
	matches := make([]classification.TagMatch, 0, len(rows))
	var revision uint64
	for _, row := range rows {
		revision = row.Revision
		matches = append(matches, classification.TagMatch{LookupKey: row.LookupKey, Kind: classification.TagMatchKind(row.Kind), TagID: row.TagID})
	}
	return matches, fmt.Sprintf("%d", revision), nil
}

func blogCatalogID(ctx context.Context, tx gdb.TX) (string, error) {
	value, err := tx.Ctx(ctx).GetValue(`SELECT id::text FROM blog_classification_catalogs WHERE catalog_key = 'blog'`)
	if err != nil {
		return "", err
	}
	if value.IsEmpty() {
		return "", fmt.Errorf("blog classification catalog is not seeded")
	}
	return value.String(), nil
}

func bumpBlogClassificationRevision(ctx context.Context, tx gdb.TX, catalogID string) error {
	_, err := tx.Ctx(ctx).Exec(`UPDATE blog_classification_catalogs SET revision = revision + 1, updated_at = NOW() WHERE id = ?::uuid`, catalogID)
	return err
}

func uuidArrayLiteral(values []string) string {
	return "{" + strings.Join(values, ",") + "}"
}
