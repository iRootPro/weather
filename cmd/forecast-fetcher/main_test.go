package main

import (
	"testing"
	"time"
)

func TestForecastTimesUseRequestedTimezone(t *testing.T) {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}

	hourly, err := parseForecastTime("2026-09-25T00:30", "2006-01-02T15:04", location)
	if err != nil {
		t.Fatal(err)
	}
	daily, err := parseForecastTime("2026-09-26", "2006-01-02", location)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := hourly.UTC(), time.Date(2026, 9, 24, 21, 30, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("hourly UTC = %s, want %s", got, want)
	}
	if got, want := daily.Hour(), 0; got != want || daily.Location() != location {
		t.Errorf("daily = %s in %s, want midnight in requested timezone", daily, daily.Location())
	}
}
