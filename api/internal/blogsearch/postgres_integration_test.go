package blogsearch

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/yueli-official/foundation/go/identifier"
)

func TestPostgresPostAndProjectionCommitOrRollbackTogether(t *testing.T) {
	dsn := os.Getenv("SEARCH_CONSUMER_PG_DSN")
	if dsn == "" {
		t.Skip("SEARCH_CONSUMER_PG_DSN is not set")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	site := "test-" + identifier.MustNew().String()
	index, err := NewPostgres(ctx, db, site)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM search_instances WHERE instance_key=$1`, "blog."+site)
	})

	rolledBackID := identifier.MustNew().String()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO posts (id,author_id,title,slug,content,status,published_at)
		VALUES ($1::uuid,'author','Rollback Search Token','rollback-search-token','body','published',NOW())
	`, rolledBackID); err != nil {
		t.Fatal(err)
	}
	if err := index.Hook(rolledBackID)(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	page, err := index.Search(ctx, "Rollback Search Token", nil, false, false, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Hits) != 0 {
		t.Fatalf("rolled-back post remained searchable: %#v", page.Hits)
	}

	committedID := identifier.MustNew().String()
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO posts (id,author_id,title,slug,content,status,published_at)
		VALUES ($1::uuid,'author','Committed Search Token','committed-search-token','body','published',NOW())
	`, committedID); err != nil {
		t.Fatal(err)
	}
	if err := index.Hook(committedID)(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, `DELETE FROM posts WHERE id=$1::uuid`, committedID) })

	page, err = index.Search(ctx, "Committed Search Token", nil, false, false, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Hits) != 1 || string(page.Hits[0].Key.ID) != committedID {
		t.Fatalf("committed search hits = %#v", page.Hits)
	}
}
