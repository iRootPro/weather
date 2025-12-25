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
}

type SensorRepository interface {
	GetAll(ctx context.Context) ([]models.Sensor, error)
	GetByCode(ctx context.Context, code string) (*models.Sensor, error)
}
