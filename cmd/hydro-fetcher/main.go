package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iRootPro/weather/internal/config"
	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/pkg/database"
	"github.com/iRootPro/weather/pkg/emercit"
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

	if !cfg.Hydro.Enabled {
		logger.Info("hydro-fetcher disabled via config, exiting")
		return
	}

	ctx := context.Background()
	pool, err := database.NewPostgresPool(ctx, cfg.DB.DSN())
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := repository.NewHydroRepository(pool)
	client := emercit.NewClient(time.Duration(cfg.Hydro.APITimeout)*time.Second, cfg.Hydro.BaseURL, cfg.Hydro.Username, cfg.Hydro.Password)

	fetcher := &Fetcher{logger: logger, client: client, repo: repo, config: cfg.Hydro}

	logger.Info("performing initial hydro fetch")
	if err := fetcher.FetchAndSave(ctx); err != nil {
		logger.Error("failed to fetch initial hydro data", "error", err)
	}

	ticker := time.NewTicker(time.Duration(cfg.Hydro.UpdateInterval) * time.Second)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logger.Info("hydro-fetcher service started", "update_interval", cfg.Hydro.UpdateInterval, "stations", len(cfg.Hydro.Stations()), "station_uuid", cfg.Hydro.StationUUID)
	for {
		select {
		case <-ticker.C:
			if err := fetcher.FetchAndSave(ctx); err != nil {
				logger.Error("failed to fetch hydro data", "error", err)
			}
		case <-sigChan:
			logger.Info("shutting down hydro-fetcher service")
			return
		}
	}
}

type Fetcher struct {
	logger *slog.Logger
	client *emercit.Client
	repo   repository.HydroRepository
	config config.HydroConfig
}

func (f *Fetcher) FetchAndSave(ctx context.Context) error {
	start := time.Now()
	now := time.Now()

	actual, err := f.client.GetActual(ctx)
	if err != nil {
		return err
	}

	stations := f.config.Stations()
	readings := make([]models.HydroLevelReading, 0, len(stations))
	savedGauges := 0
	from := now.Add(-time.Duration(f.config.HistoryHours) * time.Hour)

	for _, stationRef := range stations {
		isPrimary := stationRef.StationUUID == f.config.StationUUID
		station, ok := actual[stationRef.StationUUID]
		if !ok {
			err := fmt.Errorf("station %s not found in emercit actual response", stationRef.StationUUID)
			if isPrimary {
				return err
			}
			f.logger.Warn("skipping hydro station", "error", err)
			continue
		}
		waterLevels := station.MCHs["waterlevel"]
		mch, ok := waterLevels[stationRef.WaterLevelUUID]
		if !ok {
			err := fmt.Errorf("waterlevel %s not found for station %s", stationRef.WaterLevelUUID, stationRef.StationUUID)
			if isPrimary {
				return err
			}
			f.logger.Warn("skipping hydro station", "error", err)
			continue
		}

		gauge := buildGauge(stationRef.StationUUID, stationRef.WaterLevelUUID, station, mch, now)
		if err := f.repo.SaveGauge(ctx, gauge); err != nil {
			if isPrimary {
				return err
			}
			f.logger.Warn("failed to save upstream hydro gauge", "station_uuid", stationRef.StationUUID, "error", err)
			continue
		}
		savedGauges++

		if reading, err := buildActualReading(stationRef.StationUUID, stationRef.WaterLevelUUID, mch, now); err != nil {
			f.logger.Warn("failed to build actual reading", "station_uuid", stationRef.StationUUID, "error", err)
		} else if reading != nil {
			readings = append(readings, *reading)
		}

		history, err := f.client.GetWaterLevelHistory(ctx, stationRef.WaterLevelUUID, from, now)
		if err != nil {
			f.logger.Warn("failed to fetch hydro history", "station_uuid", stationRef.StationUUID, "error", err)
		} else {
			readings = append(readings, buildHistoryReadings(stationRef.StationUUID, stationRef.WaterLevelUUID, history, now)...)
		}
	}

	if err := f.repo.SaveReadingsBatch(ctx, readings); err != nil {
		return err
	}

	if f.config.RetentionDays > 0 {
		cutoff := now.AddDate(0, 0, -f.config.RetentionDays)
		if err := f.repo.DeleteOlderThan(ctx, cutoff); err != nil {
			f.logger.Warn("failed to clean old hydro readings", "error", err)
		}
	}

	f.logger.Info("hydro fetched and saved", "gauges", savedGauges, "readings", len(readings), "elapsed", time.Since(start))
	return nil
}

func buildGauge(stationUUID, waterLevelUUID string, st emercit.Station, mch emercit.MCH, fetchedAt time.Time) *models.HydroGauge {
	return &models.HydroGauge{
		StationUUID:          stationUUID,
		WaterLevelUUID:       waterLevelUUID,
		Name:                 st.Name,
		ShortName:            st.ShortName,
		HolderName:           st.HolderName,
		Area:                 st.Area,
		District:             st.District,
		Locality:             st.Locality,
		MonitoringObject:     st.MonitoringObject,
		Latitude:             st.Lat,
		Longitude:            st.Lon,
		FixBSM:               emercit.Float32Ptr(mch.Settings.FixBS),
		DryBSM:               emercit.Float32Ptr(mch.Settings.DryBS),
		FloodingPreventionBM: emercit.Float32Ptr(mch.Settings.FloodingPreventionBS),
		FloodingDangerBSM:    emercit.Float32Ptr(mch.Settings.FloodingDangerBS),
		FetchedAt:            fetchedAt,
	}
}

func buildActualReading(stationUUID, waterLevelUUID string, mch emercit.MCH, fetchedAt time.Time) (*models.HydroLevelReading, error) {
	last := mch.State.LastValue
	if last.BS == nil || last.Time == "" {
		return nil, nil
	}
	observedAt, err := emercit.ParseTime(last.Time)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(mch)
	level := float32(*last.BS)
	return &models.HydroLevelReading{
		StationUUID:     stationUUID,
		WaterLevelUUID:  waterLevelUUID,
		ObservedAt:      observedAt,
		LevelBSM:        level,
		LevelZeroM:      emercit.Float32Ptr(last.Zero),
		ChangeCmPerHour: emercit.Float32Ptr(last.HDIIHR),
		LeadText:        last.HDILead,
		StateCode:       mch.State.StateCode,
		LevelCode:       mch.State.LevelCode,
		RawData:         raw,
		FetchedAt:       fetchedAt,
	}, nil
}

func buildHistoryReadings(stationUUID, waterLevelUUID string, history emercit.HistoryResponse, fetchedAt time.Time) []models.HydroLevelReading {
	series, ok := history[waterLevelUUID]
	if !ok {
		return nil
	}
	out := make([]models.HydroLevelReading, 0, len(series.Values))
	for _, v := range series.Values {
		if v.BS == nil || v.Time == "" {
			continue
		}
		observedAt, err := emercit.ParseTime(v.Time)
		if err != nil {
			continue
		}
		out = append(out, models.HydroLevelReading{
			StationUUID:    stationUUID,
			WaterLevelUUID: waterLevelUUID,
			ObservedAt:     observedAt,
			LevelBSM:       float32(*v.BS),
			LevelZeroM:     emercit.Float32Ptr(v.Zero),
			FetchedAt:      fetchedAt,
		})
	}
	return out
}
