package models

import "time"

// DailyWeatherInsight contains pre-aggregated weather metrics for one calendar day.
type DailyWeatherInsight struct {
	Date time.Time `json:"date"`

	TempMin *float32 `json:"temp_min,omitempty"`
	TempMax *float32 `json:"temp_max,omitempty"`
	TempAvg *float32 `json:"temp_avg,omitempty"`

	RainTotal   *float32 `json:"rain_total,omitempty"`    // mm per day, based on max(rain_daily)
	RainRateMax *float32 `json:"rain_rate_max,omitempty"` // mm/h

	WindSpeedMax *float32 `json:"wind_speed_max,omitempty"`
	WindGustMax  *float32 `json:"wind_gust_max,omitempty"`

	SolarRadiationMax *float32 `json:"solar_radiation_max,omitempty"`
	UVIndexMax        *float32 `json:"uv_index_max,omitempty"`

	PressureAvg *float32 `json:"pressure_avg,omitempty"`
	HumidityAvg *int16   `json:"humidity_avg,omitempty"`
}

// DayInsightValue represents a notable day with a metric value.
type DayInsightValue struct {
	Date  time.Time `json:"date"`
	Value float64   `json:"value"`
}

// MonthlyWeatherInsights is a compact human-friendly summary for a month.
type MonthlyWeatherInsights struct {
	Title        string    `json:"title"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	DaysInPeriod int       `json:"days_in_period"`
	DaysWithData int       `json:"days_with_data"`

	RainDays      int              `json:"rain_days"`     // >= 1 mm
	WetDays       int              `json:"wet_days"`      // >= 0.2 mm
	DryDays       int              `json:"dry_days"`      // days with data and < 0.2 mm
	RainTotal     float64          `json:"rain_total"`    // mm
	MaxRainRate   float64          `json:"max_rain_rate"` // mm/h
	MaxRainRateAt *DayInsightValue `json:"max_rain_rate_at,omitempty"`
	MaxRainDay    *DayInsightValue `json:"max_rain_day,omitempty"`

	AvgTemp     float64          `json:"avg_temp"`
	HotDays     int              `json:"hot_days"`      // max temp >= 30°C
	VeryHotDays int              `json:"very_hot_days"` // max temp >= 35°C
	FrostDays   int              `json:"frost_days"`    // min temp < 0°C
	MaxTempDay  *DayInsightValue `json:"max_temp_day,omitempty"`
	MinTempDay  *DayInsightValue `json:"min_temp_day,omitempty"`

	WindyDays      int              `json:"windy_days"`       // gust >= 10 m/s
	StrongWindDays int              `json:"strong_wind_days"` // gust >= 15 m/s
	MaxWindGustDay *DayInsightValue `json:"max_wind_gust_day,omitempty"`

	SunnyDays   int              `json:"sunny_days"`   // max solar radiation >= 500 W/m²
	CloudyDays  int              `json:"cloudy_days"`  // max solar radiation < 150 W/m²
	HighUVDays  int              `json:"high_uv_days"` // UV >= 6
	SunniestDay *DayInsightValue `json:"sunniest_day,omitempty"`

	ComfortableDays int `json:"comfortable_days"` // 18..26°C, dry, gust < 8 m/s
}

// WeatherInsightStory is a generated text insight for the page.
type WeatherInsightStory struct {
	Icon  string `json:"icon"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// CalendarWeatherDay is one cell in the monthly insight calendar.
type CalendarWeatherDay struct {
	Date     time.Time `json:"date"`
	Day      int       `json:"day"`
	IsBlank  bool      `json:"is_blank"`
	IsToday  bool      `json:"is_today"`
	IsFuture bool      `json:"is_future"`
	HasData  bool      `json:"has_data"`

	TempMin   float64 `json:"temp_min"`
	TempMax   float64 `json:"temp_max"`
	RainTotal float64 `json:"rain_total"`
	WindGust  float64 `json:"wind_gust"`
	SolarMax  float64 `json:"solar_max"`

	RainLevel string   `json:"rain_level"` // none, wet, rain, heavy
	CardClass string   `json:"card_class"`
	Badges    []string `json:"badges"`
	Summary   string   `json:"summary"`
}

// NotableWeatherDay describes a best/worst day with explanation.
type NotableWeatherDay struct {
	Icon        string    `json:"icon"`
	Title       string    `json:"title"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	Score       int       `json:"score"`
}

// WeatherInsightsPage contains all data for the Insights web page.
type WeatherInsightsPage struct {
	GeneratedAt time.Time `json:"generated_at"`

	CurrentMonth       MonthlyWeatherInsights `json:"current_month"`
	PreviousMonth      MonthlyWeatherInsights `json:"previous_month"`
	PreviousSamePeriod MonthlyWeatherInsights `json:"previous_same_period"`

	CurrentDryStreak  int       `json:"current_dry_streak"`
	HasLastRain       bool      `json:"has_last_rain"`
	LastRainDate      time.Time `json:"last_rain_date"`
	DaysSinceLastRain int       `json:"days_since_last_rain"`

	MainInsight WeatherInsightStory   `json:"main_insight"`
	Stories     []WeatherInsightStory `json:"stories"`

	Calendar      []CalendarWeatherDay   `json:"calendar"`
	RainChartData map[string]interface{} `json:"rain_chart_data"`
	BestDay       *NotableWeatherDay     `json:"best_day,omitempty"`
	WorstDay      *NotableWeatherDay     `json:"worst_day,omitempty"`
}
