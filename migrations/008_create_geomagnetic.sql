-- +goose Up
-- +goose StatementBegin

-- Таблица 3-часовых слотов геомагнитного индекса Kp
CREATE TABLE IF NOT EXISTS geomagnetic_kp (
    slot_time   TIMESTAMPTZ NOT NULL,
    kp          REAL        NOT NULL,
    source      TEXT        NOT NULL DEFAULT 'xras.ru',
    is_forecast BOOLEAN     NOT NULL,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (slot_time, source)
);

-- Преобразуем в hypertable TimescaleDB
SELECT create_hypertable('geomagnetic_kp', 'slot_time', if_not_exists => TRUE);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_geomag_kp_time ON geomagnetic_kp (slot_time DESC);
CREATE INDEX IF NOT EXISTS idx_geomag_kp_forecast ON geomagnetic_kp (slot_time DESC) WHERE is_forecast = TRUE;

-- Таблица дневных показателей солнечной активности
CREATE TABLE IF NOT EXISTS geomagnetic_daily (
    date       DATE        PRIMARY KEY,
    f10        REAL,        -- солнечный поток 10.7 см
    sn         REAL,        -- число Вольфа (солнечные пятна)
    ap         REAL,        -- планетарный индекс возмущений (нТл)
    max_kp     REAL,        -- максимум Kp за сутки
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS geomagnetic_daily;
DROP TABLE IF EXISTS geomagnetic_kp;

-- +goose StatementEnd
