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

// WeatherSeasonContext describes season-specific interpretation of the current period.
type WeatherSeasonContext struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	FocusTitle      string `json:"focus_title"`
	FocusText       string `json:"focus_text"`
	Icon            string `json:"icon"`
	ProgressPercent int    `json:"progress_percent"`
}

// WeatherArchiveBenchmark compares the current month with the same month in previous years.
type WeatherArchiveBenchmark struct {
	Title            string  `json:"title"`
	Subtitle         string  `json:"subtitle"`
	Available        bool    `json:"available"`
	Reliable         bool    `json:"reliable"`
	SampleSize       int     `json:"sample_size"`
	RainTotalAvg     float64 `json:"rain_total_avg"`
	RainDaysAvg      float64 `json:"rain_days_avg"`
	AvgTempAvg       float64 `json:"avg_temp_avg"`
	RainRatioPercent int     `json:"rain_ratio_percent"`
	RainDeltaPercent int     `json:"rain_delta_percent"`
	TempDelta        float64 `json:"temp_delta"`
	Verdict          string  `json:"verdict"`
	StatusText       string  `json:"status_text"`
}

// RollingWeatherPeriod compares a recent rolling window with the preceding window.
type RollingWeatherPeriod struct {
	Title            string                 `json:"title"`
	Subtitle         string                 `json:"subtitle"`
	Current          MonthlyWeatherInsights `json:"current"`
	Previous         MonthlyWeatherInsights `json:"previous"`
	RainDeltaPercent int                    `json:"rain_delta_percent"`
	TempDelta        float64                `json:"temp_delta"`
	Verdict          string                 `json:"verdict"`
}

// WeatherDayTypeSummary describes the dominant character of days in the current month.
type WeatherDayTypeSummary struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Icon        string `json:"icon"`
	Count       int    `json:"count"`
	Percent     int    `json:"percent"`
	Description string `json:"description"`
	Class       string `json:"class"`
}

// WeatherFactorInsight is a compact practical insight for one weather factor.
type WeatherFactorInsight struct {
	Icon       string `json:"icon"`
	Title      string `json:"title"`
	Value      string `json:"value"`
	Detail     string `json:"detail"`
	Advice     string `json:"advice"`
	Level      string `json:"level"`
	LevelLabel string `json:"level_label"`
}

// WeatherTimelineEvent is a notable event in the month.
type WeatherTimelineEvent struct {
	Date        time.Time `json:"date"`
	Icon        string    `json:"icon"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Severity    int       `json:"severity"`
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

// WeatherInsightsPeriodOption is one selectable month or season in the archive navigator.
type WeatherInsightsPeriodOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// WeatherInsightsPage contains all data for the Insights web page.
type WeatherInsightsPage struct {
	GeneratedAt time.Time `json:"generated_at"`

	IsSeason                 bool                          `json:"is_season"`
	PeriodStatus             string                        `json:"period_status"`
	SelectedMonthParam       string                        `json:"selected_month_param"`
	SelectedMonthLabel       string                        `json:"selected_month_label"`
	PreviousMonthParam       string                        `json:"previous_month_param"`
	NextMonthParam           string                        `json:"next_month_param"`
	HasNextMonth             bool                          `json:"has_next_month"`
	IsCurrentMonth           bool                          `json:"is_current_month"`
	SelectedSeasonParam      string                        `json:"selected_season_param"`
	SelectedSeasonLabel      string                        `json:"selected_season_label"`
	PreviousSeasonParam      string                        `json:"previous_season_param"`
	NextSeasonParam          string                        `json:"next_season_param"`
	HasNextSeason            bool                          `json:"has_next_season"`
	SeasonOptions            []WeatherInsightsPeriodOption `json:"season_options"`
	CurrentPeriodLabel       string                        `json:"current_period_label"`
	CurrentPeriodGenitive    string                        `json:"current_period_genitive"`
	CurrentPeriodPreposition string                        `json:"current_period_preposition"`

	CurrentMonth       MonthlyWeatherInsights `json:"current_month"`
	PreviousMonth      MonthlyWeatherInsights `json:"previous_month"`
	PreviousSamePeriod MonthlyWeatherInsights `json:"previous_same_period"`

	Season             WeatherSeasonContext    `json:"season"`
	SameMonthBenchmark WeatherArchiveBenchmark `json:"same_month_benchmark"`
	Last7Days          RollingWeatherPeriod    `json:"last_7_days"`
	Last30Days         RollingWeatherPeriod    `json:"last_30_days"`

	DayTypes        []WeatherDayTypeSummary  `json:"day_types"`
	DominantDayType WeatherDayTypeSummary    `json:"dominant_day_type"`
	WindInsight     WeatherFactorInsight     `json:"wind_insight"`
	UVInsight       WeatherFactorInsight     `json:"uv_insight"`
	Timeline        []WeatherTimelineEvent   `json:"timeline"`
	SeasonMonths    []MonthlyWeatherInsights `json:"season_months"`

	CurrentDryStreak  int       `json:"current_dry_streak"`
	HasLastRain       bool      `json:"has_last_rain"`
	LastRainDate      time.Time `json:"last_rain_date"`
	DaysSinceLastRain int       `json:"days_since_last_rain"`

	MainInsight WeatherInsightStory   `json:"main_insight"`
	Stories     []WeatherInsightStory `json:"stories"`

	MonthProgressPercent      int `json:"month_progress_percent"`
	RainVsPreviousSamePercent int `json:"rain_vs_previous_same_percent"`
	RainVsPreviousFullPercent int `json:"rain_vs_previous_full_percent"`
	RainiestDaySharePercent   int `json:"rainiest_day_share_percent"`
	ComfortPercent            int `json:"comfort_percent"`
	SunnyPercent              int `json:"sunny_percent"`
	RainDaysPercent           int `json:"rain_days_percent"`

	Calendar      []CalendarWeatherDay   `json:"calendar"`
	RainChartData map[string]interface{} `json:"rain_chart_data"`
	BestDay       *NotableWeatherDay     `json:"best_day,omitempty"`
	WorstDay      *NotableWeatherDay     `json:"worst_day,omitempty"`
}
