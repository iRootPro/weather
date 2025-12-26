package web

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

// DetailTemperature renders the detailed temperature page
func (h *Handler) DetailTemperature(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current data with hourly change
	current, hourAgo, dailyMinMax, err := h.weatherService.GetCurrentWithHourlyChange(ctx)
	if err != nil {
		slog.Error("failed to get current weather", "error", err)
		http.Error(w, "Failed to load weather data", http.StatusInternalServerError)
		return
	}

	// Get data from 24 hours ago for daily change
	dayAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		slog.Warn("failed to get day ago data", "error", err)
	}

	// Get data from week ago for weekly change
	weekAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		slog.Warn("failed to get week ago data", "error", err)
	}

	// Get records
	records, err := h.weatherService.GetRecords(ctx)
	if err != nil {
		slog.Error("failed to get records", "error", err)
		http.Error(w, "Failed to load records", http.StatusInternalServerError)
		return
	}

	// Get chart data for different periods
	now := time.Now()

	// 24 hours chart
	chart24h, err := h.weatherService.GetHistory(ctx, now.Add(-24*time.Hour), now, "5min")
	if err != nil {
		slog.Error("failed to get 24h chart data", "error", err)
	}

	// 7 days chart
	chart7d, err := h.weatherService.GetHistory(ctx, now.Add(-7*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 7d chart data", "error", err)
	}

	// 30 days chart
	chart30d, err := h.weatherService.GetHistory(ctx, now.Add(-30*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 30d chart data", "error", err)
	}

	// Prepare template data
	templateData := struct {
		ActivePage string
		Data       interface{}
	}{
		ActivePage: "dashboard",
		Data: map[string]interface{}{
			// Current readings
			"Current":     getFloat32Value(current.TempOutdoor),
			"FeelsLike":   getFloat32Value(current.TempFeelsLike),
			"DewPoint":    getFloat32Value(current.DewPoint),
			"IndoorTemp":  getFloat32Value(current.TempIndoor),
			"UpdateTime":  current.Time.Format("15:04"),
			"UpdateDate":  current.Time.Format("2 января 2006"),

			// Calculate differences
			"IndoorDiff": calculateDiff(current.TempOutdoor, current.TempIndoor),

			// Changes
			"ChangeHour":  calculateTempChange(current.TempOutdoor, hourAgo),
			"ChangeDay":   calculateTempChange(current.TempOutdoor, dayAgo),
			"ChangeWeek":  calculateTempChange(current.TempOutdoor, weekAgo),
			"HasHourData": hourAgo != nil && hourAgo.TempOutdoor != nil,
			"HasDayData":  dayAgo != nil && dayAgo.TempOutdoor != nil,
			"HasWeekData": weekAgo != nil && weekAgo.TempOutdoor != nil,

			// Today's stats
			"TodayMin":       getFloat32Value(dailyMinMax.TempMin),
			"TodayMax":       getFloat32Value(dailyMinMax.TempMax),
			"TodayAvg":       calculateAvgFromMinMax(dailyMinMax.TempMin, dailyMinMax.TempMax),
			"TodayAmplitude": calculateAmplitude(dailyMinMax.TempMin, dailyMinMax.TempMax),
			"HasDailyData":   dailyMinMax != nil,

			// Records
			"RecordMax":     records.TempOutdoorMax.Value,
			"RecordMaxTime": records.TempOutdoorMax.Time.Format("2 января 2006, 15:04"),
			"RecordMin":     records.TempOutdoorMin.Value,
			"RecordMinTime": records.TempOutdoorMin.Time.Format("2 января 2006, 15:04"),

			// Chart data (as JSON)
			"Chart24h":  toJSON(prepareChartData(chart24h, "TempOutdoor")),
			"Chart7d":   toJSON(prepareChartData(chart7d, "TempOutdoor")),
			"Chart30d":  toJSON(prepareChartData(chart30d, "TempOutdoor")),
			"HasCharts": len(chart24h) > 0,
		},
	}

	tmpl, err := h.parseTemplate("detail/temperature.html")
	if err != nil {
		slog.Error("failed to parse temperature detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render temperature detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Helper functions

func getFloat32Value(ptr *float32) float32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func calculateDiff(a, b *float32) float32 {
	if a == nil || b == nil {
		return 0
	}
	return *a - *b
}

func calculateTempChange(current *float32, oldData *models.WeatherData) float32 {
	if current == nil || oldData == nil || oldData.TempOutdoor == nil {
		return 0
	}
	return *current - *oldData.TempOutdoor
}

func calculateAvgFromMinMax(min, max *float32) float32 {
	if min == nil || max == nil {
		return 0
	}
	return (*min + *max) / 2
}

func calculateAmplitude(min, max *float32) float32 {
	if min == nil || max == nil {
		return 0
	}
	return *max - *min
}

func prepareChartData(data []models.WeatherData, field string) map[string]interface{} {
	labels := make([]string, 0, len(data))
	temps := make([]float64, 0, len(data))
	feelsLike := make([]float64, 0, len(data))
	dewPoint := make([]float64, 0, len(data))

	for _, item := range data {
		// Format time label
		labels = append(labels, item.Time.Format("15:04"))

		// Temperature
		if item.TempOutdoor != nil {
			temps = append(temps, float64(*item.TempOutdoor))
		} else {
			temps = append(temps, 0)
		}

		// Feels like
		if item.TempFeelsLike != nil {
			feelsLike = append(feelsLike, float64(*item.TempFeelsLike))
		} else {
			feelsLike = append(feelsLike, 0)
		}

		// Dew point
		if item.DewPoint != nil {
			dewPoint = append(dewPoint, float64(*item.DewPoint))
		} else {
			dewPoint = append(dewPoint, 0)
		}
	}

	return map[string]interface{}{
		"labels":    labels,
		"temps":     temps,
		"feelsLike": feelsLike,
		"dewPoint":  dewPoint,
	}
}

func toJSON(data map[string]interface{}) template.JS {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal chart data", "error", err)
		return template.JS("{}")
	}
	return template.JS(jsonBytes)
}
