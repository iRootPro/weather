-- +goose Up
-- +goose StatementBegin

-- Таблица пользователей Telegram
CREATE TABLE IF NOT EXISTS telegram_users (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    language_code VARCHAR(10) DEFAULT 'ru',
    is_bot BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_telegram_users_chat_id ON telegram_users(chat_id);
CREATE INDEX idx_telegram_users_is_active ON telegram_users(is_active);

-- Таблица подписок на погодные события
CREATE TABLE IF NOT EXISTS telegram_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES telegram_users(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL, -- 'all', 'rain', 'temperature', 'wind', 'pressure'
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, event_type)
);

CREATE INDEX idx_telegram_subscriptions_user_id ON telegram_subscriptions(user_id);
CREATE INDEX idx_telegram_subscriptions_is_active ON telegram_subscriptions(is_active);
CREATE INDEX idx_telegram_subscriptions_event_type ON telegram_subscriptions(event_type);

-- Таблица истории отправленных уведомлений (для предотвращения дублей)
CREATE TABLE IF NOT EXISTS telegram_notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES telegram_users(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    sent_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_telegram_notifications_user_id ON telegram_notifications(user_id);
CREATE INDEX idx_telegram_notifications_sent_at ON telegram_notifications(sent_at);
CREATE INDEX idx_telegram_notifications_event_type ON telegram_notifications(event_type);
-- Составной индекс для быстрой проверки "было ли уведомление отправлено недавно"
CREATE INDEX idx_telegram_notifications_user_event_sent ON telegram_notifications(user_id, event_type, sent_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS telegram_notifications;
DROP TABLE IF EXISTS telegram_subscriptions;
DROP TABLE IF EXISTS telegram_users;

-- +goose StatementEnd
