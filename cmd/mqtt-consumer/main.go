package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/iRootPro/weather/internal/config"
	"github.com/iRootPro/weather/internal/mqtt"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/pkg/database"
	"github.com/iRootPro/weather/pkg/mqttclient"
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

	logger.Info("starting mqtt-consumer")

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

	// MQTT обработчик
	handler := mqtt.NewHandler(weatherRepo, logger)

	// MQTT клиент
	mqttClient, err := mqttclient.New(mqttclient.Config{
		BrokerURL: cfg.MQTT.BrokerURL(),
		ClientID:  cfg.MQTT.ClientID,
		Username:  cfg.MQTT.Username,
		Password:  cfg.MQTT.Password,
	}, logger)
	if err != nil {
		logger.Error("failed to connect to MQTT broker", "error", err)
		os.Exit(1)
	}
	defer mqttClient.Disconnect()

	// Подписка на топик
	if err := mqttClient.Subscribe(cfg.MQTT.Topic, 1, handler.HandleMessage()); err != nil {
		logger.Error("failed to subscribe to topic", "error", err)
		os.Exit(1)
	}

	logger.Info("mqtt-consumer started",
		"mqtt_broker", cfg.MQTT.BrokerURL(),
		"topic", cfg.MQTT.Topic,
	)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down mqtt-consumer")
}
