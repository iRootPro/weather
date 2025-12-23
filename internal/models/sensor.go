package models

import "time"

type Sensor struct {
	ID          int       `json:"id" db:"id"`
	Code        string    `json:"code" db:"code"`
	Name        string    `json:"name" db:"name"`
	Unit        string    `json:"unit" db:"unit"`
	Description string    `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
