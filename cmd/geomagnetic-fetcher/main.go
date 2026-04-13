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
	"github.com/iRootPro/weather/pkg/xras"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
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
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	if !cfg.Geomagnetic.Enabled {
		logger.Info("geomagnetic-fetcher disabled via config, exiting")
		return
	}

	logger.Info("starting geomagnetic-fetcher service")

	ctx := context.Background()
	pool, err := database.NewPostgresPool(ctx, cfg.DB.DSN())
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to database")

	repo := repository.NewGeomagneticRepository(pool)

	client, err := xras.NewClient(
		time.Duration(cfg.Geomagnetic.APITimeout)*time.Second,
		cfg.Geomagnetic.URL,
		cfg.Geomagnetic.ProxyURL,
	)
	if err != nil {
		logger.Error("failed to create xras client", "error", err)
		os.Exit(1)
	}

	fetcher := &Fetcher{
		logger: logger,
		client: client,
		repo:   repo,
	}

	logger.Info("performing initial geomagnetic fetch")
	if err := fetcher.FetchAndSave(ctx); err != nil {
		logger.Error("failed to fetch initial geomagnetic data", "error", err)
	}

	ticker := time.NewTicker(time.Duration(cfg.Geomagnetic.UpdateInterval) * time.Second)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logger.Info("geomagnetic-fetcher service started",
		"update_interval", cfg.Geomagnetic.UpdateInterval,
		"alert_threshold", cfg.Geomagnetic.AlertThreshold,
	)

	for {
		select {
		case <-ticker.C:
			logger.Info("fetching geomagnetic update")
			if err := fetcher.FetchAndSave(ctx); err != nil {
				logger.Error("failed to fetch geomagnetic data", "error", err)
			}
		case <-sigChan:
			logger.Info("shutting down geomagnetic-fetcher service")
			return
		}
	}
}

type Fetcher struct {
	logger *slog.Logger
	client *xras.Client
	repo   repository.GeomagneticRepository
}

func (f *Fetcher) FetchAndSave(ctx context.Context) error {
	startTime := time.Now()

	resp, err := f.client.GetKpData(ctx)
	if err != nil {
		return err
	}

	loc, err := xras.ParseTzone(resp.Tzone)
	if err != nil {
		f.logger.Warn("failed to parse tzone, using fallback", "tzone", resp.Tzone, "error", err)
	}

	now := time.Now()
	slots := make([]models.GeomagneticKp, 0, len(resp.Data)*8)
	dailies := make([]models.GeomagneticDaily, 0, len(resp.Data))

	for _, d := range resp.Data {
		dayMidnight, err := time.ParseInLocation("2006-01-02", d.Time, loc)
		if err != nil {
			f.logger.Warn("failed to parse day", "time", d.Time, "error", err)
			continue
		}

		for i, slot := range d.Slots() {
			if slot == nil {
				continue
			}
			slotTime := dayMidnight.Add(time.Duration(i*3) * time.Hour)
			slots = append(slots, models.GeomagneticKp{
				SlotTime:   slotTime.UTC(),
				Kp:         float32(*slot),
				Source:     "xras.ru",
				IsForecast: slotTime.After(now),
				FetchedAt:  now,
			})
		}

		f10 := d.F10Float()
		sn := d.SnFloat()
		ap := d.ApFloat()
		maxKp := d.MaxKpFloat()
		if f10 != nil || sn != nil || ap != nil || maxKp != nil {
			dailies = append(dailies, models.GeomagneticDaily{
				Date:      dayMidnight.UTC(),
				F10:       float32Ptr(f10),
				Sn:        float32Ptr(sn),
				Ap:        float32Ptr(ap),
				MaxKp:     float32Ptr(maxKp),
				FetchedAt: now,
			})
		}
	}

	if err := f.repo.SaveKpBatch(ctx, slots); err != nil {
		return err
	}
	if err := f.repo.SaveDailyBatch(ctx, dailies); err != nil {
		return err
	}

	// Retention: храним 90 дней истории.
	cutoff := now.AddDate(0, 0, -90)
	if err := f.repo.DeleteOlderThan(ctx, cutoff); err != nil {
		f.logger.Warn("failed to clean old geomagnetic data", "error", err)
	}

	f.logger.Info("geomagnetic fetched and saved",
		"kp_slots", len(slots),
		"daily", len(dailies),
		"elapsed", time.Since(startTime),
	)
	return nil
}

func float32Ptr(p *float64) *float32 {
	if p == nil {
		return nil
	}
	v := float32(*p)
	return &v
}
