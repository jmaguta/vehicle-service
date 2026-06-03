CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS vehicle;

CREATE TABLE IF NOT EXISTS vehicle.customers (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workshop_id             UUID NOT NULL,
    name                    TEXT NOT NULL,
    email                   TEXT,
    phone                   TEXT,
    address                 TEXT,
    notes                   TEXT,
    active                  BOOLEAN NOT NULL DEFAULT true,
    operator_licence_number TEXT,
    traffic_area            TEXT,
    company_reg_number      TEXT,
    licence_type            TEXT,
    transport_manager_name  TEXT,
    transport_manager_cpc   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS vehicle.vehicles (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workshop_id               UUID NOT NULL,
    customer_id               UUID REFERENCES vehicle.customers(id),
    registration              TEXT NOT NULL,
    truck_number              TEXT,
    make_model                TEXT,
    mileage                   NUMERIC,
    notes                     TEXT,
    active                    BOOLEAN NOT NULL DEFAULT true,
    vin                       TEXT,
    dvsa_class                TEXT,
    gross_vehicle_weight      NUMERIC,
    axle_count                INT,
    date_first_registered     DATE,
    mot_expiry                DATE,
    tachograph_cal_due        DATE,
    inspection_interval_weeks INT DEFAULT 8,
    last_pmi_date             DATE,
    next_pmi_due              DATE,
    has_adbluedpf             BOOLEAN NOT NULL DEFAULT false,
    vor_status                BOOLEAN NOT NULL DEFAULT false,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

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
