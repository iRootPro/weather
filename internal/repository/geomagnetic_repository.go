package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iRootPro/weather/internal/models"
)

type geomagneticRepository struct {
	pool *pgxpool.Pool
}

func NewGeomagneticRepository(pool *pgxpool.Pool) GeomagneticRepository {
	return &geomagneticRepository{pool: pool}
}

// SaveKpBatch вставляет/обновляет 3-часовые слоты Kp.
// UPSERT обновляет только слоты не старше 24 часов — прошлое не переписываем.
func (r *geomagneticRepository) SaveKpBatch(ctx context.Context, data []models.GeomagneticKp) error {
	if len(data) == 0 {
		return nil
	}

	query := `
		INSERT INTO geomagnetic_kp (slot_time, kp, source, is_forecast, fetched_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (slot_time, source) DO UPDATE SET
			kp = EXCLUDED.kp,
			is_forecast = EXCLUDED.is_forecast,
			fetched_at = EXCLUDED.fetched_at
		WHERE geomagnetic_kp.slot_time > NOW() - INTERVAL '24 hours'`

	batch := &pgx.Batch{}
	for _, d := range data {
		batch.Queue(query, d.SlotTime, d.Kp, d.Source, d.IsForecast, d.FetchedAt)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := range data {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to execute kp batch item %d: %w", i, err)
		}
	}
	return nil
}

// SaveDailyBatch вставляет/обновляет суточные показатели солнечной активности.
func (r *geomagneticRepository) SaveDailyBatch(ctx context.Context, data []models.GeomagneticDaily) error {
	if len(data) == 0 {
		return nil
	}

	query := `
		INSERT INTO geomagnetic_daily (date, f10, sn, ap, max_kp, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (date) DO UPDATE SET
			f10 = EXCLUDED.f10,
			sn = EXCLUDED.sn,
			ap = EXCLUDED.ap,
			max_kp = EXCLUDED.max_kp,
			fetched_at = EXCLUDED.fetched_at`

	batch := &pgx.Batch{}
	for _, d := range data {
		batch.Queue(query, d.Date, d.F10, d.Sn, d.Ap, d.MaxKp, d.FetchedAt)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := range data {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to execute daily batch item %d: %w", i, err)
		}
	}
	return nil
}

// GetCurrentKp возвращает последний слот в окне (now-6h .. now]. Окно шире
// интервала источника (3ч) с запасом: между публикациями новых данных
// карточка не должна пропадать. Если ничего нет — nil без ошибки.
func (r *geomagneticRepository) GetCurrentKp(ctx context.Context, now time.Time) (*models.GeomagneticKp, error) {
	query := `
		SELECT slot_time, kp, source, is_forecast, fetched_at
		FROM geomagnetic_kp
		WHERE slot_time <= $1
			AND slot_time > $1 - INTERVAL '6 hours'
		ORDER BY slot_time DESC
		LIMIT 1`

	row := r.pool.QueryRow(ctx, query, now)
	var d models.GeomagneticKp
	err := row.Scan(&d.SlotTime, &d.Kp, &d.Source, &d.IsForecast, &d.FetchedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query current kp: %w", err)
	}
	return &d, nil
}

// GetKpRange возвращает все слоты в полуоткрытом интервале [from, to].
func (r *geomagneticRepository) GetKpRange(ctx context.Context, from, to time.Time) ([]models.GeomagneticKp, error) {
	query := `
		SELECT slot_time, kp, source, is_forecast, fetched_at
		FROM geomagnetic_kp
		WHERE slot_time >= $1 AND slot_time <= $2
		ORDER BY slot_time ASC`

	return r.queryKp(ctx, query, from, to)
}

// GetMaxKpForDay возвращает слот с максимальным Kp в пределах дня (по локальному времени).
func (r *geomagneticRepository) GetMaxKpForDay(ctx context.Context, day time.Time) (*models.GeomagneticKp, error) {
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	query := `
		SELECT slot_time, kp, source, is_forecast, fetched_at
		FROM geomagnetic_kp
		WHERE slot_time >= $1 AND slot_time < $2
		ORDER BY kp DESC, slot_time ASC
		LIMIT 1`

	row := r.pool.QueryRow(ctx, query, dayStart, dayEnd)
	var d models.GeomagneticKp
	err := row.Scan(&d.SlotTime, &d.Kp, &d.Source, &d.IsForecast, &d.FetchedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query max kp for day: %w", err)
	}
	return &d, nil
}

// GetForecastedStorms возвращает прогнозные слоты в [from, to] с Kp >= threshold.
func (r *geomagneticRepository) GetForecastedStorms(ctx context.Context, from, to time.Time, threshold float32) ([]models.GeomagneticKp, error) {
	query := `
		SELECT slot_time, kp, source, is_forecast, fetched_at
		FROM geomagnetic_kp
		WHERE slot_time >= $1 AND slot_time <= $2 AND kp >= $3
		ORDER BY slot_time ASC`

	return r.queryKp(ctx, query, from, to, threshold)
}

// GetDailyRange возвращает суточные показатели в интервале дат.
func (r *geomagneticRepository) GetDailyRange(ctx context.Context, from, to time.Time) ([]models.GeomagneticDaily, error) {
	query := `
		SELECT date, f10, sn, ap, max_kp, fetched_at
		FROM geomagnetic_daily
		WHERE date >= $1 AND date <= $2
		ORDER BY date ASC`

	rows, err := r.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query geomagnetic_daily: %w", err)
	}
	defer rows.Close()

	var out []models.GeomagneticDaily
	for rows.Next() {
		var d models.GeomagneticDaily
		if err := rows.Scan(&d.Date, &d.F10, &d.Sn, &d.Ap, &d.MaxKp, &d.FetchedAt); err != nil {
			return nil, fmt.Errorf("failed to scan geomagnetic_daily: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return out, nil
}

// DeleteOlderThan удаляет слоты старше threshold. Используется для retention.
func (r *geomagneticRepository) DeleteOlderThan(ctx context.Context, threshold time.Time) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM geomagnetic_kp WHERE slot_time < $1`, threshold); err != nil {
		return fmt.Errorf("failed to delete old geomagnetic_kp: %w", err)
	}
	if _, err := r.pool.Exec(ctx, `DELETE FROM geomagnetic_daily WHERE date < $1`, threshold); err != nil {
		return fmt.Errorf("failed to delete old geomagnetic_daily: %w", err)
	}
	return nil
}

func (r *geomagneticRepository) queryKp(ctx context.Context, query string, args ...any) ([]models.GeomagneticKp, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query geomagnetic_kp: %w", err)
	}
	defer rows.Close()

	var out []models.GeomagneticKp
	for rows.Next() {
		var d models.GeomagneticKp
		if err := rows.Scan(&d.SlotTime, &d.Kp, &d.Source, &d.IsForecast, &d.FetchedAt); err != nil {
			return nil, fmt.Errorf("failed to scan geomagnetic_kp: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return out, nil
}
