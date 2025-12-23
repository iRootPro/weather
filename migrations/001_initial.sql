-- +goose Up
-- +goose StatementBegin

-- Включаем расширение TimescaleDB
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Таблица сенсоров
CREATE TABLE IF NOT EXISTS sensors (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Основная таблица погодных данных
CREATE TABLE IF NOT EXISTS weather_data (
    time TIMESTAMPTZ NOT NULL,

    -- Температура (°C)
    temp_outdoor REAL,
    temp_indoor REAL,

    -- Влажность (%)
    humidity_outdoor SMALLINT,
    humidity_indoor SMALLINT,

    -- Давление (мм рт. ст.)
    pressure_relative REAL,
    pressure_absolute REAL,

    -- Ветер
    wind_speed REAL,           -- м/с
    wind_gust REAL,            -- м/с
    wind_direction SMALLINT,   -- градусы 0-360

    -- Осадки (мм)
    rain_rate REAL,            -- мм/ч
    rain_daily REAL,
    rain_weekly REAL,
    rain_monthly REAL,
    rain_yearly REAL,

    -- Солнце
    uv_index REAL,
    solar_radiation REAL,      -- Вт/м²

    -- Дополнительные датчики
    temp_feels_like REAL,
    dew_point REAL,

    -- Сырые данные от станции (JSON)
    raw_data JSONB
);

-- Преобразуем в hypertable TimescaleDB
SELECT create_hypertable('weather_data', 'time', if_not_exists => TRUE);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_weather_data_time ON weather_data (time DESC);

-- Заполняем таблицу сенсоров начальными данными
INSERT INTO sensors (code, name, unit, description) VALUES
    ('temp_outdoor', 'Температура (улица)', '°C', 'Температура наружного воздуха'),
    ('temp_indoor', 'Температура (дом)', '°C', 'Температура внутри помещения'),
    ('humidity_outdoor', 'Влажность (улица)', '%', 'Относительная влажность наружного воздуха'),
    ('humidity_indoor', 'Влажность (дом)', '%', 'Относительная влажность внутри помещения'),
    ('pressure_relative', 'Давление (отн.)', 'мм рт.ст.', 'Относительное атмосферное давление'),
    ('pressure_absolute', 'Давление (абс.)', 'мм рт.ст.', 'Абсолютное атмосферное давление'),
    ('wind_speed', 'Скорость ветра', 'м/с', 'Текущая скорость ветра'),
    ('wind_gust', 'Порывы ветра', 'м/с', 'Максимальная скорость порывов'),
    ('wind_direction', 'Направление ветра', '°', 'Направление ветра в градусах'),
    ('rain_rate', 'Интенсивность дождя', 'мм/ч', 'Текущая интенсивность осадков'),
    ('rain_daily', 'Осадки за день', 'мм', 'Количество осадков за сутки'),
    ('uv_index', 'UV индекс', '', 'Индекс ультрафиолетового излучения'),
    ('solar_radiation', 'Солнечная радиация', 'Вт/м²', 'Интенсивность солнечного излучения')
ON CONFLICT (code) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS weather_data;
DROP TABLE IF EXISTS sensors;

-- +goose StatementEnd
