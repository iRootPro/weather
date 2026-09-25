package service

import (
	"context"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

type ForecastService struct {
	repo     repository.ForecastRepository
	location *time.Location
}

func NewForecastService(repo repository.ForecastRepository) *ForecastService {
	return &ForecastService{repo: repo, location: time.Local}
}

// SetTimezone aligns forecast query bounds with Open-Meteo's requested local time.
func (s *ForecastService) SetTimezone(timezone string) {
	if location, err := time.LoadLocation(timezone); err == nil {
		s.location = location
	}
}

// Now returns the current instant in the location used for forecast presentation.
func (s *ForecastService) Now() time.Time {
	return time.Now().In(s.location)
}

// GetTodayForecast возвращает почасовой прогноз на сегодня
func (s *ForecastService) GetTodayForecast(ctx context.Context) ([]models.HourlyForecast, error) {
	now := time.Now().In(s.location)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	data, err := s.repo.GetHourlyForecast(ctx, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	return convertToHourlyForecast(data, s.location), nil
}

// GetHourlyForecast возвращает почасовой прогноз на N часов вперед
func (s *ForecastService) GetHourlyForecast(ctx context.Context, hours int) ([]models.HourlyForecast, error) {
	now := time.Now().In(s.location)
	to := now.Add(time.Duration(hours) * time.Hour)

	data, err := s.repo.GetHourlyForecast(ctx, now, to)
	if err != nil {
		return nil, err
	}

	return convertToHourlyForecast(data, s.location), nil
}

// GetDailyForecast возвращает дневной прогноз на N дней вперед
func (s *ForecastService) GetDailyForecast(ctx context.Context, days int) ([]models.DailyForecast, error) {
	now := time.Now().In(s.location)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location)
	data, err := s.repo.GetDailyForecast(ctx, startOfDay, startOfDay.AddDate(0, 0, days))
	if err != nil {
		return nil, err
	}

	return convertToDailyForecast(data, s.location), nil
}

// GetCurrentConditions возвращает текущие условия из прогноза (первый час)
func (s *ForecastService) GetCurrentConditions(ctx context.Context) (*models.HourlyForecast, error) {
	now := s.Now()
	data, err := s.repo.GetHourlyForecast(ctx, now, now.Add(time.Hour))
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	forecasts := convertToHourlyForecast(data, s.location)
	return &forecasts[0], nil
}

// GetShortForecast возвращает краткий прогноз (следующие 3-6 часов) для утренней сводки
func (s *ForecastService) GetShortForecast(ctx context.Context) ([]models.HourlyForecast, error) {
	return s.GetHourlyForecast(ctx, 12) // следующие 12 часов
}

func convertToHourlyForecast(data []models.ForecastData, location *time.Location) []models.HourlyForecast {
	result := make([]models.HourlyForecast, 0, len(data))
	for _, d := range data {
		forecast := models.HourlyForecast{
			Time:      d.ForecastTime.In(location),
			FetchedAt: d.FetchedAt.In(location),
		}

		if d.Temperature != nil {
			forecast.Temperature = *d.Temperature
			forecast.HasTemperature = true
		}
		if d.FeelsLike != nil {
			forecast.FeelsLike = *d.FeelsLike
			forecast.HasFeelsLike = true
		}
		if d.PrecipitationProbability != nil {
			forecast.PrecipitationProbability = *d.PrecipitationProbability
			forecast.HasPrecipitationProbability = true
		}
		if d.Precipitation != nil {
			forecast.Precipitation = *d.Precipitation
			forecast.HasPrecipitation = true
		}
		if d.WindSpeed != nil {
			forecast.WindSpeed = kilometersPerHourToMetersPerSecond(*d.WindSpeed)
			forecast.HasWindSpeed = true
		}
		if d.WindGusts != nil {
			forecast.WindGusts = kilometersPerHourToMetersPerSecond(*d.WindGusts)
			forecast.HasWindGusts = true
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

func convertToDailyForecast(data []models.ForecastData, location *time.Location) []models.DailyForecast {
	result := make([]models.DailyForecast, 0, len(data))
	for _, d := range data {
		forecast := models.DailyForecast{
			Date:      d.ForecastTime.In(location),
			FetchedAt: d.FetchedAt.In(location),
		}

		if d.TemperatureMin != nil {
			forecast.TemperatureMin = *d.TemperatureMin
			forecast.HasTemperatureMin = true
		}
		if d.TemperatureMax != nil {
			forecast.TemperatureMax = *d.TemperatureMax
			forecast.HasTemperatureMax = true
		}
		if d.PrecipitationProbability != nil {
			forecast.PrecipitationProbability = *d.PrecipitationProbability
			forecast.HasPrecipitationProbability = true
		}
		if d.Precipitation != nil {
			forecast.PrecipitationSum = *d.Precipitation
			forecast.HasPrecipitationSum = true
		}
		if d.WindSpeed != nil {
			forecast.WindSpeedMax = kilometersPerHourToMetersPerSecond(*d.WindSpeed)
			forecast.HasWindSpeedMax = true
		}
		if d.WindGusts != nil {
			forecast.WindGustsMax = kilometersPerHourToMetersPerSecond(*d.WindGusts)
			forecast.HasWindGustsMax = true
		}
		if d.UVIndex != nil {
			forecast.UVIndexMax = *d.UVIndex
			forecast.HasUVIndexMax = true
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

func kilometersPerHourToMetersPerSecond(value float32) float32 {
	return value / 3.6
}
