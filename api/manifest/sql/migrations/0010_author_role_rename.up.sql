-- Blog-local RBAC: rename authorship roles to the owner/author/contributor model.
-- lead → author (主笔, team writer), guest → contributor (客座, submitter). The site
-- OWNER is no longer a stored role: it is the operator listed in blog.operatorSubs.
-- (config), so blog authorization is decoupled from the IdP's shared global role.
UPDATE author_profiles SET role = 'author' WHERE role = 'lead';
UPDATE author_profiles SET role = 'contributor' WHERE role = 'guest';
ALTER TABLE author_profiles ALTER COLUMN role SET DEFAULT 'contributor';
