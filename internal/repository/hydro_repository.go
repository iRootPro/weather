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

type hydroRepository struct {
	pool *pgxpool.Pool
}

func NewHydroRepository(pool *pgxpool.Pool) HydroRepository {
	return &hydroRepository{pool: pool}
}

func (r *hydroRepository) SaveGauge(ctx context.Context, gauge *models.HydroGauge) error {
	query := `
		INSERT INTO hydro_gauges (
			station_uuid, waterlevel_uuid, name, short_name, holder_name, area, district, locality,
			monitoring_object, latitude, longitude, fix_bs_m, dry_bs_m,
			flooding_prevention_bs_m, flooding_danger_bs_m, fetched_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (station_uuid) DO UPDATE SET
			waterlevel_uuid = EXCLUDED.waterlevel_uuid,
			name = EXCLUDED.name,
			short_name = EXCLUDED.short_name,
			holder_name = EXCLUDED.holder_name,
			area = EXCLUDED.area,
			district = EXCLUDED.district,
			locality = EXCLUDED.locality,
			monitoring_object = EXCLUDED.monitoring_object,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			fix_bs_m = EXCLUDED.fix_bs_m,
			dry_bs_m = EXCLUDED.dry_bs_m,
			flooding_prevention_bs_m = EXCLUDED.flooding_prevention_bs_m,
			flooding_danger_bs_m = EXCLUDED.flooding_danger_bs_m,
			fetched_at = EXCLUDED.fetched_at`
	_, err := r.pool.Exec(ctx, query,
		gauge.StationUUID, gauge.WaterLevelUUID, gauge.Name, gauge.ShortName, gauge.HolderName,
		gauge.Area, gauge.District, gauge.Locality, gauge.MonitoringObject, gauge.Latitude, gauge.Longitude,
		gauge.FixBSM, gauge.DryBSM, gauge.FloodingPreventionBM, gauge.FloodingDangerBSM, gauge.FetchedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save hydro gauge: %w", err)
	}
	return nil
}

func (r *hydroRepository) SaveReadingsBatch(ctx context.Context, data []models.HydroLevelReading) error {
	if len(data) == 0 {
		return nil
	}
	query := `
		INSERT INTO hydro_level_readings (
			observed_at, station_uuid, waterlevel_uuid, level_bs_m, level_zero_m,
			change_cm_per_hour, lead_text, state_code, level_code, raw_data, fetched_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (observed_at, station_uuid) DO UPDATE SET
			waterlevel_uuid = EXCLUDED.waterlevel_uuid,
			level_bs_m = EXCLUDED.level_bs_m,
			level_zero_m = EXCLUDED.level_zero_m,
			change_cm_per_hour = COALESCE(EXCLUDED.change_cm_per_hour, hydro_level_readings.change_cm_per_hour),
			lead_text = COALESCE(EXCLUDED.lead_text, hydro_level_readings.lead_text),
			state_code = COALESCE(EXCLUDED.state_code, hydro_level_readings.state_code),
			level_code = COALESCE(EXCLUDED.level_code, hydro_level_readings.level_code),
			raw_data = COALESCE(EXCLUDED.raw_data, hydro_level_readings.raw_data),
			fetched_at = EXCLUDED.fetched_at`

	batch := &pgx.Batch{}
	for _, d := range data {
		batch.Queue(query, d.ObservedAt, d.StationUUID, d.WaterLevelUUID, d.LevelBSM, d.LevelZeroM,
			d.ChangeCmPerHour, d.LeadText, d.StateCode, d.LevelCode, d.RawData, d.FetchedAt)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for i := range data {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save hydro reading %d: %w", i, err)
		}
	}
	return nil
}

