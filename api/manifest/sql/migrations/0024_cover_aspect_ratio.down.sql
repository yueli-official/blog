ALTER TABLE home_config
    DROP CONSTRAINT IF EXISTS ck_home_config_cover_aspect,
    DROP COLUMN IF EXISTS cover_aspect_height,
    DROP COLUMN IF EXISTS cover_aspect_width;
