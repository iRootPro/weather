package repository

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestForecastRepositoryGenerationsAndRollback(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}
	ctx := context.Background()
	bootstrap, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer bootstrap.Close()
	var database string
	if err := bootstrap.QueryRow(ctx, "SELECT current_database()").Scan(&database); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(database), "test") && !strings.Contains(strings.ToLower(database), "_it") {
		t.Fatalf("refusing integration test against non-test database %q", database)
	}
	schema := fmt.Sprintf("forecast_it_%d", time.Now().UnixNano())
	if _, err := bootstrap.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = bootstrap.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE") }()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = 2
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	first, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := pool.Acquire(ctx)
	if err != nil {
		first.Release()
		t.Fatal(err)
	}
	for _, connection := range []*pgxpool.Conn{first, second} {
		var got string
		if err := connection.QueryRow(ctx, "SELECT current_schema()").Scan(&got); err != nil || got != schema {
			t.Fatalf("connection search_path = %q, want %q: %v", got, schema, err)
		}
	}
	second.Release()
	first.Release()
	if _, err := bootstrap.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS timescaledb SCHEMA public"); err != nil {
		t.Fatalf("enable Timescale for migration 005: %v", err)
	}
	_, err = pool.Exec(ctx, `CREATE TABLE forecast_data (
 id BIGSERIAL, forecast_time TIMESTAMPTZ NOT NULL, temperature REAL, temperature_min REAL, temperature_max REAL, feels_like REAL,
 precipitation_probability SMALLINT, precipitation REAL, wind_speed REAL, wind_direction SMALLINT, wind_gusts REAL,
 cloud_cover SMALLINT, pressure REAL, humidity SMALLINT, uv_index REAL, weather_code SMALLINT, weather_description TEXT,
 forecast_type TEXT NOT NULL CHECK (forecast_type IN ('hourly', 'daily')), fetched_at TIMESTAMPTZ DEFAULT NOW(), PRIMARY KEY (id, forecast_time));
 CREATE UNIQUE INDEX idx_forecast_unique ON forecast_data (forecast_time, forecast_type)`)
	if err != nil {
		t.Fatalf("apply migration 005 schema: %v", err)
	}
	if _, err := pool.Exec(ctx, "SELECT public.create_hypertable('forecast_data', 'forecast_time', if_not_exists => TRUE)"); err != nil {
		t.Fatalf("create hypertable: %v", err)
	}
	repo := &forecastRepository{pool: pool}
	now := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	oldFetch := now.Add(-time.Hour)
	oldTemp := float32(10)
	if err := repo.SaveBatch(ctx, []models.ForecastData{{ForecastTime: now.Add(time.Hour), Temperature: &oldTemp, ForecastType: "hourly", FetchedAt: oldFetch}}); err != nil {
		t.Fatal(err)
	}
	newTemp := float32(20)
	if err := repo.SaveBatch(ctx, []models.ForecastData{{ForecastTime: now.Add(time.Hour), Temperature: &newTemp, ForecastType: "hourly", FetchedAt: now}, {ForecastTime: now.Add(2 * time.Hour), ForecastType: "invalid", FetchedAt: now}}); err == nil {
		t.Fatal("SaveBatch succeeded with invalid type")
	}
	rolled, err := repo.GetHourlyForecast(ctx, now, now.Add(2*time.Hour))
	if err != nil || len(rolled) != 1 || rolled[0].Temperature == nil || *rolled[0].Temperature != oldTemp {
		t.Fatalf("rollback leaked: %#v, %v", rolled, err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE forecast_data"); err != nil {
		t.Fatal(err)
	}
	oldDaily := now.Truncate(24 * time.Hour)
	newDaily := oldDaily.Add(-3 * time.Hour)
	if err := repo.SaveBatch(ctx, []models.ForecastData{{ForecastTime: now.Add(time.Hour), ForecastType: "hourly", FetchedAt: oldFetch}, {ForecastTime: now.Add(48 * time.Hour), ForecastType: "hourly", FetchedAt: oldFetch}, {ForecastTime: oldDaily, ForecastType: "daily", FetchedAt: oldFetch}}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveBatch(ctx, []models.ForecastData{{ForecastTime: now.Add(time.Hour), ForecastType: "hourly", FetchedAt: now}, {ForecastTime: now.Add(2 * time.Hour), ForecastType: "hourly", FetchedAt: now}, {ForecastTime: newDaily, ForecastType: "daily", FetchedAt: now}}); err != nil {
		t.Fatal(err)
	}
	hourly, err := repo.GetHourlyForecast(ctx, now, now.Add(72*time.Hour))
	if err != nil || len(hourly) != 2 {
		t.Fatalf("hourly generation = %d, %v", len(hourly), err)
	}
	daily, err := repo.GetDailyForecast(ctx, newDaily.Add(-time.Hour), oldDaily.Add(time.Hour))
	if err != nil || len(daily) != 1 || !daily[0].ForecastTime.Equal(newDaily) {
		t.Fatalf("daily generation = %#v, %v", daily, err)
	}
	var before int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM forecast_data").Scan(&before); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `CREATE TEMP TABLE forecast_snapshot AS SELECT id, forecast_time, temperature, temperature_min, temperature_max, feels_like, precipitation_probability, precipitation, wind_speed, wind_direction, wind_gusts, cloud_cover, pressure, humidity, uv_index, weather_code, weather_description, forecast_type, fetched_at FROM forecast_data`)
	if err == nil {
		_, err = tx.Exec(ctx, "DELETE FROM forecast_data")
	}
	if err == nil {
		_, err = tx.Exec(ctx, `INSERT INTO forecast_data (id, forecast_time, temperature, temperature_min, temperature_max, feels_like, precipitation_probability, precipitation, wind_speed, wind_direction, wind_gusts, cloud_cover, pressure, humidity, uv_index, weather_code, weather_description, forecast_type, fetched_at) SELECT id, forecast_time, temperature, temperature_min, temperature_max, feels_like, precipitation_probability, precipitation, wind_speed, wind_direction, wind_gusts, cloud_cover, pressure, humidity, uv_index, weather_code, weather_description, forecast_type, fetched_at FROM forecast_snapshot`)
	}
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var after int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM forecast_data").Scan(&after); err != nil || after != before {
		t.Fatalf("snapshot restore count %d want %d: %v", after, before, err)
	}
}
