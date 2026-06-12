-- P8 Stage 0: extend idempotency TTL 24h -> 30 days.
-- Multi-day offline replays (mobile mutation queue) need stored responses to
-- outlive a long disconnection. Affects new rows only (column DEFAULT).
ALTER TABLE vehicle.idempotency_keys
    ALTER COLUMN expires_at SET DEFAULT now() + INTERVAL '30 days';
