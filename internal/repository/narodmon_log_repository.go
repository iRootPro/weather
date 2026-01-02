package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/iRootPro/weather/internal/models"
)

type narodmonLogRepository struct {
	pool *pgxpool.Pool
}

func NewNarodmonLogRepository(pool *pgxpool.Pool) NarodmonLogRepository {
	return &narodmonLogRepository{pool: pool}
}

func (r *narodmonLogRepository) Create(ctx context.Context, log *models.NarodmonLog) error {
	query := `
		INSERT INTO narodmon_logs (sent_at, success, sensors_count, error_message)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	return r.pool.QueryRow(ctx, query,
		log.SentAt,
		log.Success,
		log.SensorsCount,
		log.ErrorMessage,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *narodmonLogRepository) GetLatest(ctx context.Context) (*models.NarodmonLog, error) {
	query := `
		SELECT id, sent_at, success, sensors_count, error_message, created_at
		FROM narodmon_logs
		ORDER BY sent_at DESC
		LIMIT 1
	`

	var log models.NarodmonLog
	err := r.pool.QueryRow(ctx, query).Scan(
		&log.ID,
		&log.SentAt,
		&log.Success,
		&log.SensorsCount,
		&log.ErrorMessage,
		&log.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get latest narodmon log: %w", err)
	}

	return &log, nil
}

func (r *narodmonLogRepository) GetRecent(ctx context.Context, limit int) ([]models.NarodmonLog, error) {
	query := `
		SELECT id, sent_at, success, sensors_count, error_message, created_at
		FROM narodmon_logs
		ORDER BY sent_at DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent narodmon logs: %w", err)
	}
	defer rows.Close()

	var logs []models.NarodmonLog
	for rows.Next() {
		var log models.NarodmonLog
		if err := rows.Scan(
			&log.ID,
			&log.SentAt,
			&log.Success,
			&log.SensorsCount,
			&log.ErrorMessage,
			&log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan narodmon log: %w", err)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate narodmon logs: %w", err)
	}

	return logs, nil
}

func (r *narodmonLogRepository) DeleteOld(ctx context.Context, olderThan time.Time) error {
	query := `
		DELETE FROM narodmon_logs
		WHERE created_at < $1
	`

	_, err := r.pool.Exec(ctx, query, olderThan)
	if err != nil {
		return fmt.Errorf("failed to delete old narodmon logs: %w", err)
	}

	return nil
}
