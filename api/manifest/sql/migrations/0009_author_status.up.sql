-- M5 write gate: an author profile starts as a 'pending' request; a superadmin
-- approves it to 'active' before the author may write. Existing profiles (created
-- before the gate) are grandfathered to 'active' so current authors keep writing.
ALTER TABLE author_profiles ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';
UPDATE author_profiles SET status = 'active';
