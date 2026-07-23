DROP TABLE IF EXISTS search_batch_receipts;
DROP TABLE IF EXISTS search_documents;
DROP TABLE IF EXISTS search_generations;
DROP TABLE IF EXISTS search_instances;

DROP TRIGGER IF EXISTS blog_posts_search_revision ON posts;
DROP FUNCTION IF EXISTS blog_bump_search_revision();
ALTER TABLE posts DROP COLUMN IF EXISTS search_revision;
ALTER TABLE posts ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('chinese_zh', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('chinese_zh', coalesce(excerpt, '')), 'B') ||
    setweight(to_tsvector('chinese_zh', coalesce(content, '')), 'C')
) STORED;
CREATE INDEX ix_posts_search ON posts USING GIN (search_vector);
