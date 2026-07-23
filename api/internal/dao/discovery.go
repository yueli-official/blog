package dao

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/os/gtime"
)

type DiscoveryRow struct {
	Key          string      `orm:"key"`
	Path         string      `orm:"path"`
	Kind         string      `orm:"kind"`
	Title        string      `orm:"title"`
	Description  string      `orm:"description"`
	ImageURL     string      `orm:"image_url"`
	Locale       string      `orm:"locale"`
	AuthorID     string      `orm:"author_id"`
	Content      string      `orm:"content"`
	CanonicalURL string      `orm:"canonical_url"`
	Robots       string      `orm:"robots"`
	PublishedAt  *gtime.Time `orm:"published_at"`
	UpdatedAt    *gtime.Time `orm:"updated_at"`
}

type URLLifecycleClaim struct {
	ID   string `orm:"id"`
	Kind string `orm:"kind"`
	Slug string `orm:"slug"`
}

// ListURLLifecycleClaims returns the product-authoritative public identities
// used to seed or repair the local URL registry at startup.
func (p *PG) ListURLLifecycleClaims(ctx context.Context) ([]URLLifecycleClaim, error) {
	var rows []URLLifecycleClaim
	err := p.db.Ctx(ctx).Raw(`
SELECT post.id::text AS id, 'post'::text AS kind, post.slug
FROM posts post
WHERE post.published_at IS NOT NULL AND post.deleted_at IS NULL
UNION ALL
SELECT category.id::text, 'category'::text, category.current_slug
FROM blog_categories category
WHERE category.status <> 'replaced'
UNION ALL
SELECT tag.id::text, 'tag'::text, tag.current_slug
FROM blog_tags tag
WHERE tag.status <> 'replaced'
ORDER BY kind, id`).Scan(&rows)
	if rows == nil {
		rows = []URLLifecycleClaim{}
	}
	return rows, err
}

func (p *PG) ListDiscoveryPages(ctx context.Context, origin, afterURL string, limit int) ([]DiscoveryRow, error) {
	var rows []DiscoveryRow
	err := p.db.Ctx(ctx).Raw(`
WITH pages AS (
    SELECT 'post:' || post.id::text AS key,
           '/posts/' || post.slug AS path,
           'article'::text AS kind,
           COALESCE(NULLIF(seo.meta_title, ''), post.title) AS title,
           COALESCE(NULLIF(seo.meta_desc, ''), NULLIF(post.excerpt, ''), LEFT(post.content, 300)) AS description,
           COALESCE(NULLIF(seo.og_image, ''), post.cover_url, '') AS image_url,
           COALESCE(NULLIF(post.locale, ''), 'zh-CN') AS locale,
           post.author_id::text AS author_id,
           post.content,
           COALESCE(seo.canonical_url, '') AS canonical_url,
           COALESCE(seo.robots, '') AS robots,
           post.published_at,
           post.updated_at
    FROM posts post
    LEFT JOIN post_seo seo ON seo.post_id = post.id
    WHERE post.status = 'published' AND post.deleted_at IS NULL
    UNION ALL
    SELECT 'category:' || category.id::text,
           '/category/' || category.current_slug,
           'collection',
           category.current_name,
           category.description,
           '',
           'zh-CN',
           '',
           '',
           '',
           '',
           category.first_activated_at,
           category.updated_at
    FROM blog_categories category
    WHERE category.status <> 'replaced'
    UNION ALL
    SELECT 'tag:' || tag.id::text,
           '/tags/' || tag.current_slug,
           'collection',
           tag.current_name,
           tag.description,
           '',
           'zh-CN',
           '',
           '',
           '',
           '',
           tag.first_activated_at,
           tag.updated_at
    FROM blog_tags tag
    WHERE tag.status <> 'replaced'
)
SELECT * FROM pages
WHERE COALESCE(NULLIF(canonical_url, ''), ? || path) > ?
ORDER BY COALESCE(NULLIF(canonical_url, ''), ? || path) ASC
LIMIT ?`, origin, afterURL, origin, limit).Scan(&rows)
	return rows, err
}

func (p *PG) ListDiscoveryFeedPosts(
	ctx context.Context,
	origin string,
	afterURL string,
	taxonomyKind string,
	taxonomySlug string,
	limit int,
) ([]DiscoveryRow, error) {
	var rows []DiscoveryRow
	err := p.db.Ctx(ctx).Raw(`
SELECT 'post:' || post.id::text AS key,
       '/posts/' || post.slug AS path,
       'article'::text AS kind,
       COALESCE(NULLIF(seo.meta_title, ''), post.title) AS title,
       COALESCE(NULLIF(seo.meta_desc, ''), NULLIF(post.excerpt, ''), LEFT(post.content, 300)) AS description,
       COALESCE(NULLIF(seo.og_image, ''), post.cover_url, '') AS image_url,
       COALESCE(NULLIF(post.locale, ''), 'zh-CN') AS locale,
       post.author_id::text AS author_id,
       post.content,
       COALESCE(seo.canonical_url, '') AS canonical_url,
       COALESCE(seo.robots, '') AS robots,
       post.published_at,
       post.updated_at
FROM posts post
LEFT JOIN post_seo seo ON seo.post_id = post.id
WHERE post.status = 'published'
  AND post.deleted_at IS NULL
  AND COALESCE(NULLIF(seo.canonical_url, ''), ? || '/posts/' || post.slug) > ?
  AND (
    ? = ''
    OR (? = 'category' AND EXISTS (
        SELECT 1 FROM blog_post_category_assignments assignment
        JOIN blog_categories category ON category.id = assignment.category_id
        WHERE assignment.post_id = post.id
          AND category.current_slug = ?
          AND category.status <> 'replaced'
    ))
    OR (? = 'tag' AND EXISTS (
        SELECT 1 FROM blog_post_tag_assignments assignment
        JOIN blog_tags tag ON tag.id = assignment.tag_id
        WHERE assignment.post_id = post.id
          AND tag.current_slug = ?
          AND tag.status <> 'replaced'
    ))
  )
ORDER BY COALESCE(NULLIF(seo.canonical_url, ''), ? || '/posts/' || post.slug) ASC
LIMIT ?`,
		origin, afterURL,
		taxonomyKind,
		taxonomyKind, taxonomySlug,
		taxonomyKind, taxonomySlug,
		origin,
		limit,
	).Scan(&rows)
	return rows, err
}

func (p *PG) DiscoveryUpdatedAt(ctx context.Context) (time.Time, error) {
	value, err := p.db.GetValue(ctx, `
SELECT COALESCE(MAX(updated_at), TIMESTAMPTZ '1970-01-01T00:00:00Z')
FROM posts
WHERE status = 'published' AND deleted_at IS NULL`)
	if err != nil {
		return time.Time{}, err
	}
	return value.Time().UTC(), nil
}
