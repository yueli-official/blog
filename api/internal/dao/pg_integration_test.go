package dao_test

// Posts DAO round-trip against a live PG (database `blog`). Skipped unless
// BLOG_PG_HOST is set:
//
//	BLOG_PG_HOST=192.168.5.5 BLOG_PG_USER=postgres BLOG_PG_PASS=postgres \
//	  go test -run TestPGPosts ./products/blog/api/internal/dao/...

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/test/gtest"
	_ "github.com/lib/pq"

	"platform/products/blog/api/internal/dao"
	"platform/products/blog/api/internal/model"
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func TestPGPosts(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		host := os.Getenv("BLOG_PG_HOST")
		if host == "" {
			t.Skip("set BLOG_PG_HOST to run the blog DAO integration test")
		}
		port, user, pass := envOr("BLOG_PG_PORT", "5432"), envOr("BLOG_PG_USER", "postgres"), os.Getenv("BLOG_PG_PASS")
		ctx := context.Background()

		sdb, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=blog sslmode=disable", host, port, user, pass))
		t.AssertNil(err)
		resetSchema(t, sdb)
		sdb.Close()

		db, err := gdb.New(gdb.ConfigNode{Type: "pgsql", Host: host, Port: port, User: user, Pass: pass, Name: "blog"})
		t.AssertNil(err)
		d := dao.NewPG(db)

		// Insert → GetByID round trip
		post := &model.Post{AuthorID: "author-1", Title: "Hello", Slug: "hello", Content: "body", Status: model.StatusPublished, PublishedAt: gtime.Now()}
		t.AssertNil(d.Insert(ctx, post))
		t.AssertNE(post.ID, "")
		got, err := d.GetByID(ctx, post.ID)
		t.AssertNil(err)
		t.AssertNE(got, nil)
		t.Assert(got.Title, "Hello")
		t.Assert(got.PostType, "post") // defaulted

		// duplicate slug → ErrSlugTaken
		dup := &model.Post{AuthorID: "author-1", Title: "Hello again", Slug: "hello", Status: model.StatusDraft}
		t.Assert(d.Insert(ctx, dup), dao.ErrSlugTaken)

		// a draft is excluded from the published List
		draft := &model.Post{AuthorID: "author-1", Title: "Draft", Slug: "draft-1", Status: model.StatusDraft}
		t.AssertNil(d.Insert(ctx, draft))
		items, total, err := d.List(ctx, dao.ListFilter{}, 20, 0)
		t.AssertNil(err)
		t.Assert(total >= 1, true)
		for _, it := range items {
			t.Assert(it.Status, model.StatusPublished)
		}

		// ListManage (author-scoped) includes the draft
		mine, mineTotal, err := d.ListManage(ctx, "author-1", "", "", nil, false, false, 20, 0)
		t.AssertNil(err)
		t.Assert(mineTotal >= 2, true)
		t.Assert(len(mine) >= 2, true)

		// SoftDelete → GetByID returns (nil, nil)
		n, err := d.SoftDelete(ctx, "author-1", post.ID)
		t.AssertNil(err)
		t.Assert(n, int64(1))
		gone, err := d.GetByID(ctx, post.ID)
		t.AssertNil(err)
		t.AssertNil(gone)

		_, _ = db.Exec(ctx, "TRUNCATE posts CASCADE")
	})
}

// resetSchema drops the full migration chain (FK-safe order 0003→0002→0001 down)
// then re-applies 0001, leaving a clean posts schema. A bare 0001 down can't drop
// posts once comments/subscribers FK it, so this keeps the integration tests
// re-runnable against a fully-migrated blog DB (see gotchas §5).
func resetSchema(t *gtest.T, sdb *sql.DB) {
	const dir = "../../manifest/sql/migrations/"
	for _, f := range []string{"0003_subscribers.down.sql", "0002_comments.down.sql", "0001_init.down.sql"} {
		b, err := os.ReadFile(dir + f)
		t.AssertNil(err)
		_, _ = sdb.Exec(string(b))
	}
	up, err := os.ReadFile(dir + "0001_init.up.sql")
	t.AssertNil(err)
	_, err = sdb.Exec(string(up))
	t.AssertNil(err)
}

// TestPGSearch exercises the zhparser + tsvector + GIN full-text path (migration
// 0004): word-level matching (not ILIKE substring) and title>content ts_rank order.
//
//	BLOG_PG_HOST=192.168.5.5 BLOG_PG_USER=postgres BLOG_PG_PASS=postgres \
//	  go test -run TestPGSearch ./products/blog/api/internal/dao/...
func TestPGSearch(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		host := os.Getenv("BLOG_PG_HOST")
		if host == "" {
			t.Skip("set BLOG_PG_HOST to run the blog search integration test")
		}
		port, user, pass := envOr("BLOG_PG_PORT", "5432"), envOr("BLOG_PG_USER", "postgres"), os.Getenv("BLOG_PG_PASS")
		ctx := context.Background()

		sdb, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=blog sslmode=disable", host, port, user, pass))
		t.AssertNil(err)
		resetSchema(t, sdb)
		up4, err := os.ReadFile("../../manifest/sql/migrations/0004_search_tsvector.up.sql")
		t.AssertNil(err)
		_, err = sdb.Exec(string(up4))
		t.AssertNil(err)
		sdb.Close()

		db, err := gdb.New(gdb.ConfigNode{Type: "pgsql", Host: host, Port: port, User: user, Pass: pass, Name: "blog"})
		t.AssertNil(err)
		d := dao.NewPG(db)

		// "全文检索" appears in titleHit's title (weight A) and contentHit's body
		// (weight C); noHit has neither. zhparser tokenizes 数据库/全文检索 as whole words.
		titleHit := &model.Post{AuthorID: "a", Title: "PostgreSQL 全文检索实战", Slug: "ft-title",
			Content: "聊聊 tsvector 与 GIN 倒排索引", Status: model.StatusPublished, PublishedAt: gtime.Now()}
		contentHit := &model.Post{AuthorID: "a", Title: "数据库性能优化", Slug: "ft-content",
			Content: "全文检索只是其中一环，还要考虑缓存与连接池", Status: model.StatusPublished, PublishedAt: gtime.Now()}
		noHit := &model.Post{AuthorID: "a", Title: "前端工程化", Slug: "ft-none",
			Content: "Vue 与 Nuxt 的构建流水线", Status: model.StatusPublished, PublishedAt: gtime.Now()}
		for _, m := range []*model.Post{titleHit, contentHit, noHit} {
			t.AssertNil(d.Insert(ctx, m))
		}

		items, total, err := d.List(ctx, dao.ListFilter{Q: "全文检索"}, 20, 0)
		t.AssertNil(err)
		t.Assert(total, 2) // noHit excluded
		t.Assert(len(items), 2)
		t.Assert(items[0].Slug, "ft-title") // title (weight A) outranks content (weight C)
		t.Assert(items[1].Slug, "ft-content")

		// Word-level, not substring: "据库" is a non-word fragment of 数据库 — ILIKE
		// '%据库%' would have matched; tsquery must not.
		_, subTotal, err := d.List(ctx, dao.ListFilter{Q: "据库"}, 20, 0)
		t.AssertNil(err)
		t.Assert(subTotal, 0)

		_, _ = db.Exec(ctx, "TRUNCATE posts CASCADE")
	})
}
