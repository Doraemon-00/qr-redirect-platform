CREATE DATABASE IF NOT EXISTS qr_analytics;

CREATE TABLE IF NOT EXISTS qr_analytics.scan_events (
    event_id UUID,
    token String,
    scanned_at DateTime64(3, 'UTC'),
    user_agent_hash String,
    ip_hash String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(scanned_at)
ORDER BY (token, scanned_at);
