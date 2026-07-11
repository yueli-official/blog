ALTER TABLE author_profiles
    ADD COLUMN display_name TEXT,
    ADD COLUMN bio          TEXT,
    ADD COLUMN avatar_url   TEXT,
    ADD COLUMN banner_url   TEXT,
    ADD COLUMN social_links TEXT;
