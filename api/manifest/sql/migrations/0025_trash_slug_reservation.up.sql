DROP INDEX IF EXISTS uq_posts_slug;

WITH ranked_slugs AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY slug
            ORDER BY (deleted_at IS NULL) DESC, deleted_at DESC NULLS LAST, id
        ) AS slug_rank
    FROM posts
)
UPDATE posts AS post
SET slug = post.slug || '-trash-' || post.id::text
FROM ranked_slugs AS ranked
WHERE post.id = ranked.id
  AND ranked.slug_rank > 1
  AND post.deleted_at IS NOT NULL;

CREATE UNIQUE INDEX uq_posts_slug ON posts (slug);
