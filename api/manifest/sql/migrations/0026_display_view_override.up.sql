ALTER TABLE post_stats
    ADD COLUMN display_view_override BIGINT,
    ADD CONSTRAINT ck_post_stats_display_view_override
        CHECK (display_view_override IS NULL OR display_view_override >= 0);
