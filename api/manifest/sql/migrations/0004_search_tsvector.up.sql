-- Full-text search: zhparser (Chinese segmentation) + tsvector generated column + GIN inverted index.
-- Platform search standard, see flightdeck docs/backend-platform-conventions §搜索 (locked 2026-06-21).
--
-- Server-level ops dependency: the zhparser shared library + SCWS dictionary must already be
-- installed on the PG host (dev 192.168.5.5 uses the ghcr.io/mnixry/postgres-zhparser image; prod
-- must provision the same). This migration is self-contained: it enables the extension and creates
-- the `chinese_zh` text search configuration per-database, so a fresh `blog` DB needs nothing else.

CREATE EXTENSION IF NOT EXISTS zhparser;

-- CREATE TEXT SEARCH CONFIGURATION has no IF NOT EXISTS, so guard it. Mapping mirrors the validated
-- dev config (POS tokens n,v,a,i,e,l → simple): nouns / verbs / adjectives / idioms / exclamations /
-- temporary-idioms; punctuation & whitespace are intentionally left unindexed.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_ts_config WHERE cfgname = 'chinese_zh') THEN
        CREATE TEXT SEARCH CONFIGURATION chinese_zh (PARSER = zhparser);
        ALTER TEXT SEARCH CONFIGURATION chinese_zh ADD MAPPING FOR n,v,a,i,e,l WITH simple;
    END IF;
END$$;

-- Generated tsvector over title (weight A) + excerpt (B) + content (C) so ts_rank favors title hits.
ALTER TABLE posts ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('chinese_zh', coalesce(title,   '')), 'A') ||
        setweight(to_tsvector('chinese_zh', coalesce(excerpt, '')), 'B') ||
        setweight(to_tsvector('chinese_zh', coalesce(content, '')), 'C')
    ) STORED;

CREATE INDEX ix_posts_search ON posts USING GIN (search_vector);
