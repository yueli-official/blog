ALTER TABLE home_config
    ADD COLUMN contact_links JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT ck_home_config_contact_links_array
        CHECK (jsonb_typeof(contact_links) = 'array');
