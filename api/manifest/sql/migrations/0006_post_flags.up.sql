-- M4: editorial flags. pinned = stuck to the top of listings; featured = shown
-- in the home carousel. Both admin-managed; ordering uses publish time.
ALTER TABLE posts
    ADD COLUMN pinned   BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN featured BOOLEAN NOT NULL DEFAULT false;
CREATE INDEX ix_posts_pinned ON posts (pinned) WHERE pinned;
CREATE INDEX ix_posts_featured ON posts (featured) WHERE featured;
