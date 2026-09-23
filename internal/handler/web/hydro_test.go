package web

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

func TestBuildCachedWaterLevelCardReusesFreshSnapshot(t *testing.T) {
	h := &Handler{
		hydroCardCache: WaterLevelCardData{
			HasData:       true,
			LevelM:        161.681,
			DayChangeText: "+5 см",
		},
		hydroCardCachedAt: time.Now(),
	}

	card := h.buildCachedWaterLevelCard(httptest.NewRequest("GET", "/widgets/current", nil))
	if !card.HasData || card.LevelM != 161.681 || card.DayChangeText != "+5 см" {
		t.Fatalf("fresh cached water card = %+v, want cached snapshot", card)
	}
}

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
