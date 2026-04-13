// Package xras — клиент для публичного JSON геомагнитной активности с xras.ru.
//
// Источник: https://xras.ru/txt/kp_PIL9.json — индекс Kp по 3-часовым слотам
// плюс суточные показатели солнечной активности (F10.7, число Вольфа, Ap).
package xras

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://xras.ru/txt/kp_PIL9.json"

// Client — HTTP-клиент xras.ru.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// Response — корневой объект JSON.
type Response struct {
	Version string     `json:"version"`
	Type    string     `json:"type"`
	Tzone   string     `json:"tzone"`
	Stime   string     `json:"stime"`
	Etime   string     `json:"etime"`
	KpType  string     `json:"kp_type"`
	KpStep  int        `json:"kp_step"`
	Data    []DayEntry `json:"data"`
}

// DayEntry — суточная запись с почасовыми (3ч) значениями Kp.
type DayEntry struct {
	Time  string   `json:"time"` // "YYYY-MM-DD"
	F10   *float64 `json:"f10"`
	Sn    *float64 `json:"sn"`
	Ap    *float64 `json:"ap"`
	MaxKp *float64 `json:"max_kp"`
	H00   *float64 `json:"h00"`
	H03   *float64 `json:"h03"`
	H06   *float64 `json:"h06"`
	H09   *float64 `json:"h09"`
	H12   *float64 `json:"h12"`
	H15   *float64 `json:"h15"`
	H18   *float64 `json:"h18"`
	H21   *float64 `json:"h21"`
}

// Slots возвращает 8 значений Kp в порядке часов 0,3,6,...,21.
func (d DayEntry) Slots() [8]*float64 {
	return [8]*float64{d.H00, d.H03, d.H06, d.H09, d.H12, d.H15, d.H18, d.H21}
}

// NewClient создаёт клиент с заданным таймаутом и опциональным прокси.
// Пустой proxyURL → стандартный transport (уважает переменные окружения).
func NewClient(timeout time.Duration, baseURL string, proxyURL string) (*Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	httpClient := &http.Client{Timeout: timeout}
	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(parsed)}
	}

	return &Client{httpClient: httpClient, baseURL: baseURL}, nil
}

// GetKpData выполняет GET и декодирует JSON.
func (c *Client) GetKpData(ctx context.Context) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var out Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &out, nil
}

// ParseTzone превращает строку "UTC+3" / "UTC-2" / "UTC" в *time.Location.
// При ошибке возвращает UTC+3 (часовой пояс источника по умолчанию).
func ParseTzone(s string) (*time.Location, error) {
	defaultLoc := time.FixedZone("UTC+3", 3*3600)

	s = strings.TrimSpace(s)
	if s == "" {
		return defaultLoc, nil
	}

	upper := strings.ToUpper(s)
	if !strings.HasPrefix(upper, "UTC") {
		return defaultLoc, fmt.Errorf("unknown tzone format: %q", s)
	}
	rest := strings.TrimPrefix(upper, "UTC")
	if rest == "" {
		return time.UTC, nil
	}

	sign := 1
	switch rest[0] {
	case '+':
		sign = 1
		rest = rest[1:]
	case '-':
		sign = -1
		rest = rest[1:]
	}

	hours, err := strconv.Atoi(rest)
	if err != nil {
		return defaultLoc, fmt.Errorf("invalid tzone offset %q: %w", s, err)
	}
	return time.FixedZone(s, sign*hours*3600), nil
}
