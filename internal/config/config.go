package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	DB   DBConfig   `yaml:"db"`
	MQTT MQTTConfig `yaml:"mqtt"`
	HTTP HTTPConfig `yaml:"http"`
	Log  LogConfig  `yaml:"log"`
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

type LogConfig struct {
	Level  string `env:"LOG_LEVEL" env-default:"info"`
	Format string `env:"LOG_FORMAT" env-default:"text"`
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
