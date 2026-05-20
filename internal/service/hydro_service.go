package service

import (
	"context"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

type HydroService struct {
	repo           repository.HydroRepository
	stationUUID    string
	zeroPostBSM    float32
	hasZeroPostBSM bool
}

func NewHydroService(repo repository.HydroRepository, stationUUID string, zeroPostBSM float32) *HydroService {
	return &HydroService{repo: repo, stationUUID: stationUUID, zeroPostBSM: zeroPostBSM, hasZeroPostBSM: zeroPostBSM != 0}
}

func (s *HydroService) GetSnapshot(ctx context.Context, now time.Time) (*models.HydroSnapshot, error) {
	snap := &models.HydroSnapshot{}
	gauge, err := s.repo.GetGauge(ctx, s.stationUUID)
	if err != nil {
		return nil, err
	}
	snap.Gauge = gauge

	current, err := s.repo.GetLatest(ctx, s.stationUUID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return snap, nil
	}
	snap.Current = current
	snap.HasData = true
	if s.hasZeroPostBSM {
		v := (current.LevelBSM - s.zeroPostBSM) * 100
		snap.RelativeLevelCm = &v
	}

	previous, err := s.repo.GetPreviousBefore(ctx, s.stationUUID, current.ObservedAt)
	if err != nil {
		return nil, err
	}
	snap.Previous = previous
	if previous != nil {
		v := current.LevelBSM - previous.LevelBSM
		snap.ChangeM = &v
	}

	dayAgo, err := s.repo.GetNearBefore(ctx, s.stationUUID, current.ObservedAt.Add(-24*time.Hour), 2*time.Hour)
	if err != nil {
		return nil, err
	}
	snap.DayAgo = dayAgo
	if dayAgo != nil {
		v := current.LevelBSM - dayAgo.LevelBSM
		snap.Change24hM = &v
	}

	if gauge != nil {
		snap.Status = models.ClassifyHydroLevel(current.LevelBSM, gauge.FloodingPreventionBM, gauge.FloodingDangerBSM)
		if gauge.FloodingPreventionBM != nil {
			v := *gauge.FloodingPreventionBM - current.LevelBSM
			snap.ToPreventionM = &v
		}
		if gauge.FloodingDangerBSM != nil {
			v := *gauge.FloodingDangerBSM - current.LevelBSM
			snap.ToDangerM = &v
		}
	} else {
		snap.Status = models.HydroStatusUnknown
	}
	return snap, nil
}

func (s *HydroService) GetRange(ctx context.Context, from, to time.Time) ([]models.HydroLevelReading, error) {
	return s.repo.GetRange(ctx, s.stationUUID, from, to)
}

func (s *HydroService) GetGauge(ctx context.Context) (*models.HydroGauge, error) {
	return s.repo.GetGauge(ctx, s.stationUUID)
}
