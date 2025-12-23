package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iRootPro/weather/internal/models"
)

type weatherRepository struct {
	pool *pgxpool.Pool
}

func NewWeatherRepository(pool *pgxpool.Pool) WeatherRepository {
	return &weatherRepository{pool: pool}
}

func (r *weatherRepository) Save(ctx context.Context, data *models.WeatherData) error {
	query := `
		INSERT INTO weather_data (
			time, temp_outdoor, temp_indoor,
			humidity_outdoor, humidity_indoor,
			pressure_relative, pressure_absolute,
			wind_speed, wind_gust, wind_direction,
			rain_rate, rain_daily, rain_weekly, rain_monthly, rain_yearly,
			uv_index, solar_radiation,
			temp_feels_like, dew_point,
			raw_data
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)`

	_, err := r.pool.Exec(ctx, query,
		data.Time, data.TempOutdoor, data.TempIndoor,
		data.HumidityOutdoor, data.HumidityIndoor,
		data.PressureRelative, data.PressureAbsolute,
		data.WindSpeed, data.WindGust, data.WindDirection,
		data.RainRate, data.RainDaily, data.RainWeekly, data.RainMonthly, data.RainYearly,
		data.UVIndex, data.SolarRadiation,
		data.TempFeelsLike, data.DewPoint,
		data.RawData,
	)
	if err != nil {
		return fmt.Errorf("failed to insert weather data: %w", err)
	}

	return nil
}

func (r *weatherRepository) GetLatest(ctx context.Context) (*models.WeatherData, error) {
	query := `
		SELECT time, temp_outdoor, temp_indoor,
			humidity_outdoor, humidity_indoor,
			pressure_relative, pressure_absolute,
			wind_speed, wind_gust, wind_direction,
			rain_rate, rain_daily, rain_weekly, rain_monthly, rain_yearly,
			uv_index, solar_radiation,
			temp_feels_like, dew_point,
			raw_data
		FROM weather_data
		ORDER BY time DESC
		LIMIT 1`

	var data models.WeatherData
	err := r.pool.QueryRow(ctx, query).Scan(
		&data.Time, &data.TempOutdoor, &data.TempIndoor,
		&data.HumidityOutdoor, &data.HumidityIndoor,
		&data.PressureRelative, &data.PressureAbsolute,
		&data.WindSpeed, &data.WindGust, &data.WindDirection,
		&data.RainRate, &data.RainDaily, &data.RainWeekly, &data.RainMonthly, &data.RainYearly,
		&data.UVIndex, &data.SolarRadiation,
		&data.TempFeelsLike, &data.DewPoint,
		&data.RawData,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest weather data: %w", err)
	}

	return &data, nil
}

func (r *weatherRepository) GetByTimeRange(ctx context.Context, from, to time.Time) ([]models.WeatherData, error) {
	query := `
		SELECT time, temp_outdoor, temp_indoor,
			humidity_outdoor, humidity_indoor,
			pressure_relative, pressure_absolute,
			wind_speed, wind_gust, wind_direction,
			rain_rate, rain_daily, rain_weekly, rain_monthly, rain_yearly,
			uv_index, solar_radiation,
			temp_feels_like, dew_point,
			raw_data
		FROM weather_data
		WHERE time >= $1 AND time <= $2
		ORDER BY time DESC`

	rows, err := r.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query weather data: %w", err)
	}
	defer rows.Close()

	var result []models.WeatherData
	for rows.Next() {
		var data models.WeatherData
		err := rows.Scan(
			&data.Time, &data.TempOutdoor, &data.TempIndoor,
			&data.HumidityOutdoor, &data.HumidityIndoor,
			&data.PressureRelative, &data.PressureAbsolute,
			&data.WindSpeed, &data.WindGust, &data.WindDirection,
			&data.RainRate, &data.RainDaily, &data.RainWeekly, &data.RainMonthly, &data.RainYearly,
			&data.UVIndex, &data.SolarRadiation,
			&data.TempFeelsLike, &data.DewPoint,
			&data.RawData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan weather data: %w", err)
		}
		result = append(result, data)
	}

	return result, nil
}

func (r *weatherRepository) GetStats(ctx context.Context, from, to time.Time) (*models.WeatherStats, error) {
	query := `
		SELECT
			MIN(temp_outdoor), MAX(temp_outdoor), AVG(temp_outdoor),
			MIN(humidity_outdoor), MAX(humidity_outdoor), AVG(humidity_outdoor)::smallint,
			MIN(pressure_relative), MAX(pressure_relative), AVG(pressure_relative),
			MAX(wind_speed), MAX(wind_gust),
			SUM(rain_rate)
		FROM weather_data
		WHERE time >= $1 AND time <= $2`

	stats := &models.WeatherStats{
		Period:    "custom",
		StartTime: from,
		EndTime:   to,
	}

	err := r.pool.QueryRow(ctx, query, from, to).Scan(
		&stats.TempOutdoorMin, &stats.TempOutdoorMax, &stats.TempOutdoorAvg,
		&stats.HumidityOutdoorMin, &stats.HumidityOutdoorMax, &stats.HumidityOutdoorAvg,
		&stats.PressureRelativeMin, &stats.PressureRelativeMax, &stats.PressureRelativeAvg,
		&stats.WindSpeedMax, &stats.WindGustMax,
		&stats.RainTotal,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather stats: %w", err)
	}

	return stats, nil
}
