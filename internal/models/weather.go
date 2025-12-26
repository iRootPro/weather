package models

import (
	"encoding/json"
	"time"
)

type WeatherData struct {
	Time time.Time `json:"time" db:"time"`

	// Температура (°C)
	TempOutdoor *float32 `json:"temp_outdoor,omitempty" db:"temp_outdoor"`
	TempIndoor  *float32 `json:"temp_indoor,omitempty" db:"temp_indoor"`

	// Влажность (%)
	HumidityOutdoor *int16 `json:"humidity_outdoor,omitempty" db:"humidity_outdoor"`
	HumidityIndoor  *int16 `json:"humidity_indoor,omitempty" db:"humidity_indoor"`

	// Давление (мм рт. ст.)
	PressureRelative *float32 `json:"pressure_relative,omitempty" db:"pressure_relative"`
	PressureAbsolute *float32 `json:"pressure_absolute,omitempty" db:"pressure_absolute"`

	// Ветер
	WindSpeed     *float32 `json:"wind_speed,omitempty" db:"wind_speed"`         // м/с
	WindGust      *float32 `json:"wind_gust,omitempty" db:"wind_gust"`           // м/с
	WindDirection *int16   `json:"wind_direction,omitempty" db:"wind_direction"` // градусы 0-360

	// Осадки (мм)
	RainRate    *float32 `json:"rain_rate,omitempty" db:"rain_rate"`       // мм/ч
	RainDaily   *float32 `json:"rain_daily,omitempty" db:"rain_daily"`     // мм
	RainWeekly  *float32 `json:"rain_weekly,omitempty" db:"rain_weekly"`   // мм
	RainMonthly *float32 `json:"rain_monthly,omitempty" db:"rain_monthly"` // мм
	RainYearly  *float32 `json:"rain_yearly,omitempty" db:"rain_yearly"`   // мм

	// Солнце
	UVIndex        *float32 `json:"uv_index,omitempty" db:"uv_index"`
	SolarRadiation *float32 `json:"solar_radiation,omitempty" db:"solar_radiation"` // Вт/м²

	// Дополнительные
	TempFeelsLike *float32 `json:"temp_feels_like,omitempty" db:"temp_feels_like"`
	DewPoint      *float32 `json:"dew_point,omitempty" db:"dew_point"`

	// Сырые данные
	RawData json.RawMessage `json:"raw_data,omitempty" db:"raw_data"`
}

type WeatherStats struct {
	Period    string   `json:"period"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	TempOutdoorMin *float32 `json:"temp_outdoor_min,omitempty"`
	TempOutdoorMax *float32 `json:"temp_outdoor_max,omitempty"`
	TempOutdoorAvg *float32 `json:"temp_outdoor_avg,omitempty"`

	HumidityOutdoorMin *int16 `json:"humidity_outdoor_min,omitempty"`
	HumidityOutdoorMax *int16 `json:"humidity_outdoor_max,omitempty"`
	HumidityOutdoorAvg *int16 `json:"humidity_outdoor_avg,omitempty"`

	PressureRelativeMin *float32 `json:"pressure_relative_min,omitempty"`
	PressureRelativeMax *float32 `json:"pressure_relative_max,omitempty"`
	PressureRelativeAvg *float32 `json:"pressure_relative_avg,omitempty"`

	WindSpeedMax *float32 `json:"wind_speed_max,omitempty"`
	WindGustMax  *float32 `json:"wind_gust_max,omitempty"`

	RainTotal *float32 `json:"rain_total,omitempty"`
}

type ChartData struct {
	Labels   []string             `json:"labels"`
	Datasets map[string][]float64 `json:"datasets"`
}

// RecordValue represents a single record (min/max) with its value and timestamp
type RecordValue struct {
	Value float64   `json:"value"`
	Time  time.Time `json:"time"`
}

// WeatherRecords contains all-time records for various measurements
type WeatherRecords struct {
	// Период данных
	FirstRecord time.Time `json:"first_record"`
	LastRecord  time.Time `json:"last_record"`
	TotalDays   int       `json:"total_days"`

	// Температура
	TempOutdoorMin RecordValue `json:"temp_outdoor_min"`
	TempOutdoorMax RecordValue `json:"temp_outdoor_max"`

	// Влажность
	HumidityOutdoorMin RecordValue `json:"humidity_outdoor_min"`
	HumidityOutdoorMax RecordValue `json:"humidity_outdoor_max"`

	// Давление
	PressureMin RecordValue `json:"pressure_min"`
	PressureMax RecordValue `json:"pressure_max"`

	// Ветер
	WindSpeedMax RecordValue `json:"wind_speed_max"`
	WindGustMax  RecordValue `json:"wind_gust_max"`

	// Осадки
	RainDailyMax RecordValue `json:"rain_daily_max"`

	// Солнечная радиация
	SolarRadiationMax RecordValue `json:"solar_radiation_max"`
	UVIndexMax        RecordValue `json:"uv_index_max"`
}

// WeatherEvent represents a detected weather event
type WeatherEvent struct {
	Type        string    `json:"type"`        // "rain_start", "rain_end", "temp_drop", "temp_rise", "wind_gust", "pressure_drop", "pressure_rise"
	Time        time.Time `json:"time"`        // Время события
	Value       float64   `json:"value"`       // Текущее значение (температура, скорость ветра и т.д.)
	ValueFrom   float64   `json:"value_from"`  // Начальное значение (для изменений)
	Change      float64   `json:"change"`      // Изменение (для температуры/давления)
	Period      string    `json:"period"`      // Период изменения ("за час", "за 3 часа")
	Description string    `json:"description"` // "Начало дождя", "Порыв ветра 15 м/с"
	Details     string    `json:"details"`     // Подробности "755 → 752 мм за 3 часа"
	Icon        string    `json:"icon"`        // Эмодзи для иконки
}
