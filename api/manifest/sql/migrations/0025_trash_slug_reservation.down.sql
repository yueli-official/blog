DROP INDEX IF EXISTS uq_posts_slug;
CREATE UNIQUE INDEX uq_posts_slug ON posts (slug) WHERE deleted_at IS NULL;
