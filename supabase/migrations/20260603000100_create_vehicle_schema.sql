CREATE SCHEMA IF NOT EXISTS vehicle;

-- Authoritative customers table
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
CREATE INDEX IF NOT EXISTS idx_vehicle_customers_workshop ON vehicle.customers(workshop_id);

-- Authoritative vehicles table
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
CREATE INDEX IF NOT EXISTS idx_vehicle_vehicles_workshop    ON vehicle.vehicles(workshop_id);
CREATE INDEX IF NOT EXISTS idx_vehicle_vehicles_customer_id ON vehicle.vehicles(customer_id);

-- Migrate existing data from jobs schema
INSERT INTO vehicle.customers (
    id, workshop_id, name, email, phone, address, notes, active,
    operator_licence_number, traffic_area, company_reg_number,
    licence_type, transport_manager_name, transport_manager_cpc,
    created_at, updated_at
)
SELECT
    id, workshop_id, name, email, phone, address, notes, active,
    operator_licence_number, traffic_area, company_reg_number,
    licence_type, transport_manager_name, transport_manager_cpc,
    created_at, updated_at
FROM jobs.customers
ON CONFLICT (id) DO NOTHING;

INSERT INTO vehicle.vehicles (
    id, workshop_id, customer_id, registration, truck_number, make_model,
    mileage, notes, active, vin, dvsa_class, gross_vehicle_weight, axle_count,
    date_first_registered, mot_expiry, tachograph_cal_due,
    inspection_interval_weeks, last_pmi_date, next_pmi_due,
    has_adbluedpf, vor_status, created_at, updated_at
)
SELECT
    id, workshop_id, customer_id, registration, truck_number, make_model,
    mileage, notes, active, vin, dvsa_class, gross_vehicle_weight, axle_count,
    date_first_registered, mot_expiry, tachograph_cal_due,
    inspection_interval_weeks, last_pmi_date, next_pmi_due,
    has_adbluedpf, vor_status, created_at, updated_at
FROM jobs.vehicles
ON CONFLICT (id) DO NOTHING;

-- Replace source tables with views for backward compat (job-service queries unchanged)
DROP TABLE IF EXISTS jobs.vehicles;
DROP TABLE IF EXISTS jobs.customers;

CREATE OR REPLACE VIEW jobs.customers AS SELECT * FROM vehicle.customers;
CREATE OR REPLACE VIEW jobs.vehicles  AS SELECT * FROM vehicle.vehicles;
