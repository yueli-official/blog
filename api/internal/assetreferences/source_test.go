package assetreferences

import (
	"context"
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/yueli-official/asset/referencesync"
	"os"
	"testing"
)

// Real PostgreSQL query tests use transaction-local tables only; no production data is touched.
func TestCommittedUsageLifecycle(t *testing.T) {
	dsn := os.Getenv("REFERENCE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set REFERENCE_TEST_DATABASE_URL")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	exec := func(query string) {
		t.Helper()
		if _, err := tx.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	exec(`CREATE TEMP TABLE posts(id text,title text,content text,cover_asset_id text,deleted_at timestamptz) ON COMMIT DROP; CREATE TEMP TABLE series(id text,name text,slug text,cover_asset_id text) ON COMMIT DROP; CREATE TEMP TABLE post_seo(post_id text,og_image text) ON COMMIT DROP;INSERT INTO posts VALUES ('a','A','![x](/media/KeyA)','asset-a',NULL),('b','B','![x](/media/KeyA)','asset-b',NULL)`)
	read := func(want int) []referencesync.Reference {
		t.Helper()
		snapshots, err := Source()(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range snapshots {
			if s.RefType == "post-body" {
				if len(s.References) != want {
					t.Fatalf("references=%d want=%d", len(s.References), want)
				}
				return s.References
			}
		}
		t.Fatal("snapshot missing")
		return nil
	}
	read(2)
	read(2) // Backfill and repeat keep two independent users of the same asset.
	exec(`UPDATE posts SET deleted_at=now() WHERE id='a'`)
	read(1)
	exec(`UPDATE posts SET deleted_at=NULL WHERE id='a'`)
	read(2)
	exec(`UPDATE posts SET content='![new](/media/KeyB)' WHERE id='a'`)
	for _, ref := range read(2) {
		if ref.RefID == "a" && ref.AssetID != "KeyB" && ref.MediaKey != "KeyB" {
			t.Fatalf("replacement retained old asset: %+v", ref)
		}
	}

	check := func(kind string, want int) {
		t.Helper()
		snapshots, err := Source()(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}
		for _, snapshot := range snapshots {
			if snapshot.RefType == kind {
				if len(snapshot.References) != want {
					t.Fatalf("%s count=%d want=%d", kind, len(snapshot.References), want)
				}
				return
			}
		}
		t.Fatalf("missing %s", kind)
	}

	exec(`INSERT INTO series VALUES ('s','Series','series','SeriesCover');INSERT INTO post_seo VALUES ('a','/media/OG')`)
	check("series-cover", 1)
	check("post-seo", 1)
	check("post-cover", 2)
	exec(`UPDATE posts SET deleted_at=now() WHERE id='a';DELETE FROM series`)
	check("series-cover", 0)
	check("post-seo", 0)
	check("post-cover", 1)
}
