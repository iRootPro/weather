package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

// Client is an HTTP client for the weather API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetCurrent fetches current weather data
func (c *Client) GetCurrent(ctx context.Context) (*models.WeatherData, error) {
	url := fmt.Sprintf("%s/api/weather/current", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current weather: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var data models.WeatherData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &data, nil
}

// GetCurrentWithHourlyChange fetches current weather with hourly comparison
func (c *Client) GetCurrentWithHourlyChange(ctx context.Context) (*models.WeatherData, *models.WeatherData, *repository.DailyMinMax, error) {
	// Get current weather
	current, err := c.GetCurrent(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	// Get data from 1 hour ago
	hourAgoTime := time.Now().Add(-1 * time.Hour)
	hourAgo, _ := c.GetDataNearTime(ctx, hourAgoTime)

	// Get daily min/max
	dailyMinMax, _ := c.GetDailyMinMax(ctx)

	return current, hourAgo, dailyMinMax, nil
}

// GetDataNearTime fetches weather data closest to the specified time
func (c *Client) GetDataNearTime(ctx context.Context, targetTime time.Time) (*models.WeatherData, error) {
	// For now, we don't have a specific endpoint for this
	// We'll fetch history and find the closest point
	from := targetTime.Add(-30 * time.Minute)
	to := targetTime.Add(30 * time.Minute)

	history, err := c.GetHistory(ctx, from, to, "raw")
	if err != nil || len(history) == 0 {
		return nil, err
	}

	// Find closest data point
	closest := &history[0]
	minDiff := targetTime.Sub(history[0].Time).Abs()

	for i := range history {
		diff := targetTime.Sub(history[i].Time).Abs()
		if diff < minDiff {
			minDiff = diff
			closest = &history[i]
		}
	}

	return closest, nil
}

// GetDailyMinMax fetches daily min/max values
func (c *Client) GetDailyMinMax(ctx context.Context) (*repository.DailyMinMax, error) {
	// Get stats for today
	stats, err := c.GetStats(ctx, "day")
	if err != nil {
		return nil, err
	}

	dailyMinMax := &repository.DailyMinMax{}
	if stats.TempOutdoorMin != nil {
		dailyMinMax.TempMin = stats.TempOutdoorMin
	}
	if stats.TempOutdoorMax != nil {
		dailyMinMax.TempMax = stats.TempOutdoorMax
	}
	if stats.HumidityOutdoorMin != nil {
		dailyMinMax.HumidityMin = stats.HumidityOutdoorMin
	}
	if stats.HumidityOutdoorMax != nil {
		dailyMinMax.HumidityMax = stats.HumidityOutdoorMax
	}
	if stats.PressureRelativeMin != nil {
		dailyMinMax.PressureMin = stats.PressureRelativeMin
	}
	if stats.PressureRelativeMax != nil {
		dailyMinMax.PressureMax = stats.PressureRelativeMax
	}
	if stats.WindSpeedMax != nil {
		dailyMinMax.WindMax = stats.WindSpeedMax
	}
	if stats.WindGustMax != nil {
		dailyMinMax.GustMax = stats.WindGustMax
	}

	return dailyMinMax, nil
}

// GetHistory fetches historical weather data
func (c *Client) GetHistory(ctx context.Context, from, to time.Time, interval string) ([]models.WeatherData, error) {
	url := fmt.Sprintf("%s/api/weather/history?from=%s&to=%s&interval=%s",
		c.baseURL,
		from.Format(time.RFC3339),
		to.Format(time.RFC3339),
		interval,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var data []models.WeatherData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return data, nil
}

// GetStats fetches weather statistics
func (c *Client) GetStats(ctx context.Context, period string) (*models.WeatherStats, error) {
	url := fmt.Sprintf("%s/api/weather/stats?period=%s", c.baseURL, period)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var stats models.WeatherStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &stats, nil
}

// GetRecentEvents fetches recent weather events
func (c *Client) GetRecentEvents(ctx context.Context, hours int) ([]models.WeatherEvent, error) {
	url := fmt.Sprintf("%s/api/weather/events?hours=%d", c.baseURL, hours)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var events []models.WeatherEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return events, nil
}
