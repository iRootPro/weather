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
	"github.com/iRootPro/weather/pkg/narodmon"
)

func main() {
	// Логгер
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("starting narodmon-sender service")

	// Конфигурация
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Проверяем включена ли интеграция
	if !cfg.Narodmon.Enabled {
		logger.Info("narodmon integration is disabled")
		os.Exit(0)
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
	weatherRepo := repository.NewWeatherRepository(pool)
	narodmonLogRepo := repository.NewNarodmonLogRepository(pool)

	// Клиент Narodmon
	narodmonClient := narodmon.NewClient(cfg.Narodmon.Server, cfg.Narodmon.Timeout)

	// Создаём sender
	sender := &Sender{
		logger:          logger,
		weatherRepo:     weatherRepo,
		narodmonClient:  narodmonClient,
		narodmonLogRepo: narodmonLogRepo,
		config:          cfg.Narodmon,
	}

	// Выполняем первую отправку сразу при старте
	logger.Info("performing initial data send")
	if err := sender.SendData(ctx); err != nil {
		logger.Error("failed to send initial data", "error", err)
	}

	// Запускаем периодическую отправку
	ticker := time.NewTicker(time.Duration(cfg.Narodmon.Interval) * time.Second)
	defer ticker.Stop()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logger.Info("narodmon-sender service started",
		"interval", cfg.Narodmon.Interval,
		"server", cfg.Narodmon.Server,
		"mac", cfg.Narodmon.MAC,
	)

	for {
		select {
		case <-ticker.C:
			logger.Info("sending data to narodmon")
			if err := sender.SendData(ctx); err != nil {
				logger.Error("failed to send data", "error", err)
			}

		case <-sigChan:
			logger.Info("shutting down narodmon-sender service")
			return
		}
	}
}

type Sender struct {
	logger          *slog.Logger
	weatherRepo     repository.WeatherRepository
	narodmonClient  *narodmon.Client
	narodmonLogRepo repository.NarodmonLogRepository
	config          config.NarodmonConfig
}

func (s *Sender) SendData(ctx context.Context) error {
	// Получаем последние данные из БД
	latestData, err := s.weatherRepo.GetLatest(ctx)
	if err != nil {
		s.saveLog(ctx, false, 0, err.Error())
		return err
	}

	if latestData == nil {
		s.logger.Warn("no weather data available")
		return nil
	}

	// Формируем датчики для отправки
	sensors := s.buildSensors(latestData)

	// Отправляем данные
	if err := s.narodmonClient.SendData(
		s.config.MAC,
		s.config.DeviceName,
		sensors,
	); err != nil {
		s.saveLog(ctx, false, len(sensors), err.Error())
		return err
	}

	s.saveLog(ctx, true, len(sensors), "")

	s.logger.Info("data sent successfully",
		"sensors_count", len(sensors),
		"timestamp", latestData.Time,
	)

	return nil
}

func (s *Sender) saveLog(ctx context.Context, success bool, sensorsCount int, errorMsg string) {
	log := &models.NarodmonLog{
		SentAt:       time.Now(),
		Success:      success,
		SensorsCount: sensorsCount,
	}
	if errorMsg != "" {
		log.ErrorMessage = &errorMsg
	}

	if err := s.narodmonLogRepo.Create(ctx, log); err != nil {
		s.logger.Error("failed to save narodmon log", "error", err)
		// Не возвращаем ошибку - лог не критичен
	}
}

func (s *Sender) buildSensors(data *models.WeatherData) []narodmon.Sensor {
	sensors := make([]narodmon.Sensor, 0)

	// Температура улица
	if data.TempOutdoor != nil {
		sensors = append(sensors, narodmon.Sensor{
			ID:    "TEMP",
			Value: float64(*data.TempOutdoor),
			Name:  "Температура улица",
		})
	}

	// Ощущаемая температура
	if data.TempFeelsLike != nil {
		sensors = append(sensors, narodmon.Sensor{
			ID:    "TEMPFEEL",
			Value: float64(*data.TempFeelsLike),
			Name:  "Ощущается как",
		})
	}

	// Точка росы
	if data.DewPoint != nil {
		sensors = append(sensors, narodmon.Sensor{
			ID:    "DEWPOINT",
			Value: float64(*data.DewPoint),
			Name:  "Точка росы",
		})
	}

	// Влажность улица
	if data.HumidityOutdoor != nil {
		sensors = append(sensors, narodmon.Sensor{
			ID:    "HUM",
			Value: float64(*data.HumidityOutdoor),
			Name:  "Влажность улица",
		})
	}

	// Давление
	if data.PressureRelative != nil {
		sensors = append(sensors, narodmon.Sensor{
			ID:    "PRES",
			Value: float64(*data.PressureRelative),
			Name:  "Давление",
		})
	}

	// Скорость ветра
	if data.WindSpeed != nil {
		sensors = append(sensors, narodmon.Sensor{
			ID:    "WIND",
			Value: float64(*data.WindSpeed),
			Name:  "Скорость ветра",
		})
	}

	// Направление ветра
	if data.WindDirection != nil {
		sensors = append(sensors, narodmon.Sensor{
			ID:    "WINDDIR",
			Value: float64(*data.WindDirection),
			Name:  "Направление ветра",
		})
	}

	// Интенсивность дождя
	if data.RainRate != nil {
		sensors = append(sensors, narodmon.Sensor{
			ID:    "RAINRATE",
			Value: float64(*data.RainRate),
			Name:  "Дождь интенсивность",
		})
	}

	// Дождь за день
	if data.RainDaily != nil {
		sensors = append(sensors, narodmon.Sensor{
			ID:    "RAINDAY",
			Value: float64(*data.RainDaily),
			Name:  "Дождь за день",
		})
	}

	// UV индекс
	if data.UVIndex != nil {
		sensors = append(sensors, narodmon.Sensor{
			ID:    "UV",
			Value: float64(*data.UVIndex),
			Name:  "UV индекс",
		})
	}

	// Солнечная радиация (конвертируем в люксы)
	if data.SolarRadiation != nil {
		sensors = append(sensors, narodmon.Sensor{
			ID:    "LUX",
			Value: float64(*data.SolarRadiation) * 120, // W/m² -> lux
			Name:  "Освещенность",
		})
	}

	return sensors
}
