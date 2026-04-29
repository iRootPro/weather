-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS max_users (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    language_code VARCHAR(20) DEFAULT 'ru',
    is_bot BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_max_users_user_id ON max_users(user_id);
CREATE INDEX idx_max_users_is_active ON max_users(is_active);

CREATE TABLE IF NOT EXISTS max_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES max_users(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, event_type)
);

CREATE INDEX idx_max_subscriptions_user_id ON max_subscriptions(user_id);
CREATE INDEX idx_max_subscriptions_is_active ON max_subscriptions(is_active);
CREATE INDEX idx_max_subscriptions_event_type ON max_subscriptions(event_type);

CREATE TABLE IF NOT EXISTS max_notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES max_users(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    sent_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_max_notifications_user_id ON max_notifications(user_id);
CREATE INDEX idx_max_notifications_sent_at ON max_notifications(sent_at);
CREATE INDEX idx_max_notifications_event_type ON max_notifications(event_type);
CREATE INDEX idx_max_notifications_user_event_sent ON max_notifications(user_id, event_type, sent_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS max_notifications;
DROP TABLE IF EXISTS max_subscriptions;
DROP TABLE IF EXISTS max_users;

-- +goose StatementEnd
