package service

import (
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

func TestResolveArchivePeriod(t *testing.T) {
	loc := time.FixedZone("test", 3*60*60)
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, loc)
	tests := []struct {
		name               string
		period, month      string
		season, year       string
		from, to           string
		wantStart, wantEnd string
	}{
		{name: "month", period: "month", month: "2026-02", wantStart: "2026-02-01", wantEnd: "2026-03-01"},
		{name: "winter", period: "season", season: "2026-winter", wantStart: "2025-12-01", wantEnd: "2026-03-01"},
		{name: "year", period: "year", year: "2025", wantStart: "2025-01-01", wantEnd: "2026-01-01"},
		{name: "range", period: "range", from: "2026-04-10", to: "2026-04-18", wantStart: "2026-04-10", wantEnd: "2026-04-19"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, _, _, err := resolveArchivePeriod(tt.period, tt.month, tt.season, tt.year, tt.from, tt.to, now, loc)
			if err != nil {
				t.Fatalf("resolveArchivePeriod() error = %v", err)
			}
			if got := start.Format("2006-01-02"); got != tt.wantStart {
				t.Errorf("start = %s, want %s", got, tt.wantStart)
			}
			if got := end.Format("2006-01-02"); got != tt.wantEnd {
				t.Errorf("end = %s, want %s", got, tt.wantEnd)
			}
		})
	}
}

func TestResolveArchivePeriodRejectsLongRange(t *testing.T) {
	loc := time.UTC
	_, _, _, _, err := resolveArchivePeriod("range", "", "", "", "2024-01-01", "2025-01-02", time.Date(2026, 1, 1, 0, 0, 0, 0, loc), loc)
	if err != ErrInvalidArchiveRange {
		t.Fatalf("error = %v, want %v", err, ErrInvalidArchiveRange)
	}
}

func TestBuildArchiveSummaryUsesDailyValues(t *testing.T) {
	min1, avg1, max1, rain1, humidity1, pressure1 := float32(10), float32(15), float32(20), float32(2), int16(40), float32(750)
	min2, avg2, max2, rain2, humidity2, pressure2 := float32(5), float32(25), float32(30), float32(0), int16(60), float32(770)
	summary := buildArchiveSummary([]models.DailyWeatherInsight{
		{TempMin: &min1, TempAvg: &avg1, TempMax: &max1, RainTotal: &rain1, HumidityAvg: &humidity1, PressureAvg: &pressure1},
		{TempMin: &min2, TempAvg: &avg2, TempMax: &max2, RainTotal: &rain2, HumidityAvg: &humidity2, PressureAvg: &pressure2},
	}, 2)
	if !summary.HasTemp || summary.TempMin != 5 || summary.TempAvg != 20 || summary.TempMax != 30 {
		t.Fatalf("unexpected temperature summary: %#v", summary)
	}
	if !summary.HasRain || summary.RainTotal != 2 || summary.RainDays != 1 {
		t.Fatalf("unexpected rain summary: %#v", summary)
	}
	if !summary.HasAir || summary.HumidityAvg != 50 || summary.PressureAvg != 760 {
		t.Fatalf("unexpected air summary: %#v", summary)
	}
}
