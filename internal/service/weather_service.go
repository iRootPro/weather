package service

import (
	"context"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

type WeatherService struct {
	repo repository.WeatherRepository
}

func NewWeatherService(repo repository.WeatherRepository) *WeatherService {
	return &WeatherService{repo: repo}
}

func (s *WeatherService) GetCurrent(ctx context.Context) (*models.WeatherData, error) {
	return s.repo.GetLatest(ctx)
}

func (s *WeatherService) GetHistory(ctx context.Context, from, to time.Time, interval string) ([]models.WeatherData, error) {
	if interval == "" || interval == "raw" {
		return s.repo.GetByTimeRange(ctx, from, to)
	}
	return s.repo.GetAggregated(ctx, from, to, interval)
}

func (s *WeatherService) GetStats(ctx context.Context, period string) (*models.WeatherStats, error) {
	now := time.Now()
	var from time.Time

	switch period {
	case "day":
		from = now.AddDate(0, 0, -1)
	case "week":
		from = now.AddDate(0, 0, -7)
	case "month":
		from = now.AddDate(0, -1, 0)
	case "year":
		from = now.AddDate(-1, 0, 0)
	default:
		from = now.AddDate(0, 0, -1) // по умолчанию день
	}

	stats, err := s.repo.GetStats(ctx, from, now)
	if err != nil {
		return nil, err
	}
	stats.Period = period
	return stats, nil
}

func (s *WeatherService) GetChartData(ctx context.Context, from, to time.Time, interval string, fields []string) (*models.ChartData, error) {
	data, err := s.repo.GetAggregated(ctx, from, to, interval)
	if err != nil {
		return nil, err
	}

	chart := &models.ChartData{
		Labels:   make([]string, len(data)),
		Datasets: make(map[string][]float64),
	}

	// Инициализируем datasets для запрошенных полей
	for _, field := range fields {
		chart.Datasets[field] = make([]float64, len(data))
	}

	for i, d := range data {
		chart.Labels[i] = d.Time.Format("2006-01-02 15:04")

		for _, field := range fields {
			var val float64
			switch field {
			case "temp_outdoor":
				if d.TempOutdoor != nil {
					val = float64(*d.TempOutdoor)
				}
			case "temp_indoor":
				if d.TempIndoor != nil {
					val = float64(*d.TempIndoor)
				}
			case "humidity_outdoor":
				if d.HumidityOutdoor != nil {
					val = float64(*d.HumidityOutdoor)
				}
			case "humidity_indoor":
				if d.HumidityIndoor != nil {
					val = float64(*d.HumidityIndoor)
				}
			case "pressure_relative":
				if d.PressureRelative != nil {
					val = float64(*d.PressureRelative)
				}
			case "wind_speed":
				if d.WindSpeed != nil {
					val = float64(*d.WindSpeed)
				}
			case "wind_gust":
				if d.WindGust != nil {
					val = float64(*d.WindGust)
				}
			case "rain_daily":
				if d.RainDaily != nil {
					val = float64(*d.RainDaily)
				}
			case "uv_index":
				if d.UVIndex != nil {
					val = float64(*d.UVIndex)
				}
			case "solar_radiation":
				if d.SolarRadiation != nil {
					val = float64(*d.SolarRadiation)
				}
			}
			chart.Datasets[field][i] = val
		}
	}

	return chart, nil
}

func (s *WeatherService) GetRecords(ctx context.Context) (*models.WeatherRecords, error) {
	return s.repo.GetRecords(ctx)
}

// GetCurrentWithHourlyChange returns current data and data from 1 hour ago for comparison
func (s *WeatherService) GetCurrentWithHourlyChange(ctx context.Context) (current *models.WeatherData, hourAgo *models.WeatherData, err error) {
	current, err = s.repo.GetLatest(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Получаем данные за час назад (игнорируем ошибку - данных может не быть)
	targetTime := time.Now().Add(-1 * time.Hour)
	hourAgo, _ = s.repo.GetDataNearTime(ctx, targetTime)

	return current, hourAgo, nil
}
