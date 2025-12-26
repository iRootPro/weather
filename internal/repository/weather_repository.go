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
			wh65batt, ws90cap_volt,
			raw_data
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		)`

	_, err := r.pool.Exec(ctx, query,
		data.Time, data.TempOutdoor, data.TempIndoor,
		data.HumidityOutdoor, data.HumidityIndoor,
		data.PressureRelative, data.PressureAbsolute,
		data.WindSpeed, data.WindGust, data.WindDirection,
		data.RainRate, data.RainDaily, data.RainWeekly, data.RainMonthly, data.RainYearly,
		data.UVIndex, data.SolarRadiation,
		data.TempFeelsLike, data.DewPoint,
		data.WH65Batt, data.WS90CapVolt,
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
			wh65batt, ws90cap_volt,
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
		&data.WH65Batt, &data.WS90CapVolt,
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
			wh65batt, ws90cap_volt,
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
			&data.WH65Batt, &data.WS90CapVolt,
			&data.RawData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan weather data: %w", err)
		}
		result = append(result, data)
	}

	return result, nil
}

func (r *weatherRepository) GetAggregated(ctx context.Context, from, to time.Time, interval string) ([]models.WeatherData, error) {
	// Преобразуем интервал в формат PostgreSQL
	pgInterval := intervalToPostgres(interval)

	query := fmt.Sprintf(`
		SELECT
			time_bucket('%s', time) AS bucket,
			AVG(temp_outdoor) as temp_outdoor,
			AVG(temp_indoor) as temp_indoor,
			AVG(humidity_outdoor)::smallint as humidity_outdoor,
			AVG(humidity_indoor)::smallint as humidity_indoor,
			AVG(pressure_relative) as pressure_relative,
			AVG(pressure_absolute) as pressure_absolute,
			AVG(wind_speed) as wind_speed,
			MAX(wind_gust) as wind_gust,
			AVG(wind_direction)::smallint as wind_direction,
			AVG(rain_rate) as rain_rate,
			MAX(rain_daily) as rain_daily,
			MAX(rain_weekly) as rain_weekly,
			MAX(rain_monthly) as rain_monthly,
			MAX(rain_yearly) as rain_yearly,
			AVG(uv_index) as uv_index,
			AVG(solar_radiation) as solar_radiation,
			AVG(temp_feels_like) as temp_feels_like,
			AVG(dew_point) as dew_point
		FROM weather_data
		WHERE time >= $1 AND time <= $2
		GROUP BY bucket
		ORDER BY bucket ASC`, pgInterval)

	rows, err := r.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregated weather data: %w", err)
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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan aggregated weather data: %w", err)
		}
		result = append(result, data)
	}

	return result, nil
}

func intervalToPostgres(interval string) string {
	switch interval {
	case "5m":
		return "5 minutes"
	case "15m":
		return "15 minutes"
	case "1h":
		return "1 hour"
	case "1d":
		return "1 day"
	case "1w":
		return "1 week"
	case "1M":
		return "1 month"
	default:
		return "1 hour"
	}
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

// DailyMinMax contains min/max values for the current day
type DailyMinMax struct {
	TempMin     *float32
	TempMax     *float32
	HumidityMin *int16
	HumidityMax *int16
	PressureMin *float32
	PressureMax *float32
	WindMax     *float32
	GustMax     *float32
}

func (r *weatherRepository) GetDailyMinMax(ctx context.Context) (*DailyMinMax, error) {
	// Начало текущих суток (00:00 по локальному времени)
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	query := `
		SELECT
			MIN(temp_outdoor), MAX(temp_outdoor),
			MIN(humidity_outdoor), MAX(humidity_outdoor),
			MIN(pressure_relative), MAX(pressure_relative),
			MAX(wind_speed), MAX(wind_gust)
		FROM weather_data
		WHERE time >= $1`

	result := &DailyMinMax{}
	err := r.pool.QueryRow(ctx, query, startOfDay).Scan(
		&result.TempMin, &result.TempMax,
		&result.HumidityMin, &result.HumidityMax,
		&result.PressureMin, &result.PressureMax,
		&result.WindMax, &result.GustMax,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily min/max: %w", err)
	}

	return result, nil
}

func (r *weatherRepository) GetDataNearTime(ctx context.Context, targetTime time.Time) (*models.WeatherData, error) {
	// Ищем ближайшую запись к указанному времени (в пределах 10 минут)
	query := `
		SELECT time, temp_outdoor, temp_indoor,
			humidity_outdoor, humidity_indoor,
			pressure_relative, pressure_absolute,
			wind_speed, wind_gust, wind_direction,
			rain_rate, rain_daily, rain_weekly, rain_monthly, rain_yearly,
			uv_index, solar_radiation,
			temp_feels_like, dew_point,
			wh65batt, ws90cap_volt,
			raw_data
		FROM weather_data
		WHERE time BETWEEN $1 AND $2
		ORDER BY ABS(EXTRACT(EPOCH FROM (time - $3)))
		LIMIT 1`

	// Ищем в окне ±10 минут от целевого времени
	from := targetTime.Add(-10 * time.Minute)
	to := targetTime.Add(10 * time.Minute)

	var data models.WeatherData
	err := r.pool.QueryRow(ctx, query, from, to, targetTime).Scan(
		&data.Time, &data.TempOutdoor, &data.TempIndoor,
		&data.HumidityOutdoor, &data.HumidityIndoor,
		&data.PressureRelative, &data.PressureAbsolute,
		&data.WindSpeed, &data.WindGust, &data.WindDirection,
		&data.RainRate, &data.RainDaily, &data.RainWeekly, &data.RainMonthly, &data.RainYearly,
		&data.UVIndex, &data.SolarRadiation,
		&data.TempFeelsLike, &data.DewPoint,
		&data.WH65Batt, &data.WS90CapVolt,
		&data.RawData,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather data near time: %w", err)
	}

	return &data, nil
}

func (r *weatherRepository) GetRecords(ctx context.Context) (*models.WeatherRecords, error) {
	records := &models.WeatherRecords{}

	// Получаем диапазон данных
	rangeQuery := `SELECT MIN(time), MAX(time) FROM weather_data`
	err := r.pool.QueryRow(ctx, rangeQuery).Scan(&records.FirstRecord, &records.LastRecord)
	if err != nil {
		return nil, fmt.Errorf("failed to get data range: %w", err)
	}
	records.TotalDays = int(records.LastRecord.Sub(records.FirstRecord).Hours() / 24)

	// Минимальная температура
	err = r.pool.QueryRow(ctx, `
		SELECT temp_outdoor, time FROM weather_data
		WHERE temp_outdoor IS NOT NULL
		ORDER BY temp_outdoor ASC LIMIT 1
	`).Scan(&records.TempOutdoorMin.Value, &records.TempOutdoorMin.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to get min temp: %w", err)
	}

	// Максимальная температура
	err = r.pool.QueryRow(ctx, `
		SELECT temp_outdoor, time FROM weather_data
		WHERE temp_outdoor IS NOT NULL
		ORDER BY temp_outdoor DESC LIMIT 1
	`).Scan(&records.TempOutdoorMax.Value, &records.TempOutdoorMax.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to get max temp: %w", err)
	}

	// Минимальная влажность
	err = r.pool.QueryRow(ctx, `
		SELECT humidity_outdoor, time FROM weather_data
		WHERE humidity_outdoor IS NOT NULL
		ORDER BY humidity_outdoor ASC LIMIT 1
	`).Scan(&records.HumidityOutdoorMin.Value, &records.HumidityOutdoorMin.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to get min humidity: %w", err)
	}

	// Максимальная влажность
	err = r.pool.QueryRow(ctx, `
		SELECT humidity_outdoor, time FROM weather_data
		WHERE humidity_outdoor IS NOT NULL
		ORDER BY humidity_outdoor DESC LIMIT 1
	`).Scan(&records.HumidityOutdoorMax.Value, &records.HumidityOutdoorMax.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to get max humidity: %w", err)
	}

	// Минимальное давление
	err = r.pool.QueryRow(ctx, `
		SELECT pressure_relative, time FROM weather_data
		WHERE pressure_relative IS NOT NULL
		ORDER BY pressure_relative ASC LIMIT 1
	`).Scan(&records.PressureMin.Value, &records.PressureMin.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to get min pressure: %w", err)
	}

	// Максимальное давление
	err = r.pool.QueryRow(ctx, `
		SELECT pressure_relative, time FROM weather_data
		WHERE pressure_relative IS NOT NULL
		ORDER BY pressure_relative DESC LIMIT 1
	`).Scan(&records.PressureMax.Value, &records.PressureMax.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to get max pressure: %w", err)
	}

	// Максимальная скорость ветра
	err = r.pool.QueryRow(ctx, `
		SELECT wind_speed, time FROM weather_data
		WHERE wind_speed IS NOT NULL
		ORDER BY wind_speed DESC LIMIT 1
	`).Scan(&records.WindSpeedMax.Value, &records.WindSpeedMax.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to get max wind speed: %w", err)
	}

	// Максимальные порывы ветра
	err = r.pool.QueryRow(ctx, `
		SELECT wind_gust, time FROM weather_data
		WHERE wind_gust IS NOT NULL
		ORDER BY wind_gust DESC LIMIT 1
	`).Scan(&records.WindGustMax.Value, &records.WindGustMax.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to get max wind gust: %w", err)
	}

	// Максимальные осадки за день
	err = r.pool.QueryRow(ctx, `
		SELECT rain_daily, time FROM weather_data
		WHERE rain_daily IS NOT NULL
		ORDER BY rain_daily DESC LIMIT 1
	`).Scan(&records.RainDailyMax.Value, &records.RainDailyMax.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to get max rain daily: %w", err)
	}

	// Максимальная солнечная радиация
	err = r.pool.QueryRow(ctx, `
		SELECT solar_radiation, time FROM weather_data
		WHERE solar_radiation IS NOT NULL
		ORDER BY solar_radiation DESC LIMIT 1
	`).Scan(&records.SolarRadiationMax.Value, &records.SolarRadiationMax.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to get max solar radiation: %w", err)
	}

	// Максимальный UV индекс
	err = r.pool.QueryRow(ctx, `
		SELECT uv_index, time FROM weather_data
		WHERE uv_index IS NOT NULL
		ORDER BY uv_index DESC LIMIT 1
	`).Scan(&records.UVIndexMax.Value, &records.UVIndexMax.Time)
	if err != nil {
		return nil, fmt.Errorf("failed to get max uv index: %w", err)
	}

	return records, nil
}

// GetDataForEventDetection returns weather data for event detection with 5-minute intervals
func (r *weatherRepository) GetDataForEventDetection(ctx context.Context, from, to time.Time) ([]models.WeatherData, error) {
	query := `
		SELECT
			time_bucket('5 minutes', time) AS bucket,
			AVG(temp_outdoor) as temp_outdoor,
			AVG(humidity_outdoor)::smallint as humidity_outdoor,
			AVG(pressure_relative) as pressure_relative,
			AVG(wind_speed) as wind_speed,
			MAX(wind_gust) as wind_gust,
			AVG(wind_direction)::smallint as wind_direction,
			AVG(rain_rate) as rain_rate,
			MAX(rain_daily) as rain_daily
		FROM weather_data
		WHERE time >= $1 AND time <= $2
		GROUP BY bucket
		ORDER BY bucket ASC`

	rows, err := r.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query weather data for events: %w", err)
	}
	defer rows.Close()

	var result []models.WeatherData
	for rows.Next() {
		var data models.WeatherData
		err := rows.Scan(
			&data.Time,
			&data.TempOutdoor,
			&data.HumidityOutdoor,
			&data.PressureRelative,
			&data.WindSpeed,
			&data.WindGust,
			&data.WindDirection,
			&data.RainRate,
			&data.RainDaily,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan weather data for events: %w", err)
		}
		result = append(result, data)
	}

	return result, nil
}
