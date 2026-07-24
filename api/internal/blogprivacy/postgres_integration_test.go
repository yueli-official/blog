package blogprivacy

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/yueli-official/foundation/go/privacy"
)

func TestPostgresNewsletterStateAndEvidenceAreAtomic(t *testing.T) {
	base := strings.TrimSpace(os.Getenv("PRIVACY_CONSUMER_PG_DSN"))
	if base == "" {
		t.Skip("PRIVACY_CONSUMER_PG_DSN is not set")
	}
	ctx := context.Background()
	admin, err := sql.Open("postgres", base)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "blog_privacy_" + strings.ReplaceAll(time.Now().UTC().Format("150405.000000"), ".", "")
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = admin.ExecContext(ctx, `DROP SCHEMA IF EXISTS `+schema+` CASCADE`) }()
	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := sql.Open("postgres", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migration, err := privacy.Schema(privacy.CurrentSchemaVersion)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, migration.UpSQL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
CREATE TABLE subscribers(
  id uuid PRIMARY KEY, email text NOT NULL UNIQUE, status text NOT NULL,
  confirm_token text NOT NULL, created_at timestamptz NOT NULL,
  confirmed_at timestamptz, unsubscribed_at timestamptz
);
CREATE TABLE comments(id uuid PRIMARY KEY, user_id text NOT NULL, author_name text NOT NULL,
  author_email text NOT NULL, ip text NOT NULL, user_agent text NOT NULL, deleted_at timestamptz);
CREATE TABLE posts(id uuid PRIMARY KEY, author_id text NOT NULL);
CREATE TABLE post_revisions(id uuid PRIMARY KEY, author_id text NOT NULL);
CREATE TABLE author_profiles(author_id text PRIMARY KEY);
CREATE TABLE post_likes(user_id text NOT NULL);
CREATE TABLE post_bookmarks(user_id text NOT NULL);
INSERT INTO subscribers(id,email,status,confirm_token,created_at)
VALUES ('00000000-0000-0000-0000-000000000001','reader@example.com','pending','confirm-1',now());
`); err != nil {
		t.Fatal(err)
	}
	service, err := NewPostgres(ctx, db, "blog:test")
	if err != nil {
		t.Fatal(err)
	}
	email, ok, err := service.ConfirmSubscription(ctx, "confirm-1")
	if err != nil || !ok || email != "reader@example.com" {
		t.Fatalf("confirm = %q, %v, %v", email, ok, err)
	}
	// HTTP retry replays both the product update and immutable receipt.
	if _, ok, err = service.ConfirmSubscription(ctx, "confirm-1"); err != nil || !ok {
		t.Fatalf("confirm replay = %v, %v", ok, err)
	}
	allowed, err := service.CanDeliverNewsletter(ctx, email)
	if err != nil || !allowed {
		t.Fatalf("delivery decision = %v, %v", allowed, err)
	}
	if ok, err = service.Unsubscribe(ctx, "confirm-1"); err != nil || !ok {
		t.Fatalf("unsubscribe = %v, %v", ok, err)
	}
	if ok, err = service.Unsubscribe(ctx, "confirm-1"); err != nil || !ok {
		t.Fatalf("unsubscribe replay = %v, %v", ok, err)
	}
	allowed, err = service.CanDeliverNewsletter(ctx, email)
	if err != nil || allowed {
		t.Fatalf("post-withdrawal decision = %v, %v", allowed, err)
	}
}
