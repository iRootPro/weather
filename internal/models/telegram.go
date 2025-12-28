package models

import (
	"encoding/json"
	"time"
)

// TelegramUser представляет пользователя Telegram
type TelegramUser struct {
	ID           int64     `json:"id" db:"id"`
	ChatID       int64     `json:"chat_id" db:"chat_id"`
	Username     *string   `json:"username,omitempty" db:"username"`
	FirstName    *string   `json:"first_name,omitempty" db:"first_name"`
	LastName     *string   `json:"last_name,omitempty" db:"last_name"`
	LanguageCode string    `json:"language_code" db:"language_code"`
	IsBot        bool      `json:"is_bot" db:"is_bot"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// TelegramSubscription представляет подписку пользователя на погодные события
type TelegramSubscription struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	EventType string    `json:"event_type" db:"event_type"` // all, rain, temperature, wind, pressure
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TelegramNotification представляет отправленное уведомление
type TelegramNotification struct {
	ID        int64           `json:"id" db:"id"`
	UserID    int64           `json:"user_id" db:"user_id"`
	EventType string          `json:"event_type" db:"event_type"`
	EventData json.RawMessage `json:"event_data" db:"event_data"`
	SentAt    time.Time       `json:"sent_at" db:"sent_at"`
}
