CREATE TABLE blog_traffic_source_receipts (
    event_id   TEXT PRIMARY KEY,
    day        DATE NOT NULL,
    source     TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_blog_traffic_source_receipts_day
    ON blog_traffic_source_receipts (day);

CREATE TABLE blog_traffic_source_daily (
    day    DATE NOT NULL,
    source TEXT NOT NULL,
    views  BIGINT NOT NULL DEFAULT 0 CHECK (views >= 0),
    PRIMARY KEY (day, source)
);

CREATE INDEX ix_blog_traffic_source_daily_rank
    ON blog_traffic_source_daily (day, views DESC);
