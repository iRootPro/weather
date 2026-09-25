package service

import (
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

func TestForecastConversionsPreserveFetchedAtForWidgetFreshness(t *testing.T) {
	fetchedAt := time.Date(2026, time.September, 25, 8, 15, 0, 0, time.UTC)
	data := []models.ForecastData{{
		ForecastTime: fetchedAt.Add(time.Hour),
		FetchedAt:    fetchedAt,
	}}

	hourly := convertToHourlyForecast(data)
	if len(hourly) != 1 || !hourly[0].FetchedAt.Equal(fetchedAt) {
		t.Fatalf("hourly FetchedAt = %v, want %v", hourly[0].FetchedAt, fetchedAt)
	}

	daily := convertToDailyForecast(data)
	if len(daily) != 1 || !daily[0].FetchedAt.Equal(fetchedAt) {
		t.Fatalf("daily FetchedAt = %v, want %v", daily[0].FetchedAt, fetchedAt)
	}
}
