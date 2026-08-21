ALTER TABLE post_stats
    DROP CONSTRAINT IF EXISTS ck_post_stats_display_view_override,
    DROP COLUMN IF EXISTS display_view_override;
