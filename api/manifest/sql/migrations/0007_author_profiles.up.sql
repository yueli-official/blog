-- M5: author profiles. A blog-side overlay keyed by the IdP identity (the
-- author_id carried on posts). display_name/avatar_url default-empty and the UI
-- falls back to the IdP claim; bio/banner/social/role are blog-owned. social_links
-- is a JSON array of {label,url}. role distinguishes 主笔 (lead) from 客座 (guest).
CREATE TABLE author_profiles (
    author_id    TEXT PRIMARY KEY,
    display_name TEXT NOT NULL DEFAULT '',
    bio          TEXT NOT NULL DEFAULT '',
    avatar_url   TEXT NOT NULL DEFAULT '',
    banner_url   TEXT NOT NULL DEFAULT '',
    social_links JSONB NOT NULL DEFAULT '[]',
    role         TEXT NOT NULL DEFAULT 'lead',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
