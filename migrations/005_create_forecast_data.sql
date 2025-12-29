-- +goose Up
-- +goose StatementBegin

-- Таблица данных прогноза погоды
CREATE TABLE IF NOT EXISTS forecast_data (
    id BIGSERIAL,
    forecast_time TIMESTAMPTZ NOT NULL,

    -- Температура (°C)
    temperature REAL,
    temperature_min REAL,
    temperature_max REAL,
    feels_like REAL,

    -- Осадки
    precipitation_probability SMALLINT,  -- вероятность осадков (%)
    precipitation REAL,                  -- количество осадков (мм)

    -- Ветер
    wind_speed REAL,            -- м/с
    wind_direction SMALLINT,    -- градусы 0-360
    wind_gusts REAL,            -- порывы м/с

    -- Облачность и другое
    cloud_cover SMALLINT,       -- облачность (%)
    pressure REAL,              -- давление (гПа)
    humidity SMALLINT,          -- влажность (%)
    uv_index REAL,              -- UV индекс

    -- Описание погоды
    weather_code SMALLINT,      -- код погоды WMO
    weather_description TEXT,   -- описание (ясно, облачно, дождь и т.д.)

    -- Тип прогноза
    forecast_type TEXT NOT NULL CHECK (forecast_type IN ('hourly', 'daily')),

    -- Время получения данных
    fetched_at TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (id, forecast_time)
);

-- Преобразуем в hypertable TimescaleDB
SELECT create_hypertable('forecast_data', 'forecast_time', if_not_exists => TRUE);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_forecast_data_time ON forecast_data (forecast_time DESC);
CREATE INDEX IF NOT EXISTS idx_forecast_data_type ON forecast_data (forecast_type);
CREATE INDEX IF NOT EXISTS idx_forecast_data_fetched ON forecast_data (fetched_at DESC);

-- Уникальное ограничение для предотвращения дублей
CREATE UNIQUE INDEX IF NOT EXISTS idx_forecast_unique ON forecast_data (forecast_time, forecast_type);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS forecast_data;

-- +goose StatementEnd
