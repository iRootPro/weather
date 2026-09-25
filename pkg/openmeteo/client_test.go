package openmeteo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetForecastRequestsWindInKilometersPerHour(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("wind_speed_unit"), "kmh"; got != want {
			t.Errorf("wind_speed_unit = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hourly":{},"daily":{}}`))
	}))
	defer server.Close()

	client := NewClient(time.Second)
	client.baseURL = server.URL
	if _, err := client.GetForecast(context.Background(), ForecastRequest{Latitude: 1, Longitude: 2, Timezone: "Europe/Moscow"}); err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}
}
