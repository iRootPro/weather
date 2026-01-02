package repository

import (
	"context"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

type WeatherRepository interface {
	Save(ctx context.Context, data *models.WeatherData) error
	GetLatest(ctx context.Context) (*models.WeatherData, error)
	GetByTimeRange(ctx context.Context, from, to time.Time) ([]models.WeatherData, error)
	GetAggregated(ctx context.Context, from, to time.Time, interval string) ([]models.WeatherData, error)
	GetStats(ctx context.Context, from, to time.Time) (*models.WeatherStats, error)
	GetRecords(ctx context.Context) (*models.WeatherRecords, error)
	GetDataNearTime(ctx context.Context, targetTime time.Time) (*models.WeatherData, error)
	GetDailyMinMax(ctx context.Context) (*DailyMinMax, error)
	GetDataForEventDetection(ctx context.Context, from, to time.Time) ([]models.WeatherData, error)
}

type SensorRepository interface {
	GetAll(ctx context.Context) ([]models.Sensor, error)
	GetByCode(ctx context.Context, code string) (*models.Sensor, error)
}

type TelegramUserRepository interface {
	Create(ctx context.Context, user *models.TelegramUser) error
	GetByID(ctx context.Context, id int64) (*models.TelegramUser, error)
	GetByChatID(ctx context.Context, chatID int64) (*models.TelegramUser, error)
	GetAll(ctx context.Context) ([]models.TelegramUser, error)
	GetAllActive(ctx context.Context) ([]models.TelegramUser, error)
	UpdateActivity(ctx context.Context, chatID int64, isActive bool) error
}

type TelegramSubscriptionRepository interface {
	Create(ctx context.Context, sub *models.TelegramSubscription) error
	GetByUserID(ctx context.Context, userID int64) ([]models.TelegramSubscription, error)
	GetActiveSubscribers(ctx context.Context, eventType string) ([]int64, error)
	Delete(ctx context.Context, userID int64, eventType string) error
	DeleteAll(ctx context.Context, userID int64) error
	Toggle(ctx context.Context, userID int64, eventType string, isActive bool) error
}

type TelegramNotificationRepository interface {
	Create(ctx context.Context, notification *models.TelegramNotification) error
	WasRecentlySent(ctx context.Context, userID int64, eventType string, within time.Duration) (bool, error)
}

type ForecastRepository interface {
	SaveHourly(ctx context.Context, data *models.ForecastData) error
	SaveDaily(ctx context.Context, data *models.ForecastData) error
	SaveBatch(ctx context.Context, data []models.ForecastData) error
	GetHourlyForecast(ctx context.Context, from, to time.Time) ([]models.ForecastData, error)
	GetDailyForecast(ctx context.Context, from, to time.Time) ([]models.ForecastData, error)
	GetLatestHourly(ctx context.Context, hours int) ([]models.ForecastData, error)
	GetLatestDaily(ctx context.Context, days int) ([]models.ForecastData, error)
	DeleteOldForecasts(ctx context.Context, olderThan time.Time) error
}

type PhotoRepository interface {
	Create(ctx context.Context, photo *models.Photo) error
	GetByID(ctx context.Context, id int64) (*models.Photo, error)
	GetAll(ctx context.Context, limit, offset int) ([]models.Photo, error)
	GetVisible(ctx context.Context, limit, offset int) ([]models.Photo, error)
	GetByUserID(ctx context.Context, userID int64) ([]models.Photo, error)
	UpdateVisibility(ctx context.Context, id int64, isVisible bool) error
	Delete(ctx context.Context, id int64) error
	GetWeatherForTime(ctx context.Context, takenAt time.Time) (*models.WeatherData, error)
}

type NarodmonLogRepository interface {
	Create(ctx context.Context, log *models.NarodmonLog) error
	GetLatest(ctx context.Context) (*models.NarodmonLog, error)
	GetRecent(ctx context.Context, limit int) ([]models.NarodmonLog, error)
	DeleteOld(ctx context.Context, olderThan time.Time) error
}
