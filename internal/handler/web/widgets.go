package web

import (
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
)

func (h *Handler) parsePartial(name string) (*template.Template, error) {
	partialPath := filepath.Join(h.templatesDir, "partials", name)
	return template.ParseFiles(partialPath)
}

// CurrentWeatherWidget renders the current weather widget
func (h *Handler) CurrentWeatherWidget(w http.ResponseWriter, r *http.Request) {
	data, err := h.weatherService.GetCurrent(r.Context())
	if err != nil {
		slog.Error("failed to get current weather", "error", err)
		http.Error(w, "Failed to load weather data", http.StatusInternalServerError)
		return
	}

	// Convert pointer values for template
	templateData := struct {
		Time             string
		TempOutdoor      float32
		TempIndoor       float32
		HumidityOutdoor  int16
		HumidityIndoor   int16
		PressureRelative float32
		WindSpeed        float32
		WindGust         float32
		RainRate         float32
		RainDaily        float32
		RainMonthly      float32
		UVIndex          float32
		SolarRadiation   float32
		Illuminance      float32 // lux = solar radiation * 120
	}{
		Time: data.Time.Format("15:04"),
	}

	if data.TempOutdoor != nil {
		templateData.TempOutdoor = *data.TempOutdoor
	}
	if data.TempIndoor != nil {
		templateData.TempIndoor = *data.TempIndoor
	}
	if data.HumidityOutdoor != nil {
		templateData.HumidityOutdoor = *data.HumidityOutdoor
	}
	if data.HumidityIndoor != nil {
		templateData.HumidityIndoor = *data.HumidityIndoor
	}
	if data.PressureRelative != nil {
		templateData.PressureRelative = *data.PressureRelative
	}
	if data.WindSpeed != nil {
		templateData.WindSpeed = *data.WindSpeed
	}
	if data.WindGust != nil {
		templateData.WindGust = *data.WindGust
	}
	if data.RainRate != nil {
		templateData.RainRate = *data.RainRate
	}
	if data.RainDaily != nil {
		templateData.RainDaily = *data.RainDaily
	}
	if data.RainMonthly != nil {
		templateData.RainMonthly = *data.RainMonthly
	}
	if data.UVIndex != nil {
		templateData.UVIndex = *data.UVIndex
	}
	if data.SolarRadiation != nil {
		templateData.SolarRadiation = *data.SolarRadiation
		templateData.Illuminance = *data.SolarRadiation * 120 // approximate conversion to lux
	}

	tmpl, err := h.parsePartial("current_weather.html")
	if err != nil {
		slog.Error("failed to parse current weather template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render current weather widget", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// StatsWidget renders the daily stats widget
func (h *Handler) StatsWidget(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "day"
	}

	stats, err := h.weatherService.GetStats(r.Context(), period)
	if err != nil {
		slog.Error("failed to get stats", "error", err)
		http.Error(w, "Failed to load stats", http.StatusInternalServerError)
		return
	}

	// Convert pointer values for template
	templateData := struct {
		TempOutdoorMin      float32
		TempOutdoorMax      float32
		TempOutdoorAvg      float32
		PressureRelativeMin float32
		PressureRelativeMax float32
		PressureRelativeAvg float32
		WindSpeedMax        float32
		RainTotal           float32
	}{}

	if stats.TempOutdoorMin != nil {
		templateData.TempOutdoorMin = *stats.TempOutdoorMin
	}
	if stats.TempOutdoorMax != nil {
		templateData.TempOutdoorMax = *stats.TempOutdoorMax
	}
	if stats.TempOutdoorAvg != nil {
		templateData.TempOutdoorAvg = *stats.TempOutdoorAvg
	}
	if stats.PressureRelativeMin != nil {
		templateData.PressureRelativeMin = *stats.PressureRelativeMin
	}
	if stats.PressureRelativeMax != nil {
		templateData.PressureRelativeMax = *stats.PressureRelativeMax
	}
	if stats.PressureRelativeAvg != nil {
		templateData.PressureRelativeAvg = *stats.PressureRelativeAvg
	}
	if stats.WindSpeedMax != nil {
		templateData.WindSpeedMax = *stats.WindSpeedMax
	}
	if stats.RainTotal != nil {
		templateData.RainTotal = *stats.RainTotal
	}

	tmpl, err := h.parsePartial("daily_stats.html")
	if err != nil {
		slog.Error("failed to parse stats template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render stats widget", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
