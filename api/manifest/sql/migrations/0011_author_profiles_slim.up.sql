-- Author display data (name / avatar / cover / bio / social) now lives in the
-- identity service (single source of truth); the blog keeps only domain-specific
-- author state. Drop the duplicated display columns from author_profiles.
ALTER TABLE author_profiles
    DROP COLUMN IF EXISTS display_name,
    DROP COLUMN IF EXISTS bio,
    DROP COLUMN IF EXISTS avatar_url,
    DROP COLUMN IF EXISTS banner_url,
    DROP COLUMN IF EXISTS social_links;
