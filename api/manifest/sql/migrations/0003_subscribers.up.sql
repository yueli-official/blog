CREATE TABLE subscribers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',  -- pending / confirmed / unsubscribed
    confirm_token TEXT NOT NULL DEFAULT '',          -- per-subscriber secret: confirm + unsubscribe links
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at  TIMESTAMPTZ
);
CREATE UNIQUE INDEX uq_subscribers_email ON subscribers (email);
CREATE INDEX        ix_subscribers_status ON subscribers (status);
CREATE INDEX        ix_subscribers_token  ON subscribers (confirm_token);
