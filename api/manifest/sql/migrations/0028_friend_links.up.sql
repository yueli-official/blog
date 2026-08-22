ALTER TABLE home_config
    ADD COLUMN friend_links JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT ck_home_config_friend_links_array
        CHECK (jsonb_typeof(friend_links) = 'array');
