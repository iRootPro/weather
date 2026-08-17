package models

import "time"

// WeatherArchiveSummary is a period-level summary calculated from daily station aggregates.
// Temperature, humidity and pressure are averaged by calendar day so uneven packet volume
// cannot bias the result.
type WeatherArchiveSummary struct {
	DaysInPeriod int `json:"days_in_period"`
	DaysWithData int `json:"days_with_data"`

	TempMin float64 `json:"temp_min"`
	TempAvg float64 `json:"temp_avg"`
	TempMax float64 `json:"temp_max"`
	HasTemp bool    `json:"has_temp"`

	RainTotal float64 `json:"rain_total"`
	RainDays  int     `json:"rain_days"`
	HasRain   bool    `json:"has_rain"`

	WindSpeedMax float64 `json:"wind_speed_max"`
	WindGustMax  float64 `json:"wind_gust_max"`
	HasWind      bool    `json:"has_wind"`

	HumidityAvg float64 `json:"humidity_avg"`
	PressureAvg float64 `json:"pressure_avg"`
	HasAir      bool    `json:"has_air"`

	UVIndexMax        float64 `json:"uv_index_max"`
	SolarRadiationMax float64 `json:"solar_radiation_max"`
	HasSun            bool    `json:"has_sun"`
}

// WeatherArchivePage is the data contract for the interactive HTMX weather archive.
type WeatherArchivePage struct {
	GeneratedAt time.Time `json:"generated_at"`

	Period         string `json:"period"`
	Metric         string `json:"metric"`
	PeriodLabel    string `json:"period_label"`
	FromParam      string `json:"from_param"`
	ToParam        string `json:"to_param"`
	FirstDateParam string `json:"first_date_param"`
	LastDateParam  string `json:"last_date_param"`
	MonthParam     string `json:"month_param"`
	SeasonParam    string `json:"season_param"`
	YearParam      int    `json:"year_param"`

	SeasonOptions []WeatherInsightsPeriodOption `json:"season_options"`
	YearOptions   []int                         `json:"year_options"`

	Summary WeatherArchiveSummary `json:"summary"`
	Daily   []DailyWeatherInsight `json:"daily"`
}
