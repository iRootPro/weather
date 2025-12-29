package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iRootPro/weather/internal/models"
)

type forecastRepository struct {
	pool *pgxpool.Pool
}

func NewForecastRepository(pool *pgxpool.Pool) ForecastRepository {
	return &forecastRepository{pool: pool}
}

func (r *forecastRepository) SaveHourly(ctx context.Context, data *models.ForecastData) error {
	data.ForecastType = "hourly"
	return r.save(ctx, data)
}

func (r *forecastRepository) SaveDaily(ctx context.Context, data *models.ForecastData) error {
	data.ForecastType = "daily"
	return r.save(ctx, data)
}

func (r *forecastRepository) save(ctx context.Context, data *models.ForecastData) error {
	query := `
		INSERT INTO forecast_data (
			forecast_time, temperature, temperature_min, temperature_max, feels_like,
			precipitation_probability, precipitation,
			wind_speed, wind_direction, wind_gusts,
			cloud_cover, pressure, humidity, uv_index,
			weather_code, weather_description,
			forecast_type, fetched_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18
		)
		ON CONFLICT (forecast_time, forecast_type)
		DO UPDATE SET
			temperature = EXCLUDED.temperature,
			temperature_min = EXCLUDED.temperature_min,
			temperature_max = EXCLUDED.temperature_max,
			feels_like = EXCLUDED.feels_like,
			precipitation_probability = EXCLUDED.precipitation_probability,
			precipitation = EXCLUDED.precipitation,
			wind_speed = EXCLUDED.wind_speed,
			wind_direction = EXCLUDED.wind_direction,
			wind_gusts = EXCLUDED.wind_gusts,
			cloud_cover = EXCLUDED.cloud_cover,
			pressure = EXCLUDED.pressure,
			humidity = EXCLUDED.humidity,
			uv_index = EXCLUDED.uv_index,
			weather_code = EXCLUDED.weather_code,
			weather_description = EXCLUDED.weather_description,
			fetched_at = EXCLUDED.fetched_at`

	_, err := r.pool.Exec(ctx, query,
		data.ForecastTime, data.Temperature, data.TemperatureMin, data.TemperatureMax, data.FeelsLike,
		data.PrecipitationProbability, data.Precipitation,
		data.WindSpeed, data.WindDirection, data.WindGusts,
		data.CloudCover, data.Pressure, data.Humidity, data.UVIndex,
		data.WeatherCode, data.WeatherDescription,
		data.ForecastType, data.FetchedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert forecast data: %w", err)
	}

	return nil
}

func (r *forecastRepository) SaveBatch(ctx context.Context, data []models.ForecastData) error {
	if len(data) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO forecast_data (
			forecast_time, temperature, temperature_min, temperature_max, feels_like,
			precipitation_probability, precipitation,
			wind_speed, wind_direction, wind_gusts,
			cloud_cover, pressure, humidity, uv_index,
			weather_code, weather_description,
			forecast_type, fetched_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18
		)
		ON CONFLICT (forecast_time, forecast_type)
		DO UPDATE SET
			temperature = EXCLUDED.temperature,
			temperature_min = EXCLUDED.temperature_min,
			temperature_max = EXCLUDED.temperature_max,
			feels_like = EXCLUDED.feels_like,
			precipitation_probability = EXCLUDED.precipitation_probability,
			precipitation = EXCLUDED.precipitation,
			wind_speed = EXCLUDED.wind_speed,
			wind_direction = EXCLUDED.wind_direction,
			wind_gusts = EXCLUDED.wind_gusts,
			cloud_cover = EXCLUDED.cloud_cover,
			pressure = EXCLUDED.pressure,
			humidity = EXCLUDED.humidity,
			uv_index = EXCLUDED.uv_index,
			weather_code = EXCLUDED.weather_code,
			weather_description = EXCLUDED.weather_description,
			fetched_at = EXCLUDED.fetched_at`

	for _, d := range data {
		batch.Queue(query,
			d.ForecastTime, d.Temperature, d.TemperatureMin, d.TemperatureMax, d.FeelsLike,
			d.PrecipitationProbability, d.Precipitation,
			d.WindSpeed, d.WindDirection, d.WindGusts,
			d.CloudCover, d.Pressure, d.Humidity, d.UVIndex,
			d.WeatherCode, d.WeatherDescription,
			d.ForecastType, d.FetchedAt,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(data); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to execute batch item %d: %w", i, err)
		}
	}

	return nil
}

func (r *forecastRepository) GetHourlyForecast(ctx context.Context, from, to time.Time) ([]models.ForecastData, error) {
	query := `
		SELECT id, forecast_time, temperature, temperature_min, temperature_max, feels_like,
			precipitation_probability, precipitation,
			wind_speed, wind_direction, wind_gusts,
			cloud_cover, pressure, humidity, uv_index,
			weather_code, weather_description,
			forecast_type, fetched_at
		FROM forecast_data
		WHERE forecast_type = 'hourly'
			AND forecast_time >= $1
			AND forecast_time <= $2
		ORDER BY forecast_time ASC`

	return r.query(ctx, query, from, to)
}

func (r *forecastRepository) GetDailyForecast(ctx context.Context, from, to time.Time) ([]models.ForecastData, error) {
	query := `
		SELECT id, forecast_time, temperature, temperature_min, temperature_max, feels_like,
			precipitation_probability, precipitation,
			wind_speed, wind_direction, wind_gusts,
			cloud_cover, pressure, humidity, uv_index,
			weather_code, weather_description,
			forecast_type, fetched_at
		FROM forecast_data
		WHERE forecast_type = 'daily'
			AND forecast_time >= $1
			AND forecast_time <= $2
		ORDER BY forecast_time ASC`

	return r.query(ctx, query, from, to)
}

func (r *forecastRepository) GetLatestHourly(ctx context.Context, hours int) ([]models.ForecastData, error) {
	now := time.Now()
	to := now.Add(time.Duration(hours) * time.Hour)
	return r.GetHourlyForecast(ctx, now, to)
}

func (r *forecastRepository) GetLatestDaily(ctx context.Context, days int) ([]models.ForecastData, error) {
	now := time.Now().Truncate(24 * time.Hour) // начало дня
	to := now.Add(time.Duration(days) * 24 * time.Hour)
	return r.GetDailyForecast(ctx, now, to)
}

func (r *forecastRepository) DeleteOldForecasts(ctx context.Context, olderThan time.Time) error {
	query := `DELETE FROM forecast_data WHERE forecast_time < $1`

	_, err := r.pool.Exec(ctx, query, olderThan)
	if err != nil {
		return fmt.Errorf("failed to delete old forecasts: %w", err)
	}

	return nil
}

func (r *forecastRepository) query(ctx context.Context, query string, args ...interface{}) ([]models.ForecastData, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query forecast data: %w", err)
	}
	defer rows.Close()

	var result []models.ForecastData
	for rows.Next() {
		var data models.ForecastData
		err := rows.Scan(
			&data.ID, &data.ForecastTime, &data.Temperature, &data.TemperatureMin, &data.TemperatureMax, &data.FeelsLike,
			&data.PrecipitationProbability, &data.Precipitation,
			&data.WindSpeed, &data.WindDirection, &data.WindGusts,
			&data.CloudCover, &data.Pressure, &data.Humidity, &data.UVIndex,
			&data.WeatherCode, &data.WeatherDescription,
			&data.ForecastType, &data.FetchedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan forecast data: %w", err)
		}
		result = append(result, data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return result, nil
}
