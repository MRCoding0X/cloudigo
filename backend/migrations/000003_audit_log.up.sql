-- Fas 5: a real, actually-written audit log — replacing the legacy
-- droppy_log table, which had no INSERT path anywhere in the original code
-- (see PROJECT_DOCUMENTATION.md §19.5).

CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_email VARCHAR(255) NOT NULL DEFAULT '',
    event_type  VARCHAR(100) NOT NULL,
    details     TEXT NOT NULL DEFAULT '',
    ip          VARCHAR(64) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_created_at ON audit_log(created_at DESC);
