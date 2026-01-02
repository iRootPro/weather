package models

import "time"

// NarodmonLog представляет запись отправки данных на Narodmon
type NarodmonLog struct {
	ID           int64     `json:"id" db:"id"`
	SentAt       time.Time `json:"sent_at" db:"sent_at"`
	Success      bool      `json:"success" db:"success"`
	SensorsCount int       `json:"sensors_count" db:"sensors_count"`
	ErrorMessage *string   `json:"error_message,omitempty" db:"error_message"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// NarodmonStatus представляет статус последней отправки для отображения в виджете
type NarodmonStatus struct {
	LastSentAt   *time.Time
	Success      bool
	SensorsCount int
	ErrorMessage string
}
