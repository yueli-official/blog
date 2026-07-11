package dao

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"

	"platform/products/blog/api/internal/model"
)

const (
	tTerms      = "terms"
	tTaxonomies = "taxonomies"
	tObjTax     = "object_taxonomies"
)

// UpsertTerm returns the id of the term with this slug, inserting it if absent.
func (p *PG) UpsertTerm(ctx context.Context, name, slug string) (string, error) {
	var t *model.Term
	if err := p.db.Model(tTerms).Ctx(ctx).Where("slug", slug).Limit(1).Scan(&t); err != nil {
		return "", err
	}
	if t != nil {
		return t.ID, nil
	}
	id := uuid.NewString()
	if _, err := p.db.Model(tTerms).Ctx(ctx).Data(g.Map{"id": id, "name": name, "slug": slug}).Insert(); err != nil {
		// lost a race? the slug now exists — re-read.
		var t2 *model.Term
		if e := p.db.Model(tTerms).Ctx(ctx).Where("slug", slug).Limit(1).Scan(&t2); e == nil && t2 != nil {
			return t2.ID, nil
		}
		return "", err
	}
	return id, nil
}

// UpsertTaxonomy returns the id of the (term, kind) taxonomy, inserting if absent.
func (p *PG) UpsertTaxonomy(ctx context.Context, termID, kind, description, parentID string) (string, error) {
	var tx *model.Taxonomy
	if err := p.db.Model(tTaxonomies).Ctx(ctx).
		Where("term_id", termID).Where("taxonomy", kind).Limit(1).Scan(&tx); err != nil {
		return "", err
	}
	if tx != nil {
		return tx.ID, nil
	}
	id := uuid.NewString()
	data := g.Map{"id": id, "term_id": termID, "taxonomy": kind, "description": description}
	if parentID != "" {
		data["parent_id"] = parentID
	}
	if _, err := p.db.Model(tTaxonomies).Ctx(ctx).Data(data).Insert(); err != nil {
		return "", err
	}
	return id, nil
}

// GetTaxonomy returns one taxonomy joined with its term (name/slug), or (nil,nil).
func (p *PG) GetTaxonomy(ctx context.Context, id string) (*model.Taxonomy, error) {
	var tx *model.Taxonomy
	err := p.db.Model(tTaxonomies+" tx").Ctx(ctx).
		LeftJoin(tTerms+" t", "t.id = tx.term_id").
		Fields("tx.*, t.name, t.slug").
		Where("tx.id", id).Limit(1).Scan(&tx)
	return tx, err
}

// ListTaxonomies returns taxonomies (optionally filtered by kind), joined with
// their term name/slug, ordered by name.
func (p *PG) ListTaxonomies(ctx context.Context, kind string) ([]*model.Taxonomy, error) {
	m := p.db.Model(tTaxonomies+" tx").Ctx(ctx).
		LeftJoin(tTerms+" t", "t.id = tx.term_id").
		Fields("tx.*, t.name, t.slug")
	if kind != "" {
		m = m.Where("tx.taxonomy", kind)
	}
	var out []*model.Taxonomy
	err := m.OrderAsc("t.name").Scan(&out)
	return out, err
}

// CountTaxonomiesByIDs returns how many of the given taxonomy ids exist.
func (p *PG) CountTaxonomiesByIDs(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	return p.db.Model(tTaxonomies).Ctx(ctx).WhereIn("id", ids).Count()
}

// SetPostTaxonomies replaces a post's taxonomy assignments (delete then insert).
func (p *PG) SetPostTaxonomies(ctx context.Context, postID string, taxIDs []string) error {
	return p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Model(tObjTax).Ctx(ctx).Where("object_id", postID).Delete(); err != nil {
			return err
		}
		for i, tid := range taxIDs {
			if _, err := tx.Model(tObjTax).Ctx(ctx).Data(g.Map{
				"object_id": postID, "taxonomy_id": tid, "sort_order": i,
			}).Insert(); err != nil {
				return err
			}
		}
		return nil
	})
}

// RelatedPosts returns up to `limit` published posts that share taxonomies with
// the given post (excluding itself), ranked by overlap count then recency.
// Returns an empty slice when the post has no taxonomies. The selected
// `COUNT(...) AS overlap` column is ignored by the Post scan (model has no such
// field); GROUP BY p.id is valid since id is the table's primary key.
func (p *PG) RelatedPosts(ctx context.Context, postID string, limit int) ([]*model.Post, error) {
	taxVals, err := p.db.Model(tObjTax).Ctx(ctx).
		Where("object_id", postID).Fields("taxonomy_id").Array()
	if err != nil {
		return nil, err
	}
	if len(taxVals) == 0 {
		return []*model.Post{}, nil
	}
	taxIDs := make([]string, 0, len(taxVals))
	for _, v := range taxVals {
		taxIDs = append(taxIDs, v.String())
	}
	var out []*model.Post
	err = p.db.Model(tPosts+" p").Ctx(ctx).
		LeftJoin(tObjTax+" ot", "ot.object_id = p.id").
		WhereIn("ot.taxonomy_id", taxIDs).
		Where("p.status", string(model.StatusPublished)).
		Where("p.deleted_at IS NULL").
		Where("p.id != ?", postID).
		Fields("p.*, COUNT(ot.taxonomy_id) AS overlap").
		Group("p.id").
		OrderDesc("overlap").
		OrderDesc("p.published_at").
		Limit(limit).
		Scan(&out)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []*model.Post{}
	}
	return out, nil
}

