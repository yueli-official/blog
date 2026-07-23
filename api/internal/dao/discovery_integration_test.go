package dao_test

import (
	"context"
	"os"
	"testing"

	"github.com/gogf/gf/v2/database/gdb"

	"platform/products/blog/api/internal/dao"
)

func TestPGDiscoveryQueries(t *testing.T) {
	host := os.Getenv("BLOG_PG_HOST")
	if host == "" {
		t.Skip("set BLOG_PG_HOST to run the blog discovery integration test")
	}
	db, err := gdb.New(gdb.ConfigNode{
		Type: "pgsql", Host: host, Port: envOr("BLOG_PG_PORT", "5432"),
		User: envOr("BLOG_PG_USER", "postgres"), Pass: os.Getenv("BLOG_PG_PASS"), Name: "blog",
	})
	if err != nil {
		t.Fatal(err)
	}
	store := dao.NewPG(db)
	if _, err := store.ListDiscoveryPages(context.Background(), "https://blog.example.com", "", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ListDiscoveryFeedPosts(context.Background(), "https://blog.example.com", "", "", "", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DiscoveryUpdatedAt(context.Background()); err != nil {
		t.Fatal(err)
	}
}
