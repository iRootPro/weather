// Package xras — клиент для публичного JSON геомагнитной активности с xras.ru.
//
// Источник: https://xras.ru/txt/kp_PIL9.json — индекс Kp по 3-часовым слотам
// плюс суточные показатели солнечной активности (F10.7, число Вольфа, Ap).
//
// Формат JSON у xras.ru нестандартный: все числовые значения приходят
// строками, отсутствующие данные — литералом "null", часовой пояс — текстом
// вида "Krasnodar (UTC+03)". Парсер разбирает это в типизированные значения.
package xras

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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

// Response — корневой объект JSON. Все числовые поля у xras.ru — строки.
type Response struct {
	Version string     `json:"version"`
	Type    string     `json:"type"`
	Error   string     `json:"error"`
	Tzone   string     `json:"tzone"`
	Stime   string     `json:"stime"`
	Etime   string     `json:"etime"`
	KpType  string     `json:"kp_type"`
	KpStep  string     `json:"kp_step"`
	Data    []DayEntry `json:"data"`
}

// DayEntry — суточная запись с почасовыми (3ч) значениями Kp.
// Все числа приходят строками; "null" означает отсутствие значения.
type DayEntry struct {
	Time  string `json:"time"` // "YYYY-MM-DD"
	F10   string `json:"f10"`
	Sn    string `json:"sn"`
	Ap    string `json:"ap"`
	MaxKp string `json:"max_kp"`
	H00   string `json:"h00"`
	H03   string `json:"h03"`
	H06   string `json:"h06"`
	H09   string `json:"h09"`
	H12   string `json:"h12"`
	H15   string `json:"h15"`
	H18   string `json:"h18"`
	H21   string `json:"h21"`
}

// Slots возвращает 8 распарсенных значений Kp в порядке часов 0,3,...,21.
// Отсутствующие значения — nil.
func (d DayEntry) Slots() [8]*float64 {
	return [8]*float64{
		ParseNullableFloat(d.H00),
		ParseNullableFloat(d.H03),
		ParseNullableFloat(d.H06),
		ParseNullableFloat(d.H09),
		ParseNullableFloat(d.H12),
		ParseNullableFloat(d.H15),
		ParseNullableFloat(d.H18),
		ParseNullableFloat(d.H21),
	}
}

// F10Float, SnFloat, ApFloat, MaxKpFloat возвращают распарсенные суточные
// показатели или nil, если значение отсутствует.
func (d DayEntry) F10Float() *float64   { return ParseNullableFloat(d.F10) }
func (d DayEntry) SnFloat() *float64    { return ParseNullableFloat(d.Sn) }
func (d DayEntry) ApFloat() *float64    { return ParseNullableFloat(d.Ap) }
func (d DayEntry) MaxKpFloat() *float64 { return ParseNullableFloat(d.MaxKp) }

// ParseNullableFloat превращает строку из JSON в *float64.
// Возвращает nil для пустой строки, "null" или нераспарсиваемых значений.
func ParseNullableFloat(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
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

// tzoneRegex выделяет часть "UTC[+-]NN" из произвольной строки часового пояса
// формата xras.ru, например "Krasnodar (UTC+03)".
var tzoneRegex = regexp.MustCompile(`(?i)UTC([+-]?\d{1,2})?`)

// ParseTzone превращает строку часового пояса в *time.Location.
// Поддерживает форматы: "UTC+3", "UTC-2", "UTC", "Krasnodar (UTC+03)".
// При ошибке или пустой строке возвращает UTC+3 (часовой пояс xras.ru).
func ParseTzone(s string) (*time.Location, error) {
	defaultLoc := time.FixedZone("UTC+3", 3*3600)

	s = strings.TrimSpace(s)
	if s == "" {
		return defaultLoc, nil
	}

	m := tzoneRegex.FindStringSubmatch(s)
	if m == nil {
		return defaultLoc, fmt.Errorf("unknown tzone format: %q", s)
	}

	offsetStr := strings.TrimSpace(m[1])
	if offsetStr == "" {
		return time.UTC, nil
	}

	hours, err := strconv.Atoi(offsetStr)
	if err != nil {
		return defaultLoc, fmt.Errorf("invalid tzone offset %q: %w", s, err)
	}
	return time.FixedZone(s, hours*3600), nil
}
