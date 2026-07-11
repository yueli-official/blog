ALTER TABLE author_profiles ALTER COLUMN role SET DEFAULT 'guest';
UPDATE author_profiles SET role = 'guest' WHERE role = 'contributor';
UPDATE author_profiles SET role = 'lead' WHERE role = 'author';
