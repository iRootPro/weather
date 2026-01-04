package ipgeolocation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	APIBaseURL = "https://api.ipgeolocation.io/astronomy"
)

type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

type AstronomyRequest struct {
	Latitude  float64
	Longitude float64
	Date      time.Time // optional, defaults to today
}

type AstronomyResponse struct {
	// Location is an object, we don't need it so we ignore it
	Date                string  `json:"date"`
	CurrentTime         string  `json:"current_time"`
	Sunrise             string  `json:"sunrise"`
	Sunset              string  `json:"sunset"`
	SunStatus           string  `json:"sun_status"`
	SolarNoon           string  `json:"solar_noon"`
	DayLength           string  `json:"day_length"`
	SunAltitude         float64 `json:"sun_altitude"`
	SunDistance         float64 `json:"sun_distance"`
	SunAzimuth          float64 `json:"sun_azimuth"`
	Moonrise            string  `json:"moonrise"`
	Moonset             string  `json:"moonset"`
	MoonStatus          string  `json:"moon_status"`
	MoonAltitude        float64 `json:"moon_altitude"`
	MoonDistance        float64 `json:"moon_distance"`
	MoonAzimuth         float64 `json:"moon_azimuth"`
	MoonParallacticAngle float64 `json:"moon_parallactic_angle"`
}

func NewClient(apiKey string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiKey:  apiKey,
		baseURL: APIBaseURL,
	}
}

func (c *Client) GetAstronomy(ctx context.Context, req AstronomyRequest) (*AstronomyResponse, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	q := u.Query()
	q.Set("apiKey", c.apiKey)
	q.Set("lat", fmt.Sprintf("%.6f", req.Latitude))
	q.Set("long", fmt.Sprintf("%.6f", req.Longitude))

	// If date is specified, format it
	if !req.Date.IsZero() {
		q.Set("date", req.Date.Format("2006-01-02"))
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

	var astronomyResp AstronomyResponse
	if err := json.NewDecoder(resp.Body).Decode(&astronomyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &astronomyResp, nil
}
