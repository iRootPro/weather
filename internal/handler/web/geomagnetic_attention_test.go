package web

import (
	"math"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

func TestGeomagneticHeaderDemoRendersObservedSignal(t *testing.T) {
	demo, err := NewAttentionDemo("../../web/templates")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ scenario, title, severity string }{
		{"geomagnetic", "Геомагнитное возмущение", "warning"},
		{"storm", "Магнитная буря G3", "danger"},
		{"stale-geomagnetic", "Геомагнитное возмущение", "warning"},
		{"six", "Магнитная буря G3", "danger"},
	} {
		response := httptest.NewRecorder()
		demo.ServeHTTP(response, httptest.NewRequest("GET", "/?scenario="+tc.scenario, nil))
		if response.Code != 200 {
			t.Fatal(response.Code)
		}
		body := response.Body.String()
		if tc.scenario == "six" && (strings.Count(body, "data-attention-trigger aria-label=") != 6 || !strings.Contains(body, "Важных сигналов: 6")) {
			t.Fatal("six-signal scenario should render six icons and their combined notice")
		}
		for _, text := range []string{`aria-describedby="attention-hint-geomagnetic"`, `data-icon="magnet"`, "ui-attention-trigger-" + tc.severity, tc.title, "наблюдение"} {
			if !strings.Contains(body, text) {
				t.Errorf("%s missing %s", tc.scenario, text)
			}
		}
	}
}

func TestGeomagneticSignalThresholds(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		kp              float32
		severity, title string
	}{
		{3.9, "", ""}, {4, "warning", "Геомагнитное возмущение"}, {4.9, "warning", "Геомагнитное возмущение"},
		{5, "danger", "Магнитная буря G1"}, {6, "danger", "Магнитная буря G2"}, {7, "danger", "Магнитная буря G3"},
		{8, "danger", "Магнитная буря G4"}, {9, "danger", "Магнитная буря G5"},
	} {
		got := buildGeomagneticSignal(&models.GeomagneticKp{SlotTime: now.Add(-time.Hour), Kp: tc.kp}, now)
		if got.Severity != tc.severity || got.Title != tc.title {
			t.Fatalf("Kp %v: %+v", tc.kp, got)
		}
		if tc.title != "" && (got.Icon() != "magnet" || !strings.Contains(got.Detail, "Kp ")) {
			t.Fatalf("missing hint context: %+v", got)
		}
	}
}

func TestGeomagneticSignalRejectsUnavailableOrForecastData(t *testing.T) {
	now := time.Now()
	for _, current := range []*models.GeomagneticKp{
		nil, {Kp: 5}, {SlotTime: now.Add(time.Second), Kp: 5},
		{SlotTime: now.Add(-6 * time.Hour), Kp: 5}, {SlotTime: now, Kp: 5, IsForecast: true},
		{SlotTime: now, Kp: 10}, {SlotTime: now, Kp: float32(math.NaN())}, {SlotTime: now, Kp: float32(math.Inf(1))},
	} {
		if got := buildGeomagneticSignal(current, now); got.Title != "" {
			t.Fatalf("unexpected signal for %+v: %+v", current, got)
		}
	}
	if buildGeomagneticSignal(&models.GeomagneticKp{SlotTime: now.Add(-6*time.Hour + time.Second), Kp: 4}, now).Title == "" {
		t.Fatal("last second in six-hour window should be accepted")
	}
}

func TestGeomagneticSignalIndependentOfStationFreshness(t *testing.T) {
	now := time.Now()
	kp := buildGeomagneticSignal(&models.GeomagneticKp{SlotTime: now, Kp: 4.3}, now)
	temp := float32(36)
	stale := buildCurrentAttention(&models.WeatherData{Time: now.Add(-30 * time.Minute), TempOutdoor: &temp}, nil, now).withGeomagnetic(kp)
	if len(stale.Signals()) != 1 || stale.Signals()[0].Icon() != "magnet" || !strings.Contains(stale.Notice, "30 мин") {
		t.Fatalf("stale station suppressed Kp or lost notice: %+v", stale)
	}
	fresh := buildCurrentAttention(&models.WeatherData{Time: now, TempOutdoor: &temp}, nil, now).withGeomagnetic(kp)
	if len(fresh.Signals()) != 2 || !strings.Contains(fresh.Notice, "Важных сигналов: 2") {
		t.Fatalf("combined count: %+v", fresh)
	}
}
