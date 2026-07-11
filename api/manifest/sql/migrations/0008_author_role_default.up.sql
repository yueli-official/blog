-- M5 governance: a newly-registered author defaults to 客座 (guest); an admin
-- promotes trusted writers to 主笔 (lead) via /manage/authors. (0007 defaulted
-- to 'lead' when role was self-set; role is now admin-governed.)
ALTER TABLE author_profiles ALTER COLUMN role SET DEFAULT 'guest';
