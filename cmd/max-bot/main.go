package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iRootPro/weather/internal/config"
	"github.com/iRootPro/weather/internal/maxbot"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
	"github.com/iRootPro/weather/pkg/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

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

	if cfg.Max.Token == "" {
		slog.Warn("MAX_TOKEN is empty, max bot is disabled")
		waitForShutdown()
		return
	}

	ctx := context.Background()
	pool, err := database.NewPostgresPool(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	weatherRepo := repository.NewWeatherRepository(pool)
	forecastRepo := repository.NewForecastRepository(pool)
	geomagRepo := repository.NewGeomagneticRepository(pool)
	userRepo := repository.NewMaxUserRepository(pool)
	subRepo := repository.NewMaxSubscriptionRepository(pool)
	notifRepo := repository.NewMaxNotificationRepository(pool)

	weatherService := service.NewWeatherService(weatherRepo)
	forecastService := service.NewForecastService(forecastRepo)
	sunService, err := service.NewSunService(cfg.Location.Latitude, cfg.Location.Longitude, cfg.Location.Timezone)
	if err != nil {
		log.Fatalf("failed to create sun service: %v", err)
	}
	geomagneticService := service.NewGeomagneticService(geomagRepo, cfg.Geomagnetic.AlertThreshold)

	client := maxbot.NewClient(cfg.Max.Token, time.Duration(cfg.Max.UpdateTimeout+10)*time.Second)
	me, err := client.GetMe(ctx)
	if err != nil {
		log.Fatalf("failed to authorize max bot: %v", err)
	}
	slog.Info("max bot authorized", "user_id", me.UserID, "username", me.Username, "name", me.FirstName)
	if err := configureBotProfile(ctx, client); err != nil {
		slog.Warn("failed to update max bot profile commands", "error", err)
	}

	handler := maxbot.NewBotHandler(client, weatherService, forecastService, userRepo, subRepo, logger)
	notifier := maxbot.NewNotifier(client, weatherService, subRepo, notifRepo, userRepo, cfg.Max.NotifyInterval, logger)
	dailySummary := maxbot.NewDailySummaryService(client, weatherService, sunService, geomagneticService, subRepo, cfg.Max.DailySummaryTime, logger)

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go notifier.Start(runCtx)
	go dailySummary.Start(runCtx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go pollUpdates(runCtx, client, handler, cfg.Max.UpdateTimeout, logger)
	sig := <-sigChan
	slog.Info("received shutdown signal", "signal", sig)
	cancel()
	slog.Info("max bot stopped")
}

func configureBotProfile(ctx context.Context, client *maxbot.Client) error {
	return client.PatchMe(ctx, maxbot.BotPatch{
		Description: "Погода в Армавире: текущие данные метеостанции, уведомления о погодных событиях и утренняя сводка.",
		Commands: []maxbot.BotCommand{
			{Name: "start", Description: "Показать главное меню"},
			{Name: "menu", Description: "Показать меню"},
			{Name: "weather", Description: "Текущая погода"},
			{Name: "subscribe", Description: "Настроить уведомления"},
			{Name: "unsubscribe", Description: "Отключить все уведомления"},
			{Name: "help", Description: "Помощь по боту"},
		},
	})
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	slog.Info("max bot disabled process stopped")
}

func pollUpdates(ctx context.Context, client *maxbot.Client, handler *maxbot.BotHandler, timeout int, logger *slog.Logger) {
	var marker *int64
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, err := client.GetUpdates(ctx, marker, 100, timeout, []string{"bot_started", "bot_stopped", "message_created", "message_callback"})
		if err != nil {
			logger.Error("failed to get max updates", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}
		if updates.Marker != nil {
			marker = updates.Marker
		}
		for _, update := range updates.Updates {
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("panic handling max update", "panic", r, "update_type", update.UpdateType)
					}
				}()
				handler.HandleUpdate(ctx, update)
			}()
		}
	}
}
