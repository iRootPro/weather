package openmeteo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	APIBaseURL = "https://api.open-meteo.com/v1/forecast"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

type ForecastRequest struct {
	Latitude  float64
	Longitude float64
	Hourly    []string // параметры для почасового прогноза
	Daily     []string // параметры для дневного прогноза
	Timezone  string
}

type ForecastResponse struct {
	Latitude  float64      `json:"latitude"`
	Longitude float64      `json:"longitude"`
	Timezone  string       `json:"timezone"`
	Hourly    HourlyData   `json:"hourly"`
	Daily     DailyData    `json:"daily"`
}

type HourlyData struct {
	Time                     []string  `json:"time"`
	Temperature              []float64 `json:"temperature_2m"`
	FeelsLike                []float64 `json:"apparent_temperature"`
	PrecipitationProbability []int     `json:"precipitation_probability"`
	Precipitation            []float64 `json:"precipitation"`
	WindSpeed                []float64 `json:"wind_speed_10m"`
	WindDirection            []int     `json:"wind_direction_10m"`
	WindGusts                []float64 `json:"wind_gusts_10m"`
	CloudCover               []int     `json:"cloud_cover"`
	Pressure                 []float64 `json:"pressure_msl"`
	Humidity                 []int     `json:"relative_humidity_2m"`
	UVIndex                  []float64 `json:"uv_index"`
	WeatherCode              []int     `json:"weather_code"`
}

type DailyData struct {
	Time                     []string  `json:"time"`
	TemperatureMin           []float64 `json:"temperature_2m_min"`
	TemperatureMax           []float64 `json:"temperature_2m_max"`
	PrecipitationProbability []int     `json:"precipitation_probability_max"`
	PrecipitationSum         []float64 `json:"precipitation_sum"`
	WindSpeedMax             []float64 `json:"wind_speed_10m_max"`
	WindDirection            []int     `json:"wind_direction_10m_dominant"`
	WindGustsMax             []float64 `json:"wind_gusts_10m_max"`
	UVIndexMax               []float64 `json:"uv_index_max"`
	WeatherCode              []int     `json:"weather_code"`
}

func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: APIBaseURL,
	}
}

func (c *Client) GetForecast(ctx context.Context, req ForecastRequest) (*ForecastResponse, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	q := u.Query()
	q.Set("latitude", fmt.Sprintf("%.6f", req.Latitude))
	q.Set("longitude", fmt.Sprintf("%.6f", req.Longitude))
	q.Set("timezone", req.Timezone)

	if len(req.Hourly) > 0 {
		for _, param := range req.Hourly {
			q.Add("hourly", param)
		}
	}

	if len(req.Daily) > 0 {
		for _, param := range req.Daily {
			q.Add("daily", param)
		}
	}

	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var forecastResp ForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecastResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &forecastResp, nil
}

// GetDefaultHourlyParams возвращает стандартный набор параметров для почасового прогноза
func GetDefaultHourlyParams() []string {
	return []string{
		"temperature_2m",
		"apparent_temperature",
		"precipitation_probability",
		"precipitation",
		"wind_speed_10m",
		"wind_direction_10m",
		"wind_gusts_10m",
		"cloud_cover",
		"pressure_msl",
		"relative_humidity_2m",
		"uv_index",
		"weather_code",
	}
}

// GetDefaultDailyParams возвращает стандартный набор параметров для дневного прогноза
func GetDefaultDailyParams() []string {
	return []string{
		"temperature_2m_min",
		"temperature_2m_max",
		"precipitation_probability_max",
		"precipitation_sum",
		"wind_speed_10m_max",
		"wind_direction_10m_dominant",
		"wind_gusts_10m_max",
		"uv_index_max",
		"weather_code",
	}
}
