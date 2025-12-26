package apiclient

import (
	"context"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

// WeatherService is a service that uses API client instead of database
type WeatherService struct {
	client *Client
}

// NewWeatherService creates a new weather service using API client
func NewWeatherService(apiURL string) *WeatherService {
	return &WeatherService{
		client: NewClient(apiURL),
	}
}

// GetCurrent returns current weather data
func (s *WeatherService) GetCurrent(ctx context.Context) (*models.WeatherData, error) {
	return s.client.GetCurrent(ctx)
}

// GetCurrentWithHourlyChange returns current data, data from 1 hour ago, and daily min/max
func (s *WeatherService) GetCurrentWithHourlyChange(ctx context.Context) (current *models.WeatherData, hourAgo *models.WeatherData, dailyMinMax *repository.DailyMinMax, err error) {
	return s.client.GetCurrentWithHourlyChange(ctx)
}

// GetHistory returns historical weather data
func (s *WeatherService) GetHistory(ctx context.Context, from, to time.Time, interval string) ([]models.WeatherData, error) {
	return s.client.GetHistory(ctx, from, to, interval)
}

// GetRecentEvents returns recent weather events
func (s *WeatherService) GetRecentEvents(ctx context.Context, hours int) ([]models.WeatherEvent, error) {
	return s.client.GetRecentEvents(ctx, hours)
}
