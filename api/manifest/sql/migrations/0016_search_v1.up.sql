CREATE TABLE search_instances (
    instance_key TEXT PRIMARY KEY,
    schema_version INTEGER NOT NULL,
    consumer TEXT NOT NULL,
    definition_version BIGINT NOT NULL,
    definition_digest TEXT NOT NULL,
    active_generation TEXT NOT NULL,
    building_generation TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE search_generations (
    instance_key TEXT NOT NULL REFERENCES search_instances(instance_key) ON DELETE CASCADE,
    id TEXT NOT NULL,
    request_id TEXT NOT NULL,
    phase TEXT NOT NULL,
    definition_digest TEXT NOT NULL,
    checkpoint TEXT NOT NULL DEFAULT '',
    document_count BIGINT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (instance_key, id),
    UNIQUE (instance_key, request_id)
);

CREATE TABLE search_documents (
    instance_key TEXT NOT NULL,
    generation_id TEXT NOT NULL,
    document_kind TEXT NOT NULL,
    document_id TEXT NOT NULL,
    revision BIGINT NOT NULL,
    content_digest TEXT NOT NULL,
    analyzer TEXT NOT NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    body TEXT NOT NULL,
    keywords TEXT[] NOT NULL,
    filters JSONB NOT NULL,
    sort_at TIMESTAMPTZ NOT NULL,
    visibility_type TEXT NOT NULL,
    visibility_id TEXT NOT NULL,
    deleted BOOLEAN NOT NULL,
    search_vector TSVECTOR NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (instance_key, generation_id, document_kind, document_id),
    FOREIGN KEY (instance_key, generation_id)
        REFERENCES search_generations(instance_key, id) ON DELETE CASCADE
);
CREATE INDEX search_documents_full_text
    ON search_documents USING GIN (search_vector);
CREATE INDEX search_documents_query
    ON search_documents (instance_key, generation_id, analyzer, deleted, sort_at DESC, document_kind, document_id);
CREATE INDEX search_documents_filters
    ON search_documents USING GIN (filters);

CREATE TABLE search_batch_receipts (
    instance_key TEXT NOT NULL REFERENCES search_instances(instance_key) ON DELETE CASCADE,
    batch_id TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    applied INTEGER NOT NULL,
    replays INTEGER NOT NULL,
    stale INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (instance_key, batch_id)
);

ALTER TABLE posts ADD COLUMN search_revision BIGINT NOT NULL DEFAULT 1;
DROP INDEX IF EXISTS ix_posts_search;
ALTER TABLE posts DROP COLUMN IF EXISTS search_vector;

CREATE FUNCTION blog_bump_search_revision() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF (NEW.title, NEW.excerpt, NEW.content, NEW.status, NEW.locale, NEW.published_at, NEW.deleted_at)
       IS DISTINCT FROM
       (OLD.title, OLD.excerpt, OLD.content, OLD.status, OLD.locale, OLD.published_at, OLD.deleted_at) THEN
        NEW.search_revision := OLD.search_revision + 1;
    END IF;
    RETURN NEW;
END$$;
CREATE TRIGGER blog_posts_search_revision
BEFORE UPDATE ON posts FOR EACH ROW EXECUTE FUNCTION blog_bump_search_revision();
