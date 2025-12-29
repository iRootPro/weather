-- +goose Up
-- +goose StatementBegin
CREATE TABLE photos (
    id SERIAL PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(512) NOT NULL,
    caption TEXT,
    taken_at TIMESTAMPTZ NOT NULL,
    uploaded_at TIMESTAMPTZ DEFAULT NOW(),

    -- Погодные данные на момент съемки
    temperature NUMERIC(5,2),
    humidity NUMERIC(5,2),
    pressure NUMERIC(6,2),
    wind_speed NUMERIC(5,2),
    wind_direction INTEGER,
    rain_rate NUMERIC(5,2),
    solar_radiation NUMERIC(7,2),
    weather_description TEXT,

    -- EXIF метаданные
    camera_make VARCHAR(100),
    camera_model VARCHAR(100),

    -- Telegram метаданные
    telegram_file_id VARCHAR(255),
    telegram_user_id BIGINT REFERENCES telegram_users(id) ON DELETE SET NULL,

    is_visible BOOLEAN DEFAULT true,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Индексы для быстрого поиска
CREATE INDEX idx_photos_taken_at ON photos(taken_at DESC);
CREATE INDEX idx_photos_is_visible ON photos(is_visible) WHERE is_visible = true;
CREATE INDEX idx_photos_telegram_user ON photos(telegram_user_id);

-- Триггер для обновления updated_at
CREATE OR REPLACE FUNCTION update_photos_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER photos_updated_at
    BEFORE UPDATE ON photos
    FOR EACH ROW
    EXECUTE FUNCTION update_photos_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS photos_updated_at ON photos;
DROP FUNCTION IF EXISTS update_photos_updated_at();
DROP TABLE IF EXISTS photos;
-- +goose StatementEnd
