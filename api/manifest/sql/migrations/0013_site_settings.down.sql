ALTER TABLE home_config
    DROP COLUMN IF EXISTS footer_copyright,
    DROP COLUMN IF EXISTS footer_tagline,
    DROP COLUMN IF EXISTS support_email,
    DROP COLUMN IF EXISTS site_description,
    DROP COLUMN IF EXISTS site_title;
