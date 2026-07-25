-- Authorization is the sole source of Blog role, grant, and application state.
-- Public author display data comes from Identity and content ownership remains
-- on posts/series/comments, so this legacy authorization overlay is redundant.
DROP TABLE IF EXISTS author_profiles;
