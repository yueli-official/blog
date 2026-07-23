package blogurls

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/yueli-official/foundation/go/urllifecycle"
)

func TestPostgresProductAndURLRollbackTogether(t *testing.T) {
	dsn := os.Getenv("URL_LIFECYCLE_CONSUMER_PG_DSN")
	if dsn == "" {
		t.Skip("URL_LIFECYCLE_CONSUMER_PG_DSN is not set")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	lifecycle, err := NewPostgres(ctx, db, "blog-test:"+uuid.NewString(), "https://blog.test")
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO posts (id, author_id, title, slug, content, status, published_at)
VALUES ($1::uuid, 'author', 'Atomic', 'atomic', 'body', 'published', NOW())`, id); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Change(ctx, tx,
		State{ID: id, Kind: PostKind},
		State{ID: id, Kind: PostKind, Slug: "atomic", Published: true},
	); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM posts WHERE id = $1::uuid`, id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("product row survived rollback")
	}
	resolution, err := lifecycle.Resolver().Resolve(ctx, urllifecycle.Lookup{EscapedPath: "/posts/atomic"})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Kind != urllifecycle.ResolutionUnknown {
		t.Fatalf("URL state survived rollback: %#v", resolution)
	}
}
