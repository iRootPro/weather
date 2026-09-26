package web

import (
	"math"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

func TestHydroSignedChangeSuppressesRoundedZero(t *testing.T) {
	for _, format := range []string{"%.0f см", "%.0f см/ч"} {
		for _, value := range []float32{0, float32(math.Copysign(0, -1)), 0.01, -0.01, 0.49, -0.49} {
			if got := formatSignedFloat(value, format); got != formatSignedFloat(0, format) {
				t.Errorf("value %v, format %q: got %q, want unsigned zero", value, format, got)
			}
		}
	}
	for _, tc := range []struct {
		value float32
		want  string
	}{{0, "0 см"}, {0.51, "+1 см"}, {-0.51, "-1 см"}, {2, "+2 см"}, {-2, "-2 см"}} {
		if got := formatSignedFloat(tc.value, "%.0f см"); got != tc.want {
			t.Errorf("value %v: got %q, want %q", tc.value, got, tc.want)
		}
	}
}

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
