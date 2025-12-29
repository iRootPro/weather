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

// Photo представляет фотографию с погодными данными
type Photo struct {
	ID       int64  `json:"id" db:"id"`
	Filename string `json:"filename" db:"filename"`
	FilePath string `json:"file_path" db:"file_path"`
	Caption  string `json:"caption,omitempty" db:"caption"`
	TakenAt  time.Time `json:"taken_at" db:"taken_at"`
	UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`

	// Погодные данные на момент съемки
	Temperature         *float64 `json:"temperature,omitempty" db:"temperature"`
	Humidity            *float64 `json:"humidity,omitempty" db:"humidity"`
	Pressure            *float64 `json:"pressure,omitempty" db:"pressure"`
	WindSpeed           *float64 `json:"wind_speed,omitempty" db:"wind_speed"`
	WindDirection       *int     `json:"wind_direction,omitempty" db:"wind_direction"`
	RainRate            *float64 `json:"rain_rate,omitempty" db:"rain_rate"`
	SolarRadiation      *float64 `json:"solar_radiation,omitempty" db:"solar_radiation"`
	WeatherDescription  string   `json:"weather_description,omitempty" db:"weather_description"`

	// EXIF метаданные
	CameraMake  string `json:"camera_make,omitempty" db:"camera_make"`
	CameraModel string `json:"camera_model,omitempty" db:"camera_model"`

	// Telegram метаданные
	TelegramFileID string  `json:"telegram_file_id,omitempty" db:"telegram_file_id"`
	TelegramUserID *int64  `json:"telegram_user_id,omitempty" db:"telegram_user_id"`

	IsVisible bool      `json:"is_visible" db:"is_visible"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
