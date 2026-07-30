package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

const initialHomeConfig = `
INSERT INTO home_config (
	key, eyebrow, title, subtitle,
	site_title, site_description, support_email,
	footer_tagline, footer_copyright
)
VALUES (
	'default', '月离博客', '把想法写成可以长期沉淀的内容',
	'文章、系列与讨论，在清晰的阅读体验中持续积累。',
	$1, $2, '',
	'记录值得反复阅读的想法。', ''
)
ON CONFLICT (key) DO NOTHING`

func main() {
	databaseURL := strings.TrimSpace(os.Getenv("BLOG_DATABASE_URL"))
	if databaseURL == "" {
		fail("BLOG_DATABASE_URL is required")
	}
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		fail("open database: %v", err)
	}
	defer database.Close()
	ctx := context.Background()
	if err := database.PingContext(ctx); err != nil {
		fail("connect database: %v", err)
	}
	brand := strings.TrimSpace(os.Getenv("BLOG_SITE_BRAND"))
	if brand == "" {
		brand = "月离博客"
	}
	description := strings.TrimSpace(os.Getenv("BLOG_SITE_DESCRIPTION"))
	if description == "" {
		description = "想法、笔记与记录"
	}
	if _, err := database.ExecContext(ctx, initialHomeConfig, brand, description); err != nil {
		fail("install initial Blog configuration: %v", err)
	}
	fmt.Println("Blog initial records are ready")
}

func fail(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, "bootstrap: "+format+"\n", arguments...)
	os.Exit(1)
}
