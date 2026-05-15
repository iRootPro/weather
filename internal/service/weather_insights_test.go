package service

import (
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

func TestSeasonBounds(t *testing.T) {
	loc := time.FixedZone("test", 3*60*60)
	tests := []struct {
		name      string
		date      time.Time
		wantStart string
		wantEnd   string
	}{
		{"winter january crosses year", time.Date(2026, time.January, 15, 12, 0, 0, 0, loc), "2025-12-01", "2026-03-01"},
		{"spring", time.Date(2026, time.May, 15, 12, 0, 0, 0, loc), "2026-03-01", "2026-06-01"},
		{"summer", time.Date(2026, time.July, 1, 12, 0, 0, 0, loc), "2026-06-01", "2026-09-01"},
		{"autumn", time.Date(2026, time.November, 30, 12, 0, 0, 0, loc), "2026-09-01", "2026-12-01"},
		{"winter december", time.Date(2026, time.December, 2, 12, 0, 0, 0, loc), "2026-12-01", "2027-03-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := seasonBounds(tt.date, loc)
			if got := start.Format("2006-01-02"); got != tt.wantStart {
				t.Fatalf("start = %s, want %s", got, tt.wantStart)
			}
			if got := end.Format("2006-01-02"); got != tt.wantEnd {
				t.Fatalf("end = %s, want %s", got, tt.wantEnd)
			}
		})
	}
}

func TestClassifyDayTypePriority(t *testing.T) {
	rain := float32(15)
	gust := float32(20)
	temp := float32(33)
	avg := float32(22)
	solar := float32(600)

	day := models.DailyWeatherInsight{
		RainTotal:         &rain,
		WindGustMax:       &gust,
		TempMax:           &temp,
		TempAvg:           &avg,
		SolarRadiationMax: &solar,
	}
	if got := classifyDayType(day); got != "storm" {
		t.Fatalf("heavy rain should win priority, got %s", got)
	}

	rain = 0
	if got := classifyDayType(day); got != "windy" {
		t.Fatalf("strong wind should win after rain, got %s", got)
	}

	gust = 3
	if got := classifyDayType(day); got != "hot" {
		t.Fatalf("hot day should win after wind, got %s", got)
	}
}

func TestBuildDayTypeSummariesDominant(t *testing.T) {
	loc := time.FixedZone("test", 3*60*60)
	rain := float32(0)
	avg := float32(21)
	max := float32(24)
	gust := float32(2)
	solar := float32(300)
	heavyRain := float32(12)

	days := []models.DailyWeatherInsight{
		{Date: time.Date(2026, time.May, 1, 0, 0, 0, 0, loc), RainTotal: &rain, TempAvg: &avg, TempMax: &max, WindGustMax: &gust, SolarRadiationMax: &solar},
		{Date: time.Date(2026, time.May, 2, 0, 0, 0, 0, loc), RainTotal: &rain, TempAvg: &avg, TempMax: &max, WindGustMax: &gust, SolarRadiationMax: &solar},
		{Date: time.Date(2026, time.May, 3, 0, 0, 0, 0, loc), RainTotal: &heavyRain, TempAvg: &avg, TempMax: &max, WindGustMax: &gust, SolarRadiationMax: &solar},
	}

	_, dominant := buildDayTypeSummaries(days)
	if dominant.Code != "comfortable" {
		t.Fatalf("dominant = %s, want comfortable", dominant.Code)
	}
	if dominant.Count != 2 {
		t.Fatalf("dominant count = %d, want 2", dominant.Count)
	}
}

func TestBuildSameMonthBenchmarkUsesOnlySameMonthAndDate(t *testing.T) {
	loc := time.FixedZone("test", 3*60*60)
	now := time.Date(2026, time.May, 15, 12, 0, 0, 0, loc)
	current := models.MonthlyWeatherInsights{RainTotal: 30, RainDays: 3, AvgTemp: 20, DaysWithData: 15, DaysInPeriod: 15}

	archive := make([]models.DailyWeatherInsight, 0)
	for year, rain := range map[int]float32{2024: 1, 2025: 3} {
		for day := 1; day <= 15; day++ {
			value := rain
			archive = append(archive, models.DailyWeatherInsight{
				Date:      time.Date(year, time.May, day, 0, 0, 0, 0, loc),
				RainTotal: &value,
			})
		}
	}
	ignoredRain := float32(100)
	archive = append(archive,
		models.DailyWeatherInsight{Date: time.Date(2025, time.April, 10, 0, 0, 0, 0, loc), RainTotal: &ignoredRain},
		models.DailyWeatherInsight{Date: time.Date(2025, time.May, 20, 0, 0, 0, 0, loc), RainTotal: &ignoredRain},
	)

	benchmark := buildSameMonthBenchmark(now, current, archive, loc)
	if !benchmark.Available {
		t.Fatal("benchmark should be available")
	}
	if benchmark.SampleSize != 2 {
		t.Fatalf("sample size = %d, want 2", benchmark.SampleSize)
	}
	if benchmark.RainTotalAvg != 30 {
		t.Fatalf("rain avg = %.1f, want 30.0", benchmark.RainTotalAvg)
	}
	if benchmark.RainRatioPercent != 100 {
		t.Fatalf("rain ratio = %d, want 100", benchmark.RainRatioPercent)
	}
}
