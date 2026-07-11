CREATE TABLE home_config (
    key        TEXT PRIMARY KEY DEFAULT 'default',
    eyebrow    TEXT NOT NULL DEFAULT 'Editorial',
    title      TEXT NOT NULL DEFAULT '博客',
    subtitle   TEXT NOT NULL DEFAULT '想法、笔记与记录, 关于技术、产品与日常的长短文。',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (key = 'default')
);
