package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iRootPro/weather/internal/models"
)

type sensorRepository struct {
	pool *pgxpool.Pool
}

func NewSensorRepository(pool *pgxpool.Pool) SensorRepository {
	return &sensorRepository{pool: pool}
}

func (r *sensorRepository) GetAll(ctx context.Context) ([]models.Sensor, error) {
	query := `
		SELECT id, code, name, unit, description, created_at
		FROM sensors
		ORDER BY id`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sensors: %w", err)
	}
	defer rows.Close()

	var result []models.Sensor
	for rows.Next() {
		var s models.Sensor
		err := rows.Scan(&s.ID, &s.Code, &s.Name, &s.Unit, &s.Description, &s.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sensor: %w", err)
		}
		result = append(result, s)
	}

	return result, nil
}

func (r *sensorRepository) GetByCode(ctx context.Context, code string) (*models.Sensor, error) {
	query := `
		SELECT id, code, name, unit, description, created_at
		FROM sensors
		WHERE code = $1`

	var s models.Sensor
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&s.ID, &s.Code, &s.Name, &s.Unit, &s.Description, &s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor by code: %w", err)
	}

	return &s, nil
}
