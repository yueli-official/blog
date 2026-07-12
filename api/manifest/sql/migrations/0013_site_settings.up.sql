ALTER TABLE home_config
    ADD COLUMN site_title TEXT NOT NULL DEFAULT '',
    ADD COLUMN site_description TEXT NOT NULL DEFAULT '',
    ADD COLUMN support_email TEXT NOT NULL DEFAULT '',
    ADD COLUMN footer_tagline TEXT NOT NULL DEFAULT '',
    ADD COLUMN footer_copyright TEXT NOT NULL DEFAULT '';
