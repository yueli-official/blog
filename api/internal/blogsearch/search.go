package blogsearch

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yueli-official/foundation/go/search"
)

const analyzer search.AnalyzerKey = "content-v1"

type Index struct {
	module   search.Module
	postgres *search.Postgres
}

func Definition() search.Definition {
	return search.Definition{
		Consumer: "blog.public", Version: 1,
		Analyzers: []search.AnalyzerDefinition{{
			Key: analyzer, QueryMode: search.QueryWeb,
			Required: []search.Capability{search.CapabilityFullText},
		}},
		Filters: []search.FilterDefinition{
			{Name: "id", MaxValues: 5000},
			{Name: "featured", MaxValues: 2},
			{Name: "pinned", MaxValues: 2},
		},
		Limits: search.Limits{MaxPageSize: 100},
	}
}

func NewPostgres(ctx context.Context, db *sql.DB, site string) (*Index, error) {
	catalog, err := search.Compile(Definition())
	if err != nil {
		return nil, err
	}
	module, err := search.NewPostgres(ctx, catalog, search.PostgresOptions{
		DB: db, InstanceKey: "blog." + site,
		AnalyzerBindings: map[search.AnalyzerKey]string{analyzer: "chinese_zh"},
	})
	if err != nil {
		return nil, err
	}
	return &Index{module: module, postgres: module}, nil
}

func NewMemory() *Index {
	return &Index{module: search.NewMemory(search.MustCompile(Definition()))}
}

type postRow struct {
	ID, Title, Excerpt, Content string
	Revision                    uint64
	Featured, Pinned            bool
	PublishedAt, UpdatedAt      time.Time
	Status                      string
	DeletedAt                   sql.NullTime
}

func (index *Index) Hook(id string) func(context.Context, *sql.Tx) error {
	return func(ctx context.Context, tx *sql.Tx) error {
		var row postRow
		err := tx.QueryRowContext(ctx, `
			SELECT id,title,excerpt,content,search_revision,featured,pinned,
				COALESCE(published_at,updated_at),updated_at,status,deleted_at
			FROM posts WHERE id=$1
		`, id).Scan(&row.ID, &row.Title, &row.Excerpt, &row.Content, &row.Revision,
			&row.Featured, &row.Pinned, &row.PublishedAt, &row.UpdatedAt, &row.Status, &row.DeletedAt)
		if err != nil {
			return err
		}
		if index.postgres == nil {
			return fmt.Errorf("blogsearch: transactional projection requires PostgreSQL")
		}
		projector, err := index.postgres.Bind(tx)
		if err != nil {
			return err
		}
		return index.applyRow(ctx, projector, row)
	}
}

func (index *Index) applyRow(ctx context.Context, projector search.Projector, row postRow) error {
	key := search.DocumentKey{Kind: "post", ID: search.DocumentID(row.ID)}
	var change search.Change
	if row.Status == "published" && !row.DeletedAt.Valid {
		change = search.Upsert(search.SourceDocument{
			Key: key, Revision: search.ProjectionRevision(row.Revision), Analyzer: analyzer,
			Title: row.Title, Summary: row.Excerpt, Body: row.Content, SortAt: row.PublishedAt.UTC(),
			Filters: search.FieldValues{
				"id": search.Keyword(row.ID), "featured": search.Keyword(fmt.Sprint(row.Featured)),
				"pinned": search.Keyword(fmt.Sprint(row.Pinned)),
			},
			Visibility: search.VisibilityReference{ResourceType: "blog.post", ResourceID: row.ID},
		})
	} else {
		change = search.Remove(key, search.ProjectionRevision(row.Revision))
	}
	_, err := projector.Apply(ctx, search.Batch{
		ID:      search.BatchID(fmt.Sprintf("blog.post.%s.%d", row.ID, row.Revision)),
		Changes: []search.Change{change},
	})
	return err
}

func (index *Index) Reconcile(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `
		SELECT id,title,excerpt,content,search_revision,featured,pinned,
			COALESCE(published_at,updated_at),updated_at,status,deleted_at
		FROM posts
	`)
	if err != nil {
		return err
	}
	var values []postRow
	for rows.Next() {
		var row postRow
		if err := rows.Scan(&row.ID, &row.Title, &row.Excerpt, &row.Content, &row.Revision,
			&row.Featured, &row.Pinned, &row.PublishedAt, &row.UpdatedAt, &row.Status, &row.DeletedAt); err != nil {
			_ = rows.Close()
			return err
		}
		values = append(values, row)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, row := range values {
		if err := index.applyRow(ctx, index.module, row); err != nil {
			return err
		}
	}
	return nil
}

func (index *Index) Search(ctx context.Context, text string, ids []string, featured, pinned bool, limit, offset int) (search.Page, error) {
	filters := []search.Filter{}
	if ids != nil {
		if len(ids) == 0 {
			return search.Page{}, nil
		}
		filters = append(filters, search.Any("id", ids...))
	}
	if featured {
		filters = append(filters, search.Equal("featured", "true"))
	}
	if pinned {
		filters = append(filters, search.Equal("pinned", "true"))
	}
	needed := offset + limit
	var result search.Page
	cursor := search.Cursor("")
	for len(result.Hits) < needed {
		page, err := index.module.Search(ctx, search.Query{
			Text: text, Analyzer: analyzer, Filters: filters,
			Page: search.PageRequest{Size: min(100, needed-len(result.Hits)), Cursor: cursor},
		})
		if err != nil {
			return search.Page{}, err
		}
		result.Plan, result.Total = page.Plan, page.Total
		result.Hits = append(result.Hits, page.Hits...)
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if offset >= len(result.Hits) {
		result.Hits = nil
	} else {
		result.Hits = result.Hits[offset:min(offset+limit, len(result.Hits))]
	}
	return result, nil
}
