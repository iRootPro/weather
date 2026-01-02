-- +goose Up
CREATE TABLE IF NOT EXISTS narodmon_logs (
    id BIGSERIAL PRIMARY KEY,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    success BOOLEAN NOT NULL,
    sensors_count INT NOT NULL,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_narodmon_logs_sent_at ON narodmon_logs(sent_at DESC);
CREATE INDEX idx_narodmon_logs_success ON narodmon_logs(success);

-- +goose Down
DROP TABLE IF EXISTS narodmon_logs;
