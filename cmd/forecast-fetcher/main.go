package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iRootPro/weather/internal/config"
	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/pkg/database"
	"github.com/iRootPro/weather/pkg/openmeteo"
)

func main() {
	// Логгер
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Конфигурация
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Настройка уровня логирования
	var logLevel slog.Level
	switch cfg.Log.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Info("starting forecast-fetcher service")

	// База данных
	ctx := context.Background()
	pool, err := database.NewPostgresPool(ctx, cfg.DB.DSN())
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to database")

	// Репозиторий
	forecastRepo := repository.NewForecastRepository(pool)

	// Open-Meteo клиент
	omClient := openmeteo.NewClient(time.Duration(cfg.Forecast.APITimeout) * time.Second)

	// Создаем fetcher
	fetcher := &Fetcher{
		logger:   logger,
		client:   omClient,
		repo:     forecastRepo,
		location: cfg.Location,
		config:   cfg.Forecast,
	}

	// Выполняем первый запрос сразу при старте
	logger.Info("performing initial forecast fetch")
	if err := fetcher.FetchAndSave(ctx); err != nil {
		logger.Error("failed to fetch initial forecast", "error", err)
	}

	// Запускаем периодическое обновление
	ticker := time.NewTicker(time.Duration(cfg.Forecast.UpdateInterval) * time.Second)
	defer ticker.Stop()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logger.Info("forecast-fetcher service started",
		"update_interval", cfg.Forecast.UpdateInterval,
		"hourly_hours", cfg.Forecast.HourlyHours,
		"daily_days", cfg.Forecast.DailyDays,
	)

	for {
		select {
		case <-ticker.C:
			logger.Info("fetching forecast update")
			if err := fetcher.FetchAndSave(ctx); err != nil {
				logger.Error("failed to fetch forecast", "error", err)
			}

		case <-sigChan:
			logger.Info("shutting down forecast-fetcher service")
			return
		}
	}
}

type Fetcher struct {
	logger   *slog.Logger
	client   *openmeteo.Client
	repo     repository.ForecastRepository
	location config.LocationConfig
	config   config.ForecastConfig
}

func (f *Fetcher) FetchAndSave(ctx context.Context) error {
	startTime := time.Now()

	// Запрос к Open-Meteo API
	req := openmeteo.ForecastRequest{
		Latitude:  f.location.Latitude,
		Longitude: f.location.Longitude,
		Timezone:  f.location.Timezone,
		Hourly:    openmeteo.GetDefaultHourlyParams(),
		Daily:     openmeteo.GetDefaultDailyParams(),
	}

	resp, err := f.client.GetForecast(ctx, req)
	if err != nil {
		return err
	}

	fetchedAt := time.Now()

	// Конвертируем почасовой прогноз
	hourlyData := make([]models.ForecastData, 0)
	for i := 0; i < len(resp.Hourly.Time) && i < f.config.HourlyHours; i++ {
		// Open-Meteo возвращает время в формате "2025-12-29T00:00"
		forecastTime, err := time.Parse("2006-01-02T15:04", resp.Hourly.Time[i])
		if err != nil {
			f.logger.Warn("failed to parse hourly time", "time", resp.Hourly.Time[i], "error", err)
			continue
		}

		temp := float32(resp.Hourly.Temperature[i])
		feelsLike := float32(resp.Hourly.FeelsLike[i])
		precip := float32(resp.Hourly.Precipitation[i])
		windSpeed := float32(resp.Hourly.WindSpeed[i])
		windGusts := float32(resp.Hourly.WindGusts[i])
		pressure := float32(resp.Hourly.Pressure[i])
		uvIndex := float32(resp.Hourly.UVIndex[i])

		precipProb := int16(resp.Hourly.PrecipitationProbability[i])
		windDir := int16(resp.Hourly.WindDirection[i])
		cloudCover := int16(resp.Hourly.CloudCover[i])
		humidity := int16(resp.Hourly.Humidity[i])
		weatherCode := int16(resp.Hourly.WeatherCode[i])

		description := models.GetWeatherDescription(weatherCode)

		data := models.ForecastData{
			ForecastTime:             forecastTime,
			Temperature:              &temp,
			FeelsLike:                &feelsLike,
			PrecipitationProbability: &precipProb,
			Precipitation:            &precip,
			WindSpeed:                &windSpeed,
			WindDirection:            &windDir,
			WindGusts:                &windGusts,
			CloudCover:               &cloudCover,
			Pressure:                 &pressure,
			Humidity:                 &humidity,
			UVIndex:                  &uvIndex,
			WeatherCode:              &weatherCode,
			WeatherDescription:       &description,
			ForecastType:             "hourly",
			FetchedAt:                fetchedAt,
		}
		hourlyData = append(hourlyData, data)
	}

	// Конвертируем дневной прогноз
	dailyData := make([]models.ForecastData, 0)
	for i := 0; i < len(resp.Daily.Time) && i < f.config.DailyDays; i++ {
		forecastTime, err := time.Parse("2006-01-02", resp.Daily.Time[i])
		if err != nil {
			f.logger.Warn("failed to parse daily time", "time", resp.Daily.Time[i], "error", err)
			continue
		}

		tempMin := float32(resp.Daily.TemperatureMin[i])
		tempMax := float32(resp.Daily.TemperatureMax[i])
		precipSum := float32(resp.Daily.PrecipitationSum[i])
		windSpeedMax := float32(resp.Daily.WindSpeedMax[i])
		windGustsMax := float32(resp.Daily.WindGustsMax[i])
		uvIndexMax := float32(resp.Daily.UVIndexMax[i])

		precipProb := int16(resp.Daily.PrecipitationProbability[i])
		windDir := int16(resp.Daily.WindDirection[i])
		weatherCode := int16(resp.Daily.WeatherCode[i])

		description := models.GetWeatherDescription(weatherCode)

		data := models.ForecastData{
			ForecastTime:             forecastTime,
			TemperatureMin:           &tempMin,
			TemperatureMax:           &tempMax,
			PrecipitationProbability: &precipProb,
			Precipitation:            &precipSum,
			WindSpeed:                &windSpeedMax,
			WindDirection:            &windDir,
			WindGusts:                &windGustsMax,
			UVIndex:                  &uvIndexMax,
			WeatherCode:              &weatherCode,
			WeatherDescription:       &description,
			ForecastType:             "daily",
			FetchedAt:                fetchedAt,
		}
		dailyData = append(dailyData, data)
	}

	// Сохраняем все данные батчами
	allData := append(hourlyData, dailyData...)
	if err := f.repo.SaveBatch(ctx, allData); err != nil {
		return err
	}

	// Удаляем старые прогнозы (старше 7 дней)
	oldThreshold := time.Now().AddDate(0, 0, -7)
	if err := f.repo.DeleteOldForecasts(ctx, oldThreshold); err != nil {
		f.logger.Warn("failed to delete old forecasts", "error", err)
	}

	elapsed := time.Since(startTime)
	f.logger.Info("forecast fetched and saved successfully",
		"hourly_count", len(hourlyData),
		"daily_count", len(dailyData),
		"elapsed", elapsed,
	)

	return nil
}
