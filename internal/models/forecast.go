package models

import "time"

// ForecastData представляет данные прогноза погоды
type ForecastData struct {
	ID           int64     `json:"id" db:"id"`
	ForecastTime time.Time `json:"forecast_time" db:"forecast_time"`

	// Температура (°C)
	Temperature    *float32 `json:"temperature,omitempty" db:"temperature"`
	TemperatureMin *float32 `json:"temperature_min,omitempty" db:"temperature_min"`
	TemperatureMax *float32 `json:"temperature_max,omitempty" db:"temperature_max"`
	FeelsLike      *float32 `json:"feels_like,omitempty" db:"feels_like"`

	// Осадки
	PrecipitationProbability *int16   `json:"precipitation_probability,omitempty" db:"precipitation_probability"` // %
	Precipitation            *float32 `json:"precipitation,omitempty" db:"precipitation"`                         // мм

	// Ветер
	WindSpeed     *float32 `json:"wind_speed,omitempty" db:"wind_speed"`         // км/ч (Open-Meteo, преобразуется сервисом)
	WindDirection *int16   `json:"wind_direction,omitempty" db:"wind_direction"` // градусы 0-360
	WindGusts     *float32 `json:"wind_gusts,omitempty" db:"wind_gusts"`         // км/ч (Open-Meteo, преобразуется сервисом)

	// Облачность и другое
	CloudCover *int16   `json:"cloud_cover,omitempty" db:"cloud_cover"` // %
	Pressure   *float32 `json:"pressure,omitempty" db:"pressure"`       // гПа
	Humidity   *int16   `json:"humidity,omitempty" db:"humidity"`       // %
	UVIndex    *float32 `json:"uv_index,omitempty" db:"uv_index"`

	// Описание погоды
	WeatherCode        *int16  `json:"weather_code,omitempty" db:"weather_code"`
	WeatherDescription *string `json:"weather_description,omitempty" db:"weather_description"`

	// Тип прогноза
	ForecastType string `json:"forecast_type" db:"forecast_type"` // "hourly" или "daily"

	// Время получения данных
	FetchedAt time.Time `json:"fetched_at" db:"fetched_at"`
}

// HourlyForecast представляет почасовой прогноз (упрощенная структура для API)
type HourlyForecast struct {
	Time                        time.Time `json:"time"`
	Temperature                 float32   `json:"temperature"`
	FeelsLike                   float32   `json:"feels_like"`
	PrecipitationProbability    int16     `json:"precipitation_probability"`
	Precipitation               float32   `json:"precipitation"`
	WindSpeed                   float32   `json:"wind_speed"`
	WindGusts                   float32   `json:"wind_gusts"`
	WindDirection               int16     `json:"wind_direction"`
	WeatherCode                 int16     `json:"weather_code"`
	WeatherDescription          string    `json:"weather_description"`
	Icon                        string    `json:"icon"`
	FetchedAt                   time.Time `json:"-"`
	HasTemperature              bool      `json:"-"`
	HasFeelsLike                bool      `json:"-"`
	HasPrecipitationProbability bool      `json:"-"`
	HasPrecipitation            bool      `json:"-"`
	HasWindSpeed                bool      `json:"-"`
	HasWindGusts                bool      `json:"-"`
}

// DailyForecast представляет дневной прогноз (упрощенная структура для API)
type DailyForecast struct {
	Date                        time.Time `json:"date"`
	TemperatureMin              float32   `json:"temperature_min"`
	TemperatureMax              float32   `json:"temperature_max"`
	PrecipitationProbability    int16     `json:"precipitation_probability"`
	PrecipitationSum            float32   `json:"precipitation_sum"`
	WindSpeedMax                float32   `json:"wind_speed_max"`
	WindGustsMax                float32   `json:"wind_gusts_max"`
	UVIndexMax                  float32   `json:"uv_index_max"`
	WindDirection               int16     `json:"wind_direction"`
	WeatherCode                 int16     `json:"weather_code"`
	WeatherDescription          string    `json:"weather_description"`
	Icon                        string    `json:"icon"`
	FetchedAt                   time.Time `json:"-"`
	HasTemperatureMin           bool      `json:"-"`
	HasTemperatureMax           bool      `json:"-"`
	HasPrecipitationProbability bool      `json:"-"`
	HasPrecipitationSum         bool      `json:"-"`
	HasWindSpeedMax             bool      `json:"-"`
	HasWindGustsMax             bool      `json:"-"`
	HasUVIndexMax               bool      `json:"-"`
}

// GetWeatherDescription возвращает текстовое описание по WMO коду погоды
func GetWeatherDescription(code int16) string {
	switch code {
	case 0:
		return "Ясно"
	case 1:
		return "Преимущественно ясно"
	case 2:
		return "Переменная облачность"
	case 3:
		return "Облачно"
	case 45, 48:
		return "Туман"
	case 51, 53, 55:
		return "Морось"
	case 56, 57:
		return "Ледяная морось"
	case 61:
		return "Небольшой дождь"
	case 63:
		return "Дождь"
	case 65:
		return "Сильный дождь"
	case 66, 67:
		return "Ледяной дождь"
	case 71:
		return "Небольшой снег"
	case 73:
		return "Снег"
	case 75:
		return "Сильный снег"
	case 77:
		return "Снежная крупа"
	case 80, 81, 82:
		return "Ливень"
	case 85, 86:
		return "Снегопад"
	case 95:
		return "Гроза"
	case 96, 99:
		return "Гроза с градом"
	default:
		return "Неизвестно"
	}
}

// GetWeatherIcon возвращает эмодзи для кода погоды
func GetWeatherIcon(code int16) string {
	switch code {
	case 0:
		return "☀️"
	case 1:
		return "🌤️"
	case 2:
		return "⛅"
	case 3:
		return "☁️"
	case 45, 48:
		return "🌫️"
	case 51, 53, 55, 56, 57:
		return "🌦️"
	case 61, 63, 65, 66, 67, 80, 81, 82:
		return "🌧️"
	case 71, 73, 75, 77, 85, 86:
		return "🌨️"
	case 95, 96, 99:
		return "⛈️"
	default:
		return "🌡️"
	}
}
