package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"platform/gokit/classification"
	"platform/products/blog/api/internal/blogclient"
	"platform/products/blog/api/internal/dao"
	"platform/products/blog/api/internal/model"
)

func TestPostgreSQLBlogClassificationConsumer(t *testing.T) {
	host := strings.TrimSpace(os.Getenv("BLOG_PG_HOST"))
	if host == "" {
		t.Skip("set BLOG_PG_HOST to run the Blog content-PostgreSQL classification integration test")
	}
	port := blogIntegrationEnvironment("BLOG_PG_PORT", "5432")
	user := blogIntegrationEnvironment("BLOG_PG_USER", "postgres")
	password := os.Getenv("BLOG_PG_PASS")
	admin, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable", host, port, user, password))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close() })
	var serverVersion int
	if err := admin.QueryRow(`SELECT current_setting('server_version_num')::int`).Scan(&serverVersion); err != nil {
		t.Fatal(err)
	}
	if serverVersion < 160000 {
		t.Fatalf("PostgreSQL %d is too old; Blog content storage requires 16 or newer", serverVersion)
	}

	database := fmt.Sprintf("blog_classification_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE DATABASE ` + pq.QuoteIdentifier(database)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, database)
		_, _ = admin.Exec(`DROP DATABASE IF EXISTS ` + pq.QuoteIdentifier(database))
	})
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, database)
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	migration, err := os.ReadFile("../../manifest/sql/migrations/0001_init.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(string(migration)); err != nil {
		t.Fatalf("apply Blog migration: %v", err)
	}
	if _, err := sqlDB.Exec(`
INSERT INTO blog_classification_catalogs (id, catalog_key, revision)
VALUES ('01990000-0000-7000-8000-000000000001', 'blog', 1);
INSERT INTO blog_classification_policy_profiles
    (catalog_id, policy_key, schema_version, policy_revision, category_policy, facet_policies, tag_policy, discovery_policy)
VALUES
    ('01990000-0000-7000-8000-000000000001', 'blog.post.default', 1, 1,
     '{"minAssignments":0,"maxAssignments":0,"requirePrimary":false,"leafOnly":false,"maxDepth":0}',
     '[]',
     '{"unknown":"create","minAssignments":0,"maxAssignments":0}',
     '{"defaultSort":"name_asc"}');`); err != nil {
		t.Fatal(err)
	}
	databaseHandle, err := gdb.New(gdb.ConfigNode{Type: "pgsql", Host: host, Port: port, User: user, Pass: password, Name: database})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = databaseHandle.Close(context.Background()) })
	store := dao.NewPG(databaseHandle)
	service := New(store, blogclient.NewFake(), "blog-cover", nil, "http://blog.test", SpamPolicy{})

	root, err := service.CreateTaxonomy(context.Background(), "技术", "category", "tech", "", "")
	if err != nil {
		t.Fatal(err)
	}
	child, err := service.CreateTaxonomy(context.Background(), "后端", "category", "backend", root.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	tag, err := service.CreateTaxonomy(context.Background(), "Go", "tag", "go", "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{root.ID, child.ID, tag.ID} {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed.Version() != 7 {
			t.Fatalf("classification ID %q is not UUIDv7: version=%v err=%v", id, parsed.Version(), err)
		}
	}
	if _, err := service.CreateTaxonomy(context.Background(), "错误层级", "tag", "nested-tag", root.ID, ""); err == nil {
		t.Fatal("Tag creation with parentId should be rejected")
	}

	post := &model.Post{AuthorID: "author-1", Title: "共享分类", Slug: "shared-classification", Content: "body", Status: model.StatusPublished}
	if err := store.Insert(context.Background(), post); err != nil {
		t.Fatal(err)
	}
	if err := service.AssignTaxonomies(context.Background(), post.AuthorID, post.ID, []string{root.ID, child.ID, tag.ID, child.ID}); err != nil {
		t.Fatal(err)
	}
	var categoryCount, tagCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM blog_post_category_assignments WHERE post_id = $1`, post.ID).Scan(&categoryCount); err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM blog_post_tag_assignments WHERE post_id = $1`, post.ID).Scan(&tagCount); err != nil {
		t.Fatal(err)
	}
	if categoryCount != 2 || tagCount != 1 {
		t.Fatalf("assignment counts = categories %d, tags %d; want 2 and 1", categoryCount, tagCount)
	}
	categories, err := service.ListTaxonomies(context.Background(), "category")
	if err != nil {
		t.Fatal(err)
	}
	counts := make(map[string]int, len(categories))
	for _, value := range categories {
		counts[value.Slug] = value.PostCount
	}
	if counts["tech"] != 1 || counts["backend"] != 1 {
		t.Fatalf("hierarchical distinct counts = %#v; want tech=1 and backend=1", counts)
	}
	if err := service.MergeTaxonomy(context.Background(), child.ID, tag.ID); err == nil {
		t.Fatal("Category and Tag merge should be rejected")
	}
	snapshot, err := store.ClassificationSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	compiled := classification.Compile(snapshot)
	if compiled.Outcome != classification.OutcomeAccepted || compiled.Catalog == nil {
		t.Fatalf("Blog snapshot did not compile: %#v", compiled.Diagnostics)
	}
}

func blogIntegrationEnvironment(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
