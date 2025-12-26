package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/iRootPro/weather/internal/service"
)

type WeatherHandler struct {
	weatherService *service.WeatherService
}

func NewWeatherHandler(weatherService *service.WeatherService) *WeatherHandler {
	return &WeatherHandler{weatherService: weatherService}
}

// GET /api/weather/current
func (h *WeatherHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	data, err := h.weatherService.GetCurrent(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, data)
}

// GET /api/weather/history?from=2024-12-01&to=2024-12-24&interval=1h
func (h *WeatherHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseTimeRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1h"
	}

	data, err := h.weatherService.GetHistory(r.Context(), from, to, interval)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, data)
}

// GET /api/weather/stats?period=day|week|month|year
func (h *WeatherHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "day"
	}

	stats, err := h.weatherService.GetStats(r.Context(), period)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, stats)
}

// GET /api/weather/chart?from=...&to=...&interval=1h&fields=temp_outdoor,humidity_outdoor
func (h *WeatherHandler) GetChartData(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseTimeRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1h"
	}

	fieldsStr := r.URL.Query().Get("fields")
	if fieldsStr == "" {
		fieldsStr = "temp_outdoor,humidity_outdoor,pressure_relative"
	}
	fields := strings.Split(fieldsStr, ",")

	data, err := h.weatherService.GetChartData(r.Context(), from, to, interval, fields)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, data)
}

// GET /api/weather/events?hours=24
func (h *WeatherHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	hours := 24 // default
	if hoursStr := r.URL.Query().Get("hours"); hoursStr != "" {
		if h, err := time.ParseDuration(hoursStr + "h"); err == nil {
			hours = int(h.Hours())
		}
	}

	events, err := h.weatherService.GetRecentEvents(r.Context(), hours)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, events)
}

func parseTimeRange(r *http.Request) (time.Time, time.Time, error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var from, to time.Time
	var err error

	if fromStr == "" {
		from = time.Now().AddDate(0, 0, -1) // по умолчанию последние 24 часа
	} else {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			from, err = time.Parse(time.RFC3339, fromStr)
			if err != nil {
				return time.Time{}, time.Time{}, err
			}
		}
	}

	if toStr == "" {
		to = time.Now()
	} else {
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			to, err = time.Parse(time.RFC3339, toStr)
			if err != nil {
				return time.Time{}, time.Time{}, err
			}
		}
		// Если указана только дата, добавляем конец дня
		if toStr == to.Format("2006-01-02") {
			to = to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
	}

	return from, to, nil
}

func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
