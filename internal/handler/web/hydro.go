package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

type WaterLevelCardData struct {
	HasData        bool
	StationName    string
	ObjectName     string
	ObservedAt     string
	LevelM         float32
	ChangeText     string
	ChangeClass    string
	DayChangeText  string
	LeadText       string
	StatusLabel    string
	StatusGradient string
	StatusText     string
	ToPrevention   string
	ToDanger       string
}

func (h *Handler) buildWaterLevelCard(r *http.Request) WaterLevelCardData {
	if h.hydroService == nil {
		return WaterLevelCardData{}
	}
	snap, err := h.hydroService.GetSnapshot(r.Context(), time.Now())
	if err != nil {
		slog.Warn("failed to get hydro snapshot", "error", err)
		return WaterLevelCardData{}
	}
	if snap == nil || !snap.HasData || snap.Current == nil {
		return WaterLevelCardData{}
	}
	card := WaterLevelCardData{
		HasData:        true,
		ObservedAt:     snap.Current.ObservedAt.In(time.Local).Format("02.01 15:04"),
		LevelM:         snap.Current.LevelBSM,
		StatusLabel:    snap.Status.Label(),
		StatusGradient: snap.Status.TailwindGradient(),
		StatusText:     snap.Status.TextColor(),
	}
	if snap.Gauge != nil {
		card.StationName = snap.Gauge.HolderName
		if card.StationName == "" {
			card.StationName = snap.Gauge.Name
		}
		card.ObjectName = snap.Gauge.MonitoringObject
	}
	if snap.Current.ChangeCmPerHour != nil {
		card.ChangeText = formatSignedFloat(*snap.Current.ChangeCmPerHour, "%.0f см/ч")
		card.ChangeClass = changeClass(*snap.Current.ChangeCmPerHour)
	} else if snap.ChangeM != nil {
		cm := *snap.ChangeM * 100
		card.ChangeText = formatSignedFloat(cm, "%.0f см")
		card.ChangeClass = changeClass(cm)
	}
	if snap.Change24hM != nil {
		card.DayChangeText = formatSignedFloat(*snap.Change24hM*100, "%.0f см за сутки")
	}
	if snap.Current.LeadText != nil {
		card.LeadText = *snap.Current.LeadText
	}
	if snap.ToPreventionM != nil {
		card.ToPrevention = formatDistanceToThreshold(*snap.ToPreventionM)
	}
	if snap.ToDangerM != nil {
		card.ToDanger = formatDistanceToThreshold(*snap.ToDangerM)
	}
	return card
}

func (h *Handler) WaterLevelWidget(w http.ResponseWriter, r *http.Request) {
	card := h.buildWaterLevelCard(r)
	if !card.HasData {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	tmpl, err := h.parsePartial("water_level.html")
	if err != nil {
		slog.Error("failed to parse water level template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, card); err != nil {
		slog.Error("failed to render water level widget", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handler) DetailWaterLevel(w http.ResponseWriter, r *http.Request) {
	if h.hydroService == nil {
		http.Error(w, "Hydro service not configured", http.StatusServiceUnavailable)
		return
	}
	now := time.Now()
	from := now.AddDate(0, 0, -7)
	readings, err := h.hydroService.GetRange(r.Context(), from, now)
	if err != nil {
		slog.Error("failed to get hydro range", "error", err)
		http.Error(w, "Failed to load data", http.StatusInternalServerError)
		return
	}
	gauge, err := h.hydroService.GetGauge(r.Context())
	if err != nil {
		slog.Error("failed to get hydro gauge", "error", err)
		http.Error(w, "Failed to load data", http.StatusInternalServerError)
		return
	}

	type point struct {
		Time  string  `json:"time"`
		Level float32 `json:"level"`
	}
	points := make([]point, 0, len(readings))
	for _, row := range readings {
		points = append(points, point{Time: row.ObservedAt.UTC().Format(time.RFC3339), Level: row.LevelBSM})
	}
	chartJSON, _ := json.Marshal(points)

	card := h.buildWaterLevelCard(r)
	data := PageData{
		ActivePage: "dashboard",
		Data: map[string]any{
			"Card":      card,
			"Gauge":     gauge,
			"ChartJSON": string(chartJSON),
			"Rows":      buildWaterLevelRows(readings),
		},
	}
	tmpl, err := h.parseTemplate("detail/water_level.html")
	if err != nil {
		slog.Error("failed to parse water level detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("failed to render water level detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

type waterLevelRow struct {
	Time  string
	Level string
}

func buildWaterLevelRows(readings []models.HydroLevelReading) []waterLevelRow {
	limit := 24
	if len(readings) < limit {
		limit = len(readings)
	}
	out := make([]waterLevelRow, 0, limit)
	for i := len(readings) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, waterLevelRow{
			Time:  readings[i].ObservedAt.In(time.Local).Format("02.01 15:04"),
			Level: fmt.Sprintf("%.3f", readings[i].LevelBSM),
		})
	}
	return out
}

func formatSignedFloat(v float32, format string) string {
	prefix := ""
	if v > 0 {
		prefix = "+"
	}
	return prefix + fmt.Sprintf(format, v)
}

func changeClass(v float32) string {
	switch {
	case v > 0:
		return "text-red-600 dark:text-red-300"
	case v < 0:
		return "text-green-600 dark:text-green-300"
	default:
		return "text-gray-500 dark:text-gray-400"
	}
}

func formatDistanceToThreshold(v float32) string {
	if v < 0 {
		return fmt.Sprintf("превышен на %.0f см", -v*100)
	}
	return fmt.Sprintf("осталось %.0f см", v*100)
}
