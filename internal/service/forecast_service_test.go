package service

import (
	"context"
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

type forecastRepositoryStub struct {
	dailyFrom time.Time
}

func (r *forecastRepositoryStub) SaveHourly(context.Context, *models.ForecastData) error { return nil }
func (r *forecastRepositoryStub) SaveDaily(context.Context, *models.ForecastData) error  { return nil }
func (r *forecastRepositoryStub) SaveBatch(context.Context, []models.ForecastData) error { return nil }
func (r *forecastRepositoryStub) GetHourlyForecast(context.Context, time.Time, time.Time) ([]models.ForecastData, error) {
	return nil, nil
}
func (r *forecastRepositoryStub) GetDailyForecast(_ context.Context, from, _ time.Time) ([]models.ForecastData, error) {
	r.dailyFrom = from
	return nil, nil
}
func (r *forecastRepositoryStub) GetLatestHourly(context.Context, int) ([]models.ForecastData, error) {
	return nil, nil
}
func (r *forecastRepositoryStub) GetLatestDaily(context.Context, int) ([]models.ForecastData, error) {
	return nil, nil
}
func (r *forecastRepositoryStub) DeleteOldForecasts(context.Context, time.Time) error { return nil }

func TestForecastConversionsPreserveFetchedAtForWidgetFreshness(t *testing.T) {
	fetchedAt := time.Date(2026, time.September, 25, 8, 15, 0, 0, time.UTC)
	data := []models.ForecastData{{
		ForecastTime: fetchedAt.Add(time.Hour),
		FetchedAt:    fetchedAt,
	}}

	hourly := convertToHourlyForecast(data, time.UTC)
	if len(hourly) != 1 || !hourly[0].FetchedAt.Equal(fetchedAt) {
		t.Fatalf("hourly FetchedAt = %v, want %v", hourly[0].FetchedAt, fetchedAt)
	}

	daily := convertToDailyForecast(data, time.UTC)
	if len(daily) != 1 || !daily[0].FetchedAt.Equal(fetchedAt) {
		t.Fatalf("daily FetchedAt = %v, want %v", daily[0].FetchedAt, fetchedAt)
	}
}

func TestForecastConversionsPreserveOptionalForecastValues(t *testing.T) {
	windGusts, precipitation, uv := float32(57.6), float32(0), float32(7)
	data := []models.ForecastData{{WindGusts: &windGusts, Precipitation: &precipitation, UVIndex: &uv}}

	hourly := convertToHourlyForecast(data, time.UTC)[0]
	if !hourly.HasWindGusts || hourly.WindGusts != 16 || !hourly.HasPrecipitation {
		t.Errorf("hourly optional values = %#v", hourly)
	}
	if hourly.HasFeelsLike || hourly.HasWindSpeed {
		t.Errorf("hourly absent values became present: %#v", hourly)
	}

	daily := convertToDailyForecast(data, time.UTC)[0]
	if !daily.HasPrecipitationSum || !daily.HasWindGustsMax || !daily.HasUVIndexMax || daily.UVIndexMax != uv {
		t.Errorf("daily optional values = %#v", daily)
	}
}

func TestForecastConversionsNormalizeTimestampToConfiguredTimezone(t *testing.T) {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	data := []models.ForecastData{{ForecastTime: time.Date(2026, 9, 24, 21, 0, 0, 0, time.UTC), FetchedAt: time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)}}
	if got := convertToHourlyForecast(data, location)[0].Time; got.Hour() != 0 || got.Day() != 25 || got.Location() != location {
		t.Errorf("hourly time = %s in %s, want 25th midnight Moscow", got, got.Location())
	}
	if got := convertToDailyForecast(data, location)[0].Date; got.Hour() != 0 || got.Day() != 25 || got.Location() != location {
		t.Errorf("daily date = %s in %s, want 25th midnight Moscow", got, got.Location())
	}
	if got := convertToHourlyForecast(data, location)[0].FetchedAt; got.Hour() != 12 || got.Location() != location {
		t.Errorf("fetched at = %s in %s, want 12:00 Moscow", got, got.Location())
	}
}

func TestDailyForecastUsesConfiguredLocalDayBoundary(t *testing.T) {
	repo := &forecastRepositoryStub{}
	service := NewForecastService(repo)
	service.SetTimezone("Europe/Moscow")
	if _, err := service.GetDailyForecast(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if got := repo.dailyFrom; got.Location().String() != "Europe/Moscow" || got.Hour() != 0 || got.Minute() != 0 {
		t.Errorf("daily query start = %s in %s, want local midnight", got, got.Location())
	}
}
