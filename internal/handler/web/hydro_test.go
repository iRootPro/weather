package web

import (
	"testing"

	"github.com/iRootPro/weather/internal/models"
)

func TestBuildWaterLevelMiniUsesCalculatedSnapshotTrend(t *testing.T) {
	sourceIndex := float32(1.407)
	calculatedRate := float32(0)
	snapshot := &models.HydroSnapshot{
		Current: &models.HydroLevelReading{
			LevelBSM:     220.280,
			SourceHDIIHR: &sourceIndex,
		},
		ChangeCmPerHour: &calculatedRate,
		HasData:         true,
	}

	mini := buildWaterLevelMini(snapshot)
	if mini == nil {
		t.Fatal("buildWaterLevelMini returned nil")
	}
	if mini.ChangeText != "0 см/ч" {
		t.Fatalf("ChangeText = %q, want calculated 0 см/ч", mini.ChangeText)
	}
	if mini.TrendText != "стабильно →" {
		t.Fatalf("TrendText = %q, want stable calculated trend", mini.TrendText)
	}
}
