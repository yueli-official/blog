package main

import (
	"context"
	"database/sql"
	"os"
	"testing"
)

func TestInitialRecordsSupportEmptyProductionDatabaseWithoutDemoContent(t *testing.T) {
	databaseURL := os.Getenv("BLOG_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set BLOG_TEST_DATABASE_URL to run the Blog bootstrap integration test")
	}
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	tx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for range 2 {
		if err := installInitialRecords(context.Background(), tx, "验收博客", "空库初始化"); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM blog_classification_catalogs c
		JOIN blog_classification_policy_profiles p ON p.catalog_id = c.id
		WHERE c.catalog_key = 'blog' AND p.policy_key = 'blog.post.default'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("initial catalog/policy count = %d, want 1", count)
	}
	if err := tx.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("production bootstrap unexpectedly contains %d posts", count)
	}
}

func TestInstallLocalAcceptanceContentIsIdempotent(t *testing.T) {
	databaseURL := os.Getenv("BLOG_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set BLOG_TEST_DATABASE_URL to run the Blog bootstrap integration test")
	}
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	tx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	if err := installInitialRecords(context.Background(), tx, "验收博客", "本地验收"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := installLocalAcceptanceContent(context.Background(), tx, "TestA123"); err != nil {
			t.Fatal(err)
		}
	}

	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM posts WHERE id IN (
		'019c52f0-1000-7000-8000-000000000401',
		'019c52f0-1000-7000-8000-000000000402',
		'019c52f0-1000-7000-8000-000000000403',
		'019c52f0-1000-7000-8000-000000000404',
		'019c52f0-1000-7000-8000-000000000405',
		'019c52f0-1000-7000-8000-000000000406',
		'019c52f0-1000-7000-8000-000000000407',
		'019c52f0-1000-7000-8000-000000000408'
	)`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 8 {
		t.Fatalf("seeded post count = %d, want 8", count)
	}
	if err := tx.QueryRow(`SELECT COUNT(*)
		FROM blog_tag_lookup_entries entry
		JOIN blog_tags tag ON tag.id = entry.target_tag_id
		WHERE entry.kind = 'canonical'
		  AND entry.lookup_key = tag.current_name
		  AND tag.current_slug IN ('writing', 'product', 'engineering', 'life')`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("seeded canonical tag lookup count = %d, want 4", count)
	}
	if err := tx.QueryRow(`SELECT COUNT(*) FROM subscribers WHERE id::text LIKE '019c52f0-1000-7000-8000-00000000050_'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deprecated subscriber sample count = %d, want 0", count)
	}
	if err := tx.QueryRow(`SELECT COUNT(*) FROM blog_traffic_source_daily WHERE day >= CURRENT_DATE - INTERVAL '13 days'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 14 {
		t.Fatalf("seeded traffic-source row count = %d, want 14", count)
	}
}
