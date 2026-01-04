package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/iRootPro/weather/internal/config"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
	"github.com/iRootPro/weather/internal/telegram"
	"github.com/iRootPro/weather/pkg/database"
	"github.com/iRootPro/weather/pkg/ipgeolocation"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Проверка наличия токена
	if cfg.Telegram.Token == "" {
		log.Fatal("TELEGRAM_TOKEN is required")
	}

	// Настройка логгера
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

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// Подключение к БД
	ctx := context.Background()
	pool, err := database.NewPostgresPool(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	slog.Info("connected to database")

	// Инициализация репозиториев
	weatherRepo := repository.NewWeatherRepository(pool)
	userRepo := repository.NewTelegramUserRepository(pool)
	subRepo := repository.NewTelegramSubscriptionRepository(pool)
	notifRepo := repository.NewTelegramNotificationRepository(pool)
	forecastRepo := repository.NewForecastRepository(pool)
	photoRepo := repository.NewPhotoRepository(pool)

	// Инициализация сервисов
	weatherService := service.NewWeatherService(weatherRepo)
	forecastService := service.NewForecastService(forecastRepo)
	sunService, err := service.NewSunService(cfg.Location.Latitude, cfg.Location.Longitude, cfg.Location.Timezone)
	if err != nil {
		log.Fatalf("failed to create sun service: %v", err)
	}

	// IPGeolocation client для точных данных о луне
	var astronomyClient *ipgeolocation.Client
	if cfg.Astronomy.APIKey != "" {
		astronomyClient = ipgeolocation.NewClient(cfg.Astronomy.APIKey, time.Duration(cfg.Astronomy.Timeout)*time.Second)
		slog.Info("astronomy API client created for telegram bot", "api", "ipgeolocation.io")
	}

	moonService, err := service.NewMoonService(cfg.Location.Latitude, cfg.Location.Longitude, cfg.Location.Timezone, astronomyClient)
	if err != nil {
		log.Fatalf("failed to create moon service: %v", err)
	}

	slog.Info("services initialized")

	// Создание Telegram бота
	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	bot.Debug = cfg.Telegram.Debug
	slog.Info("bot authorized", "username", bot.Self.UserName)

	// Создание обработчика
	handler := telegram.NewBotHandler(
		bot,
		weatherService,
		sunService,
		moonService,
		forecastService,
		userRepo,
		subRepo,
		notifRepo,
		photoRepo,
		cfg.Telegram.AdminIDs,
		cfg.Telegram.WebsiteURL,
		cfg.Location.Timezone,
		logger,
	)

	// Создание notifier
	notifier := telegram.NewNotifier(
		bot,
		weatherService,
		subRepo,
		notifRepo,
		userRepo,
		cfg.Telegram.NotifyInterval,
		logger,
	)

	// Создание daily summary service
	dailySummary := telegram.NewDailySummaryService(
		bot,
		weatherService,
		sunService,
		forecastService,
		subRepo,
		userRepo,
		cfg.Telegram.DailySummaryTime,
		logger,
	)

	// Контекст с поддержкой отмены
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск notifier в фоне
	go notifier.Start(ctx)

	// Запуск daily summary service в фоне
	go dailySummary.Start(ctx)

	// Настройка Long Polling
	u := tgbotapi.NewUpdate(0)
	u.Timeout = cfg.Telegram.UpdateTimeout

	updates := bot.GetUpdatesChan(u)

	slog.Info("telegram bot started", "polling_timeout", cfg.Telegram.UpdateTimeout)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Основной цикл обработки обновлений
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in update handler", "panic", r)
			}
		}()

		for update := range updates {
			// Обрабатываем каждое обновление с recovery от паники
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("panic handling update", "panic", r, "update_id", update.UpdateID)
					}
				}()
				handler.HandleUpdate(ctx, update)
			}()
		}
	}()

	// Ожидание сигнала завершения
	sig := <-sigChan
	slog.Info("received shutdown signal", "signal", sig)

	// Останавливаем получение обновлений
	bot.StopReceivingUpdates()

	// Отменяем контекст (остановит notifier)
	cancel()

	slog.Info("telegram bot stopped")
}
