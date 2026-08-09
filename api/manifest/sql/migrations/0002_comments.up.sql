CREATE TABLE comments (
    id           UUID PRIMARY KEY,
    post_id      UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    parent_id    UUID REFERENCES comments(id) ON DELETE CASCADE,
    user_id      TEXT NOT NULL DEFAULT '',          -- JWT sub; '' = anonymous
    author_name  TEXT NOT NULL DEFAULT '',          -- anonymous display name
    author_email TEXT NOT NULL DEFAULT '',          -- anonymous contact (optional)
    content      TEXT NOT NULL,
    status       SMALLINT NOT NULL DEFAULT 2,        -- 1 approved / 2 pending / 3 spam / 4 trash
    ip           TEXT NOT NULL DEFAULT '',
    user_agent   TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);
CREATE INDEX ix_comments_post_status_time ON comments (post_id, status, created_at DESC);
CREATE INDEX ix_comments_parent           ON comments (parent_id);
