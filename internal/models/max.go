package models

import (
	"encoding/json"
	"time"
)

// MaxUser represents a user of the Max bot.
type MaxUser struct {
	ID           int64     `json:"id" db:"id"`
	UserID       int64     `json:"user_id" db:"user_id"`
	Username     *string   `json:"username,omitempty" db:"username"`
	FirstName    *string   `json:"first_name,omitempty" db:"first_name"`
	LastName     *string   `json:"last_name,omitempty" db:"last_name"`
	LanguageCode string    `json:"language_code" db:"language_code"`
	IsBot        bool      `json:"is_bot" db:"is_bot"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// MaxSubscription represents a Max user subscription to weather events.
type MaxSubscription struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	EventType string    `json:"event_type" db:"event_type"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// MaxNotification represents a notification sent by the Max bot.
type MaxNotification struct {
	ID        int64           `json:"id" db:"id"`
	UserID    int64           `json:"user_id" db:"user_id"`
	EventType string          `json:"event_type" db:"event_type"`
	EventData json.RawMessage `json:"event_data" db:"event_data"`
	SentAt    time.Time       `json:"sent_at" db:"sent_at"`
}
