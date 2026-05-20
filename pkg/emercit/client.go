// Package emercit implements a small client for pub.emercit.ru.
package emercit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://pub.emercit.ru"

type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
	access     string
	refresh    string
}

type Preset struct {
	DefaultUser struct {
		User     string `json:"user"`
		Password string `json:"password"`
	} `json:"default_user"`
}

type tokenResponse struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

type ActualResponse map[string]Station

type Station struct {
	Name             string                    `json:"name"`
	ShortName        string                    `json:"short_name"`
	Description      string                    `json:"description"`
	Lat              *float64                  `json:"lat"`
	Lon              *float64                  `json:"lon"`
	Area             string                    `json:"area"`
	District         string                    `json:"district"`
	City             *string                   `json:"city"`
	Locality         *string                   `json:"locality"`
	MonitoringObject string                    `json:"monitoring_object"`
	HolderUUID       string                    `json:"holder_uuid"`
	HolderName       string                    `json:"holder_name"`
	MCHs             map[string]map[string]MCH `json:"mchs"`
}

type MCH struct {
	Name     string      `json:"name"`
	State    MCHState    `json:"state"`
	Settings MCHSettings `json:"settings"`
}

type MCHState struct {
	StateTime        string     `json:"state_time"`
	StateCode        *int       `json:"state_code"`
	LevelCode        *int       `json:"level_code"`
	LevelCodeHighest *int       `json:"level_code_highest"`
	LastValue        WaterValue `json:"last_value"`
}

type WaterValue struct {
	Time    string   `json:"time"`
	BS      *float64 `json:"bs"`
	Zero    *float64 `json:"zero"`
	HDIIHR  *float64 `json:"hdi_ihr"`
	HDILead *string  `json:"hdi_lead"`
}

type MCHSettings struct {
	FixBS                *float64 `json:"fix_bs"`
	ZeroBS               *float64 `json:"zero_bs"`
	DryBS                *float64 `json:"dry_bs"`
	DrainingDangerBS     *float64 `json:"draining_danger_bs"`
	DrainingPreventionBS *float64 `json:"draining_prevention_bs"`
	FloodingPreventionBS *float64 `json:"flooding_prevention_bs"`
	FloodingDangerBS     *float64 `json:"flooding_danger_bs"`
}

type HistoryResponse map[string]HistorySeries

type HistorySeries struct {
	Values []HistoryValue `json:"values"`
}

type HistoryValue struct {
	Time string   `json:"time"`
	BS   *float64 `json:"bs"`
	Zero *float64 `json:"zero"`
}

func NewClient(timeout time.Duration, baseURL, username, password string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    strings.TrimRight(baseURL, "/"),
		username:   username,
		password:   password,
	}
}

func (c *Client) LoadPreset(ctx context.Context) (*Preset, error) {
	var out Preset
	if err := c.getJSON(ctx, "/preset.json", "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Login(ctx context.Context) error {
	if c.username == "" || c.password == "" {
		preset, err := c.LoadPreset(ctx)
		if err != nil {
			return fmt.Errorf("failed to load preset: %w", err)
		}
		c.username = preset.DefaultUser.User
		c.password = preset.DefaultUser.Password
	}
	if c.username == "" || c.password == "" {
		return fmt.Errorf("emercit credentials are empty")
	}

	payload, _ := json.Marshal(map[string]string{"username": c.username, "password": c.password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/token/", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "weather-fetcher/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("login failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return fmt.Errorf("failed to decode token: %w", err)
	}
	if token.Access == "" {
		return fmt.Errorf("empty access token")
	}
	c.access = token.Access
	c.refresh = token.Refresh
	return nil
}

func (c *Client) GetActual(ctx context.Context) (ActualResponse, error) {
	if c.access == "" {
		if err := c.Login(ctx); err != nil {
			return nil, err
		}
	}
	var out ActualResponse
	if err := c.getJSON(ctx, "/api/actual/", c.access, &out); err != nil {
		// JWT у публичного пользователя короткоживущий; один раз перелогиниваемся.
		if strings.Contains(err.Error(), "status 401") || strings.Contains(err.Error(), "status 403") {
			c.access = ""
			if loginErr := c.Login(ctx); loginErr != nil {
				return nil, loginErr
			}
			if retryErr := c.getJSON(ctx, "/api/actual/", c.access, &out); retryErr != nil {
				return nil, retryErr
			}
			return out, nil
		}
		return nil, err
	}
	return out, nil
}

func (c *Client) GetWaterLevelHistory(ctx context.Context, waterLevelUUID string, from, to time.Time) (HistoryResponse, error) {
	if c.access == "" {
		if err := c.Login(ctx); err != nil {
			return nil, err
		}
	}
	path := fmt.Sprintf("/api/mchs/waterlevel/%s/?dtime_from=%s&dtime_to=%s",
		url.PathEscape(waterLevelUUID),
		url.QueryEscape(from.Format(time.RFC3339)),
		url.QueryEscape(to.Format(time.RFC3339)),
	)
	var out HistoryResponse
	if err := c.getJSON(ctx, path, c.access, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, path, bearer string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "weather-fetcher/1.0")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("unexpected status %d for %s: %s", resp.StatusCode, path, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode %s: %w", path, err)
	}
	return nil
}

func ParseTime(s string) (time.Time, error) {
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05-07:00", "2006-01-02 15:04:05Z07:00"}
	var last error
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		} else {
			last = err
		}
	}
	return time.Time{}, last
}

func Float32Ptr(v *float64) *float32 {
	if v == nil {
		return nil
	}
	out := float32(*v)
	return &out
}
