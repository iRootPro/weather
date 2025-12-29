package service

import (
	"context"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

type ForecastService struct {
	repo repository.ForecastRepository
}

func NewForecastService(repo repository.ForecastRepository) *ForecastService {
	return &ForecastService{repo: repo}
}

// GetTodayForecast возвращает почасовой прогноз на сегодня
func (s *ForecastService) GetTodayForecast(ctx context.Context) ([]models.HourlyForecast, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	data, err := s.repo.GetHourlyForecast(ctx, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	return convertToHourlyForecast(data), nil
}

// GetHourlyForecast возвращает почасовой прогноз на N часов вперед
func (s *ForecastService) GetHourlyForecast(ctx context.Context, hours int) ([]models.HourlyForecast, error) {
	now := time.Now()
	to := now.Add(time.Duration(hours) * time.Hour)

	data, err := s.repo.GetHourlyForecast(ctx, now, to)
	if err != nil {
		return nil, err
	}

	return convertToHourlyForecast(data), nil
}

// GetDailyForecast возвращает дневной прогноз на N дней вперед
func (s *ForecastService) GetDailyForecast(ctx context.Context, days int) ([]models.DailyForecast, error) {
	data, err := s.repo.GetLatestDaily(ctx, days)
	if err != nil {
		return nil, err
	}

	return convertToDailyForecast(data), nil
}

// GetCurrentConditions возвращает текущие условия из прогноза (первый час)
func (s *ForecastService) GetCurrentConditions(ctx context.Context) (*models.HourlyForecast, error) {
	data, err := s.repo.GetLatestHourly(ctx, 1)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	forecasts := convertToHourlyForecast(data)
	return &forecasts[0], nil
}

// GetShortForecast возвращает краткий прогноз (следующие 3-6 часов) для утренней сводки
func (s *ForecastService) GetShortForecast(ctx context.Context) ([]models.HourlyForecast, error) {
	return s.GetHourlyForecast(ctx, 12) // следующие 12 часов
}

func convertToHourlyForecast(data []models.ForecastData) []models.HourlyForecast {
	result := make([]models.HourlyForecast, 0, len(data))
	for _, d := range data {
		forecast := models.HourlyForecast{
			Time: d.ForecastTime,
		}

		if d.Temperature != nil {
			forecast.Temperature = *d.Temperature
		}
		if d.FeelsLike != nil {
			forecast.FeelsLike = *d.FeelsLike
		}
		if d.PrecipitationProbability != nil {
			forecast.PrecipitationProbability = *d.PrecipitationProbability
		}
		if d.Precipitation != nil {
			forecast.Precipitation = *d.Precipitation
		}
		if d.WindSpeed != nil {
			forecast.WindSpeed = *d.WindSpeed
		}
		if d.WindDirection != nil {
			forecast.WindDirection = *d.WindDirection
		}
		if d.WeatherCode != nil {
			forecast.WeatherCode = *d.WeatherCode
		}
		if d.WeatherDescription != nil {
			forecast.WeatherDescription = *d.WeatherDescription
		}

		forecast.Icon = models.GetWeatherIcon(forecast.WeatherCode)

		result = append(result, forecast)
	}
	return result
}

func convertToDailyForecast(data []models.ForecastData) []models.DailyForecast {
	result := make([]models.DailyForecast, 0, len(data))
	for _, d := range data {
		forecast := models.DailyForecast{
			Date: d.ForecastTime,
		}

		if d.TemperatureMin != nil {
			forecast.TemperatureMin = *d.TemperatureMin
		}
		if d.TemperatureMax != nil {
			forecast.TemperatureMax = *d.TemperatureMax
		}
		if d.PrecipitationProbability != nil {
			forecast.PrecipitationProbability = *d.PrecipitationProbability
		}
		if d.Precipitation != nil {
			forecast.PrecipitationSum = *d.Precipitation
		}
		if d.WindSpeed != nil {
			forecast.WindSpeedMax = *d.WindSpeed
		}
		if d.WindDirection != nil {
			forecast.WindDirection = *d.WindDirection
		}
		if d.WeatherCode != nil {
			forecast.WeatherCode = *d.WeatherCode
		}
		if d.WeatherDescription != nil {
			forecast.WeatherDescription = *d.WeatherDescription
		}

		forecast.Icon = models.GetWeatherIcon(forecast.WeatherCode)

		result = append(result, forecast)
	}
	return result
}
