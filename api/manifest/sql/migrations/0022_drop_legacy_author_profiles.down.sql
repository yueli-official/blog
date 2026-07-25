CREATE TABLE author_profiles (
    author_id  TEXT PRIMARY KEY,
    role       TEXT NOT NULL DEFAULT 'contributor',
    status     TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