// PostIDsByTaxonomySlug returns the ids of posts tagged with the given term slug.
func (p *PG) PostIDsByTaxonomySlug(ctx context.Context, slug string) ([]string, error) {
	vals, err := p.db.Model(tObjTax+" ot").Ctx(ctx).
		LeftJoin(tTaxonomies+" tx", "tx.id = ot.taxonomy_id").
		LeftJoin(tTerms+" t", "t.id = tx.term_id").
		Where("t.slug", slug).
		Fields("ot.object_id").Array()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(vals))
	for _, v := range vals {
		ids = append(ids, v.String())
	}
	return ids, nil
}

// GetTaxonomyBySlug returns the taxonomy of the given kind whose term has this
// slug, joined with the term name/slug, or (nil, nil) when absent.
func (p *PG) GetTaxonomyBySlug(ctx context.Context, kind, slug string) (*model.Taxonomy, error) {
	var tx *model.Taxonomy
	err := p.db.Model(tTaxonomies+" tx").Ctx(ctx).
		LeftJoin(tTerms+" t", "t.id = tx.term_id").
		Fields("tx.*, t.name, t.slug").
		Where("tx.taxonomy", kind).Where("t.slug", slug).Limit(1).Scan(&tx)
	return tx, err
}

// GetPostTaxonomies returns a post's assigned taxonomies, joined with term
// name/slug, ordered by the saved sort_order. Never returns nil.
func (p *PG) GetPostTaxonomies(ctx context.Context, postID string) ([]*model.Taxonomy, error) {
	var out []*model.Taxonomy
	err := p.db.Model(tObjTax+" ot").Ctx(ctx).
		LeftJoin(tTaxonomies+" tx", "tx.id = ot.taxonomy_id").
		LeftJoin(tTerms+" t", "t.id = tx.term_id").
		Fields("tx.*, t.name, t.slug").
		Where("ot.object_id", postID).
		OrderAsc("ot.sort_order").Scan(&out)
	if out == nil {
		out = []*model.Taxonomy{}
	}
	return out, err
}

// TaxonomyPostCounts returns published-post counts keyed by taxonomy id (for tag
// cloud sizing + governance). Drafts/archived/soft-deleted are excluded.
func (p *PG) TaxonomyPostCounts(ctx context.Context) (map[string]int, error) {
	var rows []struct {
		TaxonomyID string `orm:"taxonomy_id"`
		C          int    `orm:"c"`
	}
	err := p.db.Model(tObjTax+" ot").Ctx(ctx).
		LeftJoin(tPosts+" p", "p.id = ot.object_id").
		Where("p.status", string(model.StatusPublished)).
		Where("p.deleted_at IS NULL").
		Fields("ot.taxonomy_id, COUNT(*) AS c").
		Group("ot.taxonomy_id").Scan(&rows)
	if err != nil {
		return nil, err
	}
	m := make(map[string]int, len(rows))
	for _, r := range rows {
		m[r.TaxonomyID] = r.C
	}
	return m, nil
}

// TaxonomyChildCount returns how many taxonomies have this one as their parent.
func (p *PG) TaxonomyChildCount(ctx context.Context, id string) (int, error) {
	return p.db.Model(tTaxonomies).Ctx(ctx).Where("parent_id", id).Count()
}

// UpdateTaxonomy applies term-level fields (name/slug) and taxonomy-level fields
// (description/parent_id) in one transaction; either map may be empty.
func (p *PG) UpdateTaxonomy(ctx context.Context, id, termID string, termFields, taxFields g.Map) error {
	return p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if len(termFields) > 0 {
			if _, err := tx.Model(tTerms).Ctx(ctx).Where("id", termID).Data(termFields).Update(); err != nil {
				return err
			}
		}
		if len(taxFields) > 0 {
			if _, err := tx.Model(tTaxonomies).Ctx(ctx).Where("id", id).Data(taxFields).Update(); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteTaxonomy removes a taxonomy (its object_taxonomies rows cascade via FK).
// The shared term row is left intact — it may back another kind and is reused by
// slug on the next CreateTaxonomy.
func (p *PG) DeleteTaxonomy(ctx context.Context, id string) error {
	_, err := p.db.Model(tTaxonomies).Ctx(ctx).Where("id", id).Delete()
	return err
}

// MergeTaxonomy re-points every post tagged with source onto target (skipping
// posts that already carry target, to avoid the (object_id, taxonomy_id) PK
// clash), re-parents source's children onto target, then deletes source.
func (p *PG) MergeTaxonomy(ctx context.Context, sourceID, targetID string) error {
	return p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		srcVals, err := tx.Model(tObjTax).Ctx(ctx).Where("taxonomy_id", sourceID).Fields("object_id").Array()
		if err != nil {
			return err
		}
		tgtVals, err := tx.Model(tObjTax).Ctx(ctx).Where("taxonomy_id", targetID).Fields("object_id").Array()
		if err != nil {
			return err
		}
		have := make(map[string]bool, len(tgtVals))
		for _, v := range tgtVals {
			have[v.String()] = true
		}
		if _, err := tx.Model(tObjTax).Ctx(ctx).Where("taxonomy_id", sourceID).Delete(); err != nil {
			return err
		}
		for _, v := range srcVals {
			oid := v.String()
			if have[oid] {
				continue
			}
			if _, err := tx.Model(tObjTax).Ctx(ctx).Data(g.Map{
				"object_id": oid, "taxonomy_id": targetID, "sort_order": 0,
			}).Insert(); err != nil {
				return err
			}
		}
		if _, err := tx.Model(tTaxonomies).Ctx(ctx).Where("parent_id", sourceID).Data(g.Map{"parent_id": targetID}).Update(); err != nil {
			return err
		}
		if _, err := tx.Model(tTaxonomies).Ctx(ctx).Where("id", sourceID).Delete(); err != nil {
			return err
		}
		return nil
	})
}
