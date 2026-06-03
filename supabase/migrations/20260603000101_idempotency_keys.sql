CREATE TABLE IF NOT EXISTS vehicle.idempotency_keys (
    mutation_id TEXT        NOT NULL,
    workshop_id TEXT        NOT NULL DEFAULT '',
    path        TEXT        NOT NULL,
    status_code INT         NOT NULL,
    response    BYTEA       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '24 hours',
    PRIMARY KEY (mutation_id, workshop_id)
);
CREATE INDEX IF NOT EXISTS idx_vehicle_idempotency_expires ON vehicle.idempotency_keys(expires_at);
