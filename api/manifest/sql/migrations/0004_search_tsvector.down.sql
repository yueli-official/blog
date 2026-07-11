DROP INDEX IF EXISTS ix_posts_search;
ALTER TABLE posts DROP COLUMN IF EXISTS search_vector;
-- The chinese_zh text search configuration + zhparser extension are left in place: they are
-- DB-level shared infra (other tables/services may use them); removing is an ops decision.
