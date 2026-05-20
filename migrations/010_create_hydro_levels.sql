-- +goose Up
-- +goose StatementBegin

-- Гидропосты и уровни воды из pub.emercit.ru
CREATE TABLE IF NOT EXISTS hydro_gauges (
    station_uuid TEXT PRIMARY KEY,
    waterlevel_uuid TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    short_name TEXT,
    holder_name TEXT,
    area TEXT,
    district TEXT,
    locality TEXT,
    monitoring_object TEXT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    fix_bs_m REAL,
    dry_bs_m REAL,
    flooding_prevention_bs_m REAL,
    flooding_danger_bs_m REAL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hydro_level_readings (
    observed_at TIMESTAMPTZ NOT NULL,
    station_uuid TEXT NOT NULL,
    waterlevel_uuid TEXT NOT NULL,
    level_bs_m REAL NOT NULL,
    level_zero_m REAL,
    change_cm_per_hour REAL,
    lead_text TEXT,
    state_code INTEGER,
    level_code INTEGER,
    raw_data JSONB,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (observed_at, station_uuid)
);

SELECT create_hypertable('hydro_level_readings', 'observed_at', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_hydro_level_station_time ON hydro_level_readings (station_uuid, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_hydro_level_waterlevel_time ON hydro_level_readings (waterlevel_uuid, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_hydro_level_level_code ON hydro_level_readings (level_code) WHERE level_code IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS hydro_level_readings;
DROP TABLE IF EXISTS hydro_gauges;

-- +goose StatementEnd
