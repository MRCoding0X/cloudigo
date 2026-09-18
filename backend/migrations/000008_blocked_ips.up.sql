-- Simple abuse-mitigation blocklist: an admin can ban a specific IP address
-- from creating new uploads, on top of the existing per-IP rate limiting.

CREATE TABLE blocked_ips (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ip         VARCHAR(64) NOT NULL UNIQUE,
    reason     TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
