CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE posts (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id      TEXT NOT NULL,
    post_type      TEXT NOT NULL DEFAULT 'post',
    title          TEXT NOT NULL,
    slug           TEXT NOT NULL,
    content        TEXT NOT NULL DEFAULT '',
    excerpt        TEXT NOT NULL DEFAULT '',
    cover_asset_id TEXT NOT NULL DEFAULT '',
    cover_url      TEXT NOT NULL DEFAULT '',
    comment_status SMALLINT NOT NULL DEFAULT 1,
    status         TEXT NOT NULL DEFAULT 'draft',
    locale         TEXT NOT NULL DEFAULT 'zh-CN',
    published_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);
CREATE UNIQUE INDEX uq_posts_slug       ON posts (slug) WHERE deleted_at IS NULL;
CREATE INDEX        ix_posts_status_time ON posts (status, published_at DESC);
CREATE INDEX        ix_posts_author      ON posts (author_id);

CREATE TABLE post_stats (
    post_id       UUID PRIMARY KEY REFERENCES posts(id) ON DELETE CASCADE,
    view_count    BIGINT NOT NULL DEFAULT 0,
    like_count    BIGINT NOT NULL DEFAULT 0,
    comment_count BIGINT NOT NULL DEFAULT 0,
    share_count   BIGINT NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE post_seo (
    post_id         UUID PRIMARY KEY REFERENCES posts(id) ON DELETE CASCADE,
    meta_title      TEXT NOT NULL DEFAULT '',
    meta_desc       TEXT NOT NULL DEFAULT '',
    og_title        TEXT NOT NULL DEFAULT '',
    og_image        TEXT NOT NULL DEFAULT '',
    canonical_url   TEXT NOT NULL DEFAULT '',
    robots          TEXT NOT NULL DEFAULT '',
    structured_data JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE post_metas (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    meta_key   TEXT NOT NULL,
    meta_value TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX uq_post_metas_key ON post_metas (post_id, meta_key);

CREATE TABLE post_revisions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id  TEXT NOT NULL,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL,
    rev_note   TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_post_revisions_post ON post_revisions (post_id, created_at DESC);

CREATE TABLE terms (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL
);
CREATE UNIQUE INDEX uq_terms_slug ON terms (slug);

CREATE TABLE taxonomies (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    term_id     UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    taxonomy    TEXT NOT NULL,                                  -- 'category' | 'tag'
    description TEXT NOT NULL DEFAULT '',
    parent_id   UUID REFERENCES taxonomies(id) ON DELETE SET NULL,
    post_count  BIGINT NOT NULL DEFAULT 0,
    extra       JSONB NOT NULL DEFAULT '{}'
);
CREATE UNIQUE INDEX uq_taxonomies_term_taxonomy ON taxonomies (term_id, taxonomy);
CREATE INDEX        ix_taxonomies_taxonomy      ON taxonomies (taxonomy);

CREATE TABLE object_taxonomies (
    object_id   UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    taxonomy_id UUID NOT NULL REFERENCES taxonomies(id) ON DELETE CASCADE,
    sort_order  INT NOT NULL DEFAULT 0,
    PRIMARY KEY (object_id, taxonomy_id)
);
CREATE INDEX ix_object_taxonomies_tax ON object_taxonomies (taxonomy_id);

CREATE TABLE post_likes (
    user_id    TEXT NOT NULL,
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, post_id)
);

CREATE TABLE post_bookmarks (
    user_id    TEXT NOT NULL,
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, post_id)
);
