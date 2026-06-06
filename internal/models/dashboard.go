package models

import "time"

// DashboardSeverity — уровень важности/серьёзности карточки умного дашборда.
type DashboardSeverity string

const (
	DashboardSeverityCalm    DashboardSeverity = "calm"
	DashboardSeverityNormal  DashboardSeverity = "normal"
	DashboardSeverityInfo    DashboardSeverity = "info"
	DashboardSeverityWarning DashboardSeverity = "warning"
	DashboardSeverityDanger  DashboardSeverity = "danger"
)

// DashboardSnapshot — агрегированная модель главного экрана для React/PWA,
// ботов и будущих мобильных приложений. Backend уже решает, что важно сейчас.
type DashboardSnapshot struct {
	GeneratedAt    time.Time              `json:"generated_at"`
	StationStatus  StationStatus          `json:"station_status"`
	Headline       DashboardHeadline      `json:"headline"`
	Summary        string                 `json:"summary,omitempty"`
	CurrentWeather *CurrentWeatherSummary `json:"current_weather,omitempty"`
	NearForecast   []NearForecastItem     `json:"near_forecast,omitempty"`
	Cards          []AttentionCard        `json:"cards"`
	Quiet          QuietSummary           `json:"quiet"`
}

// StationStatus описывает свежесть данных метеостанции.
type StationStatus struct {
	OK         bool       `json:"ok"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	AgeMinutes *int       `json:"age_minutes,omitempty"`
	Label      string     `json:"label"`
	Severity   string     `json:"severity"`
}

// DashboardHeadline — краткий ответ на вопрос «что сейчас главное?».
type DashboardHeadline struct {
	Title    string `json:"title"`
	Summary  string `json:"summary,omitempty"`
	Severity string `json:"severity"`
	Icon     string `json:"icon,omitempty"`
}

// CurrentWeatherSummary — базовое состояние погоды прямо сейчас. Это не
// attention-card: показывается всегда как контекст для пользователя.
type CurrentWeatherSummary struct {
	ObservedAt       time.Time `json:"observed_at"`
	Temperature      *float32  `json:"temperature,omitempty"`
	FeelsLike        *float32  `json:"feels_like,omitempty"`
	TemperatureDelta *float32  `json:"temperature_delta,omitempty"`
	Humidity         *int16    `json:"humidity,omitempty"`
	Pressure         *float32  `json:"pressure,omitempty"`
	PressureDelta    *float32  `json:"pressure_delta,omitempty"`
	WindSpeed        *float32  `json:"wind_speed,omitempty"`
	WindGust         *float32  `json:"wind_gust,omitempty"`
	RainRate         *float32  `json:"rain_rate,omitempty"`
	UVIndex          *float32  `json:"uv_index,omitempty"`
	Icon             string    `json:"icon"`
	Title            string    `json:"title"`
	Subtitle         string    `json:"subtitle"`
}

// NearForecastItem — компактный прогноз на ближайшие часы для главного экрана.
type NearForecastItem struct {
	Time                     time.Time `json:"time"`
	Temperature              float32   `json:"temperature"`
	FeelsLike                float32   `json:"feels_like"`
	PrecipitationProbability int16     `json:"precipitation_probability"`
	Precipitation            float32   `json:"precipitation"`
	WindSpeed                float32   `json:"wind_speed"`
	WeatherDescription       string    `json:"weather_description"`
	Icon                     string    `json:"icon"`
}

// AttentionCard — одна смысловая карточка дашборда.
type AttentionCard struct {
	ID        string `json:"id"`
	Domain    string `json:"domain"`
	Title     string `json:"title"`
	Subtitle  string `json:"subtitle,omitempty"`
	Value     string `json:"value,omitempty"`
	Unit      string `json:"unit,omitempty"`
	Severity  string `json:"severity"`
	Priority  int    `json:"priority"`
	Reason    string `json:"reason,omitempty"`
	Icon      string `json:"icon,omitempty"`
	DetailURL string `json:"detail_url,omitempty"`
}

// QuietSummary группирует спокойные низкоприоритетные домены.
type QuietSummary struct {
	Title string   `json:"title"`
	Items []string `json:"items"`
}

func ClampPriority(priority int) int {
	if priority < 0 {
		return 0
	}
	if priority > 100 {
		return 100
	}
	return priority
}
