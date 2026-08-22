ALTER TABLE home_config
    DROP CONSTRAINT IF EXISTS ck_home_config_friend_links_array,
    DROP COLUMN IF EXISTS friend_links;
