CREATE TABLE home_config (
    key        TEXT PRIMARY KEY DEFAULT 'default',
    eyebrow    TEXT NOT NULL,
    title      TEXT NOT NULL,
    subtitle   TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (key = 'default')
);
