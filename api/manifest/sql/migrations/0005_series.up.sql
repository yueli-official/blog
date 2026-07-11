-- M3: post series (专题/连载). A post belongs to at most one series, ordered
-- within it by series_order. post_count is computed on read (published only).
CREATE TABLE series (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug           TEXT NOT NULL UNIQUE,
    name           TEXT NOT NULL,
    description    TEXT NOT NULL DEFAULT '',
    cover_asset_id TEXT NOT NULL DEFAULT '',
    cover_url      TEXT NOT NULL DEFAULT '',
    author_id      TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_series_author ON series (author_id);

ALTER TABLE posts
    ADD COLUMN series_id    UUID REFERENCES series(id) ON DELETE SET NULL,
    ADD COLUMN series_order INT NOT NULL DEFAULT 0;
CREATE INDEX ix_posts_series ON posts (series_id, series_order);
