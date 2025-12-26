package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iRootPro/weather/internal/config"
	"github.com/iRootPro/weather/internal/handler/api"
	"github.com/iRootPro/weather/internal/handler/web"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
	"github.com/iRootPro/weather/pkg/database"
)

func main() {
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
	sensorRepo := repository.NewSensorRepository(pool)

	// Инициализация сервисов
	weatherService := service.NewWeatherService(weatherRepo)
	sensorService := service.NewSensorService(sensorRepo)
	sunService, err := service.NewSunService(cfg.Location.Latitude, cfg.Location.Longitude, cfg.Location.Timezone)
	if err != nil {
		log.Fatalf("failed to create sun service: %v", err)
	}
	slog.Info("creating moon service", "latitude", cfg.Location.Latitude, "longitude", cfg.Location.Longitude, "timezone", cfg.Location.Timezone)
	moonService, err := service.NewMoonService(cfg.Location.Latitude, cfg.Location.Longitude, cfg.Location.Timezone)
	if err != nil {
		log.Fatalf("failed to create moon service: %v", err)
	}
	slog.Info("moon service created successfully")

	// Инициализация хендлеров
	weatherHandler := api.NewWeatherHandler(weatherService)
	sensorHandler := api.NewSensorHandler(sensorService)

	// Web handler - try Docker path first, then local development path
	templatesDir := "templates"
	staticDir := "static"
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		templatesDir = "internal/web/templates"
		staticDir = "internal/web/static"
	}

	slog.Info("creating web handler", "templatesDir", templatesDir)
	webHandler, err := web.NewHandler(templatesDir, weatherService, sunService, moonService)
	if err != nil {
		log.Fatalf("failed to create web handler: %v", err)
	}
	slog.Info("web handler created successfully")

	// Настройка роутера
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", healthHandler)

	// Weather API
	mux.HandleFunc("GET /api/weather/current", weatherHandler.GetCurrent)
	mux.HandleFunc("GET /api/weather/history", weatherHandler.GetHistory)
	mux.HandleFunc("GET /api/weather/stats", weatherHandler.GetStats)
	mux.HandleFunc("GET /api/weather/chart", weatherHandler.GetChartData)
	mux.HandleFunc("GET /api/weather/events", weatherHandler.GetEvents)

	// Sensors API
	mux.HandleFunc("GET /api/sensors", sensorHandler.GetAll)
	mux.HandleFunc("GET /api/sensors/{code}", sensorHandler.GetByCode)

	// Web pages
	mux.HandleFunc("GET /", webHandler.Dashboard)
	mux.HandleFunc("GET /history", webHandler.History)
	mux.HandleFunc("GET /records", webHandler.Records)
	mux.HandleFunc("GET /help", webHandler.Help)

	// Detail pages
	mux.HandleFunc("GET /detail/temperature", webHandler.DetailTemperature)
	mux.HandleFunc("GET /detail/humidity", webHandler.DetailHumidity)
	mux.HandleFunc("GET /detail/pressure", webHandler.DetailPressure)
	mux.HandleFunc("GET /detail/wind", webHandler.DetailWind)
	mux.HandleFunc("GET /detail/rain", webHandler.DetailRain)
	mux.HandleFunc("GET /detail/solar", webHandler.DetailSolar)

	// HTMX widgets
	mux.HandleFunc("GET /widgets/current", webHandler.CurrentWeatherWidget)
	mux.HandleFunc("GET /widgets/stats", webHandler.StatsWidget)
	slog.Info("registering sun widget route")
	mux.HandleFunc("GET /widgets/sun", webHandler.SunTimesWidget)
	slog.Info("sun widget route registered")
	mux.HandleFunc("GET /widgets/events", webHandler.WeatherEventsWidget)

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Middleware
	handler := corsMiddleware(loggingMiddleware(mux))

	// Создание сервера
	server := &http.Server{
		Addr:         cfg.HTTP.Addr(),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		slog.Info("shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	slog.Info("starting API server", "addr", cfg.HTTP.Addr())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to start server: %v", err)
	}

	slog.Info("server stopped")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
		)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
