package service

import (
	"context"
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

type hydroRepositoryStub struct {
	latest   *models.HydroLevelReading
	previous *models.HydroLevelReading
}

func (r *hydroRepositoryStub) SaveGauge(context.Context, *models.HydroGauge) error { return nil }
func (r *hydroRepositoryStub) SaveReadingsBatch(context.Context, []models.HydroLevelReading) error {
	return nil
}
func (r *hydroRepositoryStub) GetGauge(context.Context, string) (*models.HydroGauge, error) {
	return nil, nil
}
func (r *hydroRepositoryStub) GetLatest(context.Context, string) (*models.HydroLevelReading, error) {
	return r.latest, nil
}
func (r *hydroRepositoryStub) GetPreviousBefore(_ context.Context, _, waterLevelUUID string, _ time.Time) (*models.HydroLevelReading, error) {
	if r.previous == nil || r.previous.WaterLevelUUID != waterLevelUUID {
		return nil, nil
	}
	return r.previous, nil
}
func (r *hydroRepositoryStub) GetNearBefore(context.Context, string, time.Time, time.Duration) (*models.HydroLevelReading, error) {
	return nil, nil
}
func (r *hydroRepositoryStub) GetRange(context.Context, string, time.Time, time.Time) ([]models.HydroLevelReading, error) {
	return nil, nil
}
func (r *hydroRepositoryStub) DeleteOlderThan(context.Context, time.Time) error { return nil }

func TestHydroSnapshotUsesObservedLevelsForFlatTrend(t *testing.T) {
	observedAt := time.Date(2026, time.August, 31, 14, 40, 0, 0, time.UTC)
	sourceIndex := float32(1.407)
	repo := &hydroRepositoryStub{
		latest: &models.HydroLevelReading{
			ObservedAt:   observedAt,
			LevelBSM:     220.280,
			SourceHDIIHR: &sourceIndex,
		},
		previous: &models.HydroLevelReading{
			ObservedAt: observedAt.Add(-10 * time.Minute),
			LevelBSM:   220.280,
		},
	}

	snapshot, err := NewHydroService(repo, "malamino", 0).GetSnapshot(context.Background(), observedAt)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snapshot.ChangeCmPerHour == nil {
		t.Fatal("ChangeCmPerHour is nil, want calculated zero")
	}
	if got := *snapshot.ChangeCmPerHour; got != 0 {
		t.Fatalf("ChangeCmPerHour = %.3f, want 0 from unchanged observed levels", got)
	}
}

func TestHydroSnapshotCalculatesHourlyChangeFromObservationInterval(t *testing.T) {
	observedAt := time.Date(2026, time.August, 31, 14, 40, 0, 0, time.UTC)
	sourceIndex := float32(1.259)
	repo := &hydroRepositoryStub{
		latest: &models.HydroLevelReading{
			ObservedAt:   observedAt,
			LevelBSM:     161.852538,
			SourceHDIIHR: &sourceIndex,
		},
		previous: &models.HydroLevelReading{
			ObservedAt: observedAt.Add(-10 * time.Minute),
			LevelBSM:   161.846481,
		},
	}

	snapshot, err := NewHydroService(repo, "armavir", 0).GetSnapshot(context.Background(), observedAt)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snapshot.ChangeCmPerHour == nil {
		t.Fatal("ChangeCmPerHour is nil, want calculated rate")
	}
	if got := *snapshot.ChangeCmPerHour; got < 3.62 || got > 3.65 {
		t.Fatalf("ChangeCmPerHour = %.3f, want approximately 3.635", got)
	}
}

func TestHydroSnapshotDoesNotCompareDifferentWaterLevelSeries(t *testing.T) {
	observedAt := time.Date(2026, time.August, 31, 14, 40, 0, 0, time.UTC)
	repo := &hydroRepositoryStub{
		latest: &models.HydroLevelReading{
			WaterLevelUUID: "new-sensor",
			ObservedAt:     observedAt,
			LevelBSM:       220.280,
		},
		previous: &models.HydroLevelReading{
			WaterLevelUUID: "old-sensor",
			ObservedAt:     observedAt.Add(-10 * time.Minute),
			LevelBSM:       161.846,
		},
	}

	snapshot, err := NewHydroService(repo, "station", 0).GetSnapshot(context.Background(), observedAt)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snapshot.ChangeCmPerHour != nil {
		t.Fatalf("ChangeCmPerHour = %.3f, want nil across different water-level series", *snapshot.ChangeCmPerHour)
	}
}
