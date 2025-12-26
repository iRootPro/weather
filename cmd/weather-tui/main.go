package main

import (
	"log"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/iRootPro/weather/internal/apiclient"
	"github.com/iRootPro/weather/internal/config"
	"github.com/iRootPro/weather/internal/service"
	"github.com/iRootPro/weather/internal/tui"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
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

	slog.Info("connecting to API", "url", cfg.API.URL)

	// Инициализация сервисов через API
	weatherService := apiclient.NewWeatherService(cfg.API.URL)
	sunService, err := service.NewSunService(cfg.Location.Latitude, cfg.Location.Longitude, cfg.Location.Timezone)
	if err != nil {
		log.Fatalf("failed to create sun service: %v", err)
	}

	// Запуск TUI
	p := tea.NewProgram(
		tui.NewModel(weatherService, sunService),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}
