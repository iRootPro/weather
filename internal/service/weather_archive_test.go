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
		name                  string
		period, month, season string
		year, from, to        string
		wantStart, wantEnd    string
	}{
		{name: "month", period: "month", month: "2026-02", wantStart: "2026-02-01", wantEnd: "2026-03-01"},
		{name: "winter", period: "season", season: "2026-winter", wantStart: "2025-12-01", wantEnd: "2026-03-01"},
		{name: "year", period: "year", year: "2025", wantStart: "2025-01-01", wantEnd: "2026-01-01"},
		{name: "range", period: "range", from: "2026-04-10", to: "2026-04-18", wantStart: "2026-04-10", wantEnd: "2026-04-19"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, _, err := resolveArchivePeriod(tt.period, tt.month, tt.season, tt.year, tt.from, tt.to, now, loc)
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
	_, _, _, err := resolveArchivePeriod("range", "", "", "", "2024-01-01", "2025-01-02", time.Date(2026, 1, 1, 0, 0, 0, 0, loc), loc)
	if err != ErrInvalidArchiveRange {
		t.Fatalf("error = %v, want %v", err, ErrInvalidArchiveRange)
	}
}

func TestArchiveOptionsOnlyContainYearsAndSeasonsWithData(t *testing.T) {
	loc := time.UTC
	days := []models.DailyWeatherInsight{
		{Date: time.Date(2026, time.January, 10, 0, 0, 0, 0, loc)},
		{Date: time.Date(2026, time.April, 10, 0, 0, 0, 0, loc)},
		{Date: time.Date(2026, time.August, 10, 0, 0, 0, 0, loc)},
	}
	years := archiveYearOptions(days, time.Date(2026, time.August, 17, 0, 0, 0, 0, loc))
	if len(years) != 1 || years[0] != 2026 {
		t.Fatalf("years = %v, want [2026]", years)
	}
	seasons := archiveSeasonOptions(days, time.Date(2026, time.August, 17, 0, 0, 0, 0, loc), loc)
	if len(seasons) != 3 || seasons[0].Value != "2026-summer" || seasons[2].Value != "2026-winter" {
		t.Fatalf("unexpected seasons = %#v", seasons)
	}
}

func TestBuildArchiveEventsAndMetricFilter(t *testing.T) {
	hot, cold, rain, gust, solar := float32(35), float32(-4), float32(12), float32(18), float32(800)
	date := time.Date(2026, time.July, 12, 0, 0, 0, 0, time.UTC)
	events := buildArchiveEvents([]models.DailyWeatherInsight{{Date: date, TempMax: &hot, TempMin: &cold, RainTotal: &rain, WindGustMax: &gust, SolarRadiationMax: &solar}})
	if len(events) != 5 || events[0].Title != "Самый жаркий день" || events[4].Unit != "Вт/м²" {
		t.Fatalf("unexpected events: %#v", events)
	}
	temperatureEvents := filterArchiveEvents(events, "temperature")
	if len(temperatureEvents) != 2 || temperatureEvents[0].Group != "temperature" {
		t.Fatalf("unexpected temperature events: %#v", temperatureEvents)
	}
}

func TestArchiveDaySearchFiltersMatchingMeasurements(t *testing.T) {
	max30, max25, rain5 := float32(30), float32(25), float32(5)
	search, err := resolveArchiveDaySearch("temp_max", "gte", "28")
	if err != nil {
		t.Fatalf("resolveArchiveDaySearch() error = %v", err)
	}
	matches := filterArchiveDays([]models.DailyWeatherInsight{{TempMax: &max30}, {TempMax: &max25}, {RainTotal: &rain5}}, search)
	if len(matches) != 1 || matches[0].TempMax == nil || *matches[0].TempMax != 30 {
		t.Fatalf("unexpected matches: %#v", matches)
	}
	if search.Description != "максимальная температура ≥ 28 °C" {
		t.Fatalf("unexpected description: %q", search.Description)
	}
}

func TestArchiveDaySearchRejectsInvalidInput(t *testing.T) {
	if _, err := resolveArchiveDaySearch("rain", "gte", "many"); err != ErrInvalidArchiveSearch {
		t.Fatalf("error = %v, want %v", err, ErrInvalidArchiveSearch)
	}
	if _, err := resolveArchiveDaySearch("unknown", "gte", "1"); err != ErrInvalidArchiveSearch {
		t.Fatalf("error = %v, want %v", err, ErrInvalidArchiveSearch)
	}
}

func TestBuildArchiveCoverageFindsConsecutiveMissingDays(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, time.August, 1, 0, 0, 0, 0, loc)
	periodDays := []models.DailyWeatherInsight{
		{Date: start},
		{Date: start.AddDate(0, 0, 3)},
		{Date: start.AddDate(0, 0, 4)},
	}
	coverage := buildArchiveCoverage(start, start.AddDate(0, 0, 5), periodDays, periodDays, loc)
	if coverage.ExpectedDays != 5 || coverage.CoveredDays != 3 || coverage.MissingDays != 2 {
		t.Fatalf("unexpected coverage: %#v", coverage)
	}
	if len(coverage.Gaps) != 1 || coverage.Gaps[0].Days != 2 || coverage.LongestGapDays != 2 {
		t.Fatalf("unexpected gaps: %#v", coverage.Gaps)
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
