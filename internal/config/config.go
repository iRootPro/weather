package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	DB       DBConfig       `yaml:"db"`
	MQTT     MQTTConfig     `yaml:"mqtt"`
	HTTP     HTTPConfig     `yaml:"http"`
	API      APIConfig      `yaml:"api"`
	Log      LogConfig      `yaml:"log"`
	Location LocationConfig `yaml:"location"`
	Telegram TelegramConfig `yaml:"telegram"`
	Forecast ForecastConfig `yaml:"forecast"`
}

type LocationConfig struct {
	Latitude  float64 `env:"LOCATION_LATITUDE" env-default:"44.995574"`
	Longitude float64 `env:"LOCATION_LONGITUDE" env-default:"41.128354"`
	Timezone  string  `env:"LOCATION_TIMEZONE" env-default:"Europe/Moscow"`
}

type DBConfig struct {
	Host     string `env:"DB_HOST" env-default:"localhost"`
	Port     int    `env:"DB_PORT" env-default:"5432"`
	User     string `env:"DB_USER" env-default:"weather"`
	Password string `env:"DB_PASSWORD" env-default:"weather"`
	Name     string `env:"DB_NAME" env-default:"weather"`
	SSLMode  string `env:"DB_SSLMODE" env-default:"disable"`
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}

type MQTTConfig struct {
	Host     string `env:"MQTT_HOST" env-required:"true"`
	Port     int    `env:"MQTT_PORT" env-default:"1883"`
	Username string `env:"MQTT_USERNAME"`
	Password string `env:"MQTT_PASSWORD"`
	Topic    string `env:"MQTT_TOPIC" env-default:"ecowitt/#"`
	ClientID string `env:"MQTT_CLIENT_ID" env-default:"weather-consumer"`
}

func (c MQTTConfig) BrokerURL() string {
	return fmt.Sprintf("tcp://%s:%d", c.Host, c.Port)
}

type HTTPConfig struct {
	Host string `env:"HTTP_HOST" env-default:"0.0.0.0"`
	Port int    `env:"HTTP_PORT" env-default:"8080"`
}

func (c HTTPConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type APIConfig struct {
	URL string `env:"API_URL" env-default:"http://localhost:8080"`
}

type LogConfig struct {
	Level  string `env:"LOG_LEVEL" env-default:"info"`
	Format string `env:"LOG_FORMAT" env-default:"text"`
}

type TelegramConfig struct {
	Token             string  `env:"TELEGRAM_TOKEN"`
	Debug             bool    `env:"TELEGRAM_DEBUG" env-default:"false"`
	UpdateTimeout     int     `env:"TELEGRAM_UPDATE_TIMEOUT" env-default:"60"`
	NotifyInterval    int     `env:"TELEGRAM_NOTIFY_INTERVAL" env-default:"300"` // секунды
	MaxRetries        int     `env:"TELEGRAM_MAX_RETRIES" env-default:"3"`
	AdminIDs          []int64 `env:"TELEGRAM_ADMIN_IDS" env-separator:","` // chat_id админов через запятую
	DailySummaryTime  string  `env:"TELEGRAM_DAILY_SUMMARY_TIME" env-default:"07:00"` // Время отправки ежедневной сводки
	WebsiteURL        string  `env:"WEBSITE_URL" env-default:"https://example.com"` // Публичный URL сайта
}

type ForecastConfig struct {
	UpdateInterval int    `env:"FORECAST_UPDATE_INTERVAL" env-default:"3600"` // секунды (по умолчанию 1 час)
	HourlyHours    int    `env:"FORECAST_HOURLY_HOURS" env-default:"48"`      // сколько часов вперед получать почасовой прогноз
	DailyDays      int    `env:"FORECAST_DAILY_DAYS" env-default:"7"`         // сколько дней вперед получать дневной прогноз
	APITimeout     int    `env:"FORECAST_API_TIMEOUT" env-default:"30"`       // таймаут API запросов в секундах
}

func Load() (*Config, error) {
	// Загружаем .env файл если существует
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return &cfg, nil
}
