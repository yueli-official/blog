ALTER TABLE home_config
    DROP CONSTRAINT IF EXISTS ck_home_config_contact_links_array,
    DROP COLUMN IF EXISTS contact_links;
