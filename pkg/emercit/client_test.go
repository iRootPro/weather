package emercit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetWaterLevelHistoryReloginsAfterUnauthorized(t *testing.T) {
	loginRequests := 0
	historyRequests := 0
	const waterLevelUUID = "water-level"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/token/":
			loginRequests++
			_, _ = fmt.Fprintf(w, `{"access":"token-%d","refresh":"refresh"}`, loginRequests)
		case "/api/mchs/waterlevel/" + waterLevelUUID + "/":
			historyRequests++
			if historyRequests == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"detail":"token expired"}`))
				return
			}
			if got := r.Header.Get("Authorization"); got != "Bearer token-2" {
				t.Errorf("Authorization = %q, want refreshed token", got)
			}
			_, _ = fmt.Fprintf(w, `{"%s":{"values":[{"time":"2026-08-31T11:40:00Z","bs":220.28,"zero":null}]}}`, waterLevelUUID)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(5*time.Second, server.URL, "user", "password")
	history, err := client.GetWaterLevelHistory(
		context.Background(),
		waterLevelUUID,
		time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("GetWaterLevelHistory: %v", err)
	}
	if loginRequests != 2 {
		t.Fatalf("login requests = %d, want 2", loginRequests)
	}
	if historyRequests != 2 {
		t.Fatalf("history requests = %d, want 2", historyRequests)
	}
	if got := len(history[waterLevelUUID].Values); got != 1 {
		t.Fatalf("history values = %d, want 1", got)
	}
}

func TestGetWaterLevelHistoryRetriesAuthorizationOnlyOnce(t *testing.T) {
	loginRequests := 0
	historyRequests := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/token/":
			loginRequests++
			_, _ = fmt.Fprintf(w, `{"access":"token-%d","refresh":"refresh"}`, loginRequests)
		case "/api/mchs/waterlevel/water-level/":
			historyRequests++
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"detail":"still forbidden"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(5*time.Second, server.URL, "user", "password")
	_, err := client.GetWaterLevelHistory(
		context.Background(),
		"water-level",
		time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("GetWaterLevelHistory returned nil error after persistent 403")
	}
	if loginRequests != 2 {
		t.Fatalf("login requests = %d, want exactly 2", loginRequests)
	}
	if historyRequests != 2 {
		t.Fatalf("history requests = %d, want exactly 2", historyRequests)
	}
}