func (r *hydroRepository) GetGauge(ctx context.Context, stationUUID string) (*models.HydroGauge, error) {
	query := `SELECT station_uuid, waterlevel_uuid, name, short_name, holder_name, area, district, locality,
		monitoring_object, latitude, longitude, fix_bs_m, dry_bs_m, flooding_prevention_bs_m,
		flooding_danger_bs_m, fetched_at
		FROM hydro_gauges WHERE station_uuid = $1`
	row := r.pool.QueryRow(ctx, query, stationUUID)
	var g models.HydroGauge
	if err := row.Scan(&g.StationUUID, &g.WaterLevelUUID, &g.Name, &g.ShortName, &g.HolderName, &g.Area,
		&g.District, &g.Locality, &g.MonitoringObject, &g.Latitude, &g.Longitude, &g.FixBSM, &g.DryBSM,
		&g.FloodingPreventionBM, &g.FloodingDangerBSM, &g.FetchedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get hydro gauge: %w", err)
	}
	return &g, nil
}

func (r *hydroRepository) GetLatest(ctx context.Context, stationUUID string) (*models.HydroLevelReading, error) {
	query := `SELECT observed_at, station_uuid, waterlevel_uuid, level_bs_m, level_zero_m,
		change_cm_per_hour, lead_text, state_code, level_code, raw_data, fetched_at
		FROM hydro_level_readings WHERE station_uuid = $1 ORDER BY observed_at DESC LIMIT 1`
	rows, err := r.queryReadings(ctx, query, stationUUID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (r *hydroRepository) GetPreviousBefore(ctx context.Context, stationUUID string, before time.Time) (*models.HydroLevelReading, error) {
	query := `SELECT observed_at, station_uuid, waterlevel_uuid, level_bs_m, level_zero_m,
		change_cm_per_hour, lead_text, state_code, level_code, raw_data, fetched_at
		FROM hydro_level_readings WHERE station_uuid = $1 AND observed_at < $2 ORDER BY observed_at DESC LIMIT 1`
	rows, err := r.queryReadings(ctx, query, stationUUID, before)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (r *hydroRepository) GetNearBefore(ctx context.Context, stationUUID string, target time.Time, window time.Duration) (*models.HydroLevelReading, error) {
	query := `SELECT observed_at, station_uuid, waterlevel_uuid, level_bs_m, level_zero_m,
		change_cm_per_hour, lead_text, state_code, level_code, raw_data, fetched_at
		FROM hydro_level_readings
		WHERE station_uuid = $1 AND observed_at <= $2 AND observed_at >= $3
		ORDER BY observed_at DESC LIMIT 1`
	rows, err := r.queryReadings(ctx, query, stationUUID, target, target.Add(-window))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (r *hydroRepository) GetRange(ctx context.Context, stationUUID string, from, to time.Time) ([]models.HydroLevelReading, error) {
	query := `SELECT observed_at, station_uuid, waterlevel_uuid, level_bs_m, level_zero_m,
		change_cm_per_hour, lead_text, state_code, level_code, raw_data, fetched_at
		FROM hydro_level_readings
		WHERE station_uuid = $1 AND observed_at >= $2 AND observed_at <= $3
		ORDER BY observed_at ASC`
	return r.queryReadings(ctx, query, stationUUID, from, to)
}

func (r *hydroRepository) DeleteOlderThan(ctx context.Context, threshold time.Time) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM hydro_level_readings WHERE observed_at < $1`, threshold)
	if err != nil {
		return fmt.Errorf("failed to delete old hydro readings: %w", err)
	}
	return nil
}

func (r *hydroRepository) queryReadings(ctx context.Context, query string, args ...any) ([]models.HydroLevelReading, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query hydro readings: %w", err)
	}
	defer rows.Close()

	var out []models.HydroLevelReading
	for rows.Next() {
		var d models.HydroLevelReading
		if err := rows.Scan(&d.ObservedAt, &d.StationUUID, &d.WaterLevelUUID, &d.LevelBSM, &d.LevelZeroM,
			&d.ChangeCmPerHour, &d.LeadText, &d.StateCode, &d.LevelCode, &d.RawData, &d.FetchedAt); err != nil {
			return nil, fmt.Errorf("failed to scan hydro reading: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return out, nil
}
