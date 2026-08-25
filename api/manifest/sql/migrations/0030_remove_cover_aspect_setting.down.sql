ALTER TABLE home_config
    ADD COLUMN cover_aspect_width SMALLINT NOT NULL DEFAULT 3,
    ADD COLUMN cover_aspect_height SMALLINT NOT NULL DEFAULT 2,
    ADD CONSTRAINT ck_home_config_cover_aspect
        CHECK (
            cover_aspect_width BETWEEN 1 AND 100
            AND cover_aspect_height BETWEEN 1 AND 100
        );
