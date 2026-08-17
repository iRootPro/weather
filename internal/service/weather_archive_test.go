package service

import (
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

func TestArchivePeriodBounds(t *testing.T) {
	loc := time.FixedZone("test", 3*60*60)
	now := time.Date(2026, time.August, 17, 15, 0, 0, 0, loc)

	tests := []struct {
		period       string
		anchor       time.Time
		wantStart    string
		wantEnd      string
		wantPrevious string
	}{
		{"month", time.Date(2026, time.August, 1, 0, 0, 0, 0, loc), "2026-08-01", "2026-08-17", "2026-07-01"},
		{"season", time.Date(2026, time.July, 1, 0, 0, 0, 0, loc), "2026-06-01", "2026-08-17", "2026-03-01"},
		{"year", time.Date(2026, time.January, 1, 0, 0, 0, 0, loc), "2026-01-01", "2026-08-17", "2025-01-01"},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			start, end, previousStart, previousEnd, _, err := archivePeriodBounds(tt.period, tt.anchor, now, loc)
			if err != nil {
				t.Fatalf("archivePeriodBounds() error = %v", err)
			}
			if got := start.Format("2006-01-02"); got != tt.wantStart {
				t.Fatalf("start = %s, want %s", got, tt.wantStart)
			}
			if got := end.Format("2006-01-02"); got != tt.wantEnd {
				t.Fatalf("end = %s, want %s", got, tt.wantEnd)
			}
			if got := previousStart.Format("2006-01-02"); got != tt.wantPrevious {
				t.Fatalf("previous start = %s, want %s", got, tt.wantPrevious)
			}
			if got, want := calendarDaysInRange(previousStart, previousEnd, loc), calendarDaysInRange(start, end, loc); got != want {
				t.Fatalf("previous period spans %d calendar days, want %d", got, want)
			}
		})
	}
}

func TestBuildArchiveSummary(t *testing.T) {
	minOne, minTwo := float32(9), float32(11)
	avgOne, avgTwo := float32(15), float32(17)
	maxOne, maxTwo := float32(20), float32(24)
	rainOne, rainTwo := float32(1.2), float32(0.8)

	summary := buildArchiveSummary([]models.DailyWeatherInsight{
		{TempMin: &minOne, TempAvg: &avgOne, TempMax: &maxOne, RainTotal: &rainOne},
		{TempMin: &minTwo, TempAvg: &avgTwo, TempMax: &maxTwo, RainTotal: &rainTwo},
	}, 4)

	if summary.TempAvg == nil || *summary.TempAvg != 16 {
		t.Fatalf("temp avg = %v, want 16", summary.TempAvg)
	}
	if summary.TempMin == nil || *summary.TempMin != 9 || summary.TempMax == nil || *summary.TempMax != 24 {
		t.Fatalf("temperature range = %v..%v, want 9..24", summary.TempMin, summary.TempMax)
	}
	if summary.RainTotal < 1.99 || summary.RainTotal > 2.01 {
		t.Fatalf("rain total = %v, want 2", summary.RainTotal)
	}
	if summary.DaysWithData != 2 || summary.CoveragePercent != 50 {
		t.Fatalf("coverage = %d/%d%%, want 2/50%%", summary.DaysWithData, summary.CoveragePercent)
	}
}
