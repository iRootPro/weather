package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/service"
)

// GeomagneticCardData — данные для карточки на дашборде, готовые к рендеру
// (без условий внутри шаблона).
type GeomagneticCardData struct {
	HasData        bool
	Kp             float32
	StatusLabel    string
	StatusHeading  string // "Спокойно" / "Возмущение" / "Магнитная буря G3" — крупно для пользователя
	StatusGradient string
	StatusText     string
	KpSubLine      string // мелкая подпись «Магнитные бури · Kp 2.3»
	PeakLine       string // готовая строка «Прогноз: …» или «Макс сегодня: …», либо пустая
}

// stormGLevel — уровень шкалы NOAA G по значению Kp (G1..G5).
// Возвращает 0 для Kp < 5.
func stormGLevel(kp float32) int {
	if kp < 5 {
		return 0
	}
	return min(5, max(1, int(kp)-4))
}

// statusHeading возвращает крупную подпись для пользователя:
// «Спокойно» / «Возмущение» / «Магнитная буря G1»…
func statusHeading(status models.KpStatus, kp float32) string {
	switch status {
	case models.KpStorm:
		return fmt.Sprintf("Магнитная буря G%d", stormGLevel(kp))
	case models.KpUnsettled:
		return "Возмущение"
	default:
		return "Спокойно"
	}
}

// buildGeomagneticCard собирает данные карточки. Никогда не возвращает ошибку —
// при сбое сервиса или отсутствии данных просто отдаёт HasData=false.
func (h *Handler) buildGeomagneticCard(ctx context.Context) GeomagneticCardData {
	if h.geomagneticService == nil {
		return GeomagneticCardData{}
	}
	snap, err := h.geomagneticService.GetDashboardSnapshot(ctx, time.Now())
	if err != nil {
		slog.Warn("failed to get geomagnetic snapshot", "error", err)
		return GeomagneticCardData{}
	}
	if !snap.HasData || snap.Current == nil {
		return GeomagneticCardData{}
	}

	card := GeomagneticCardData{
		HasData:        true,
		Kp:             snap.Current.Kp,
		StatusLabel:    snap.Status.Label(),
		StatusHeading:  statusHeading(snap.Status, snap.Current.Kp),
		StatusGradient: snap.Status.TailwindGradient(),
		StatusText:     snap.Status.TextColor(),
		KpSubLine:      fmt.Sprintf("Магнитные бури · Kp %.1f", snap.Current.Kp),
	}

	switch {
	case snap.NextStorm != nil:
		card.PeakLine = fmt.Sprintf(
			"Прогноз: буря %s, Kp %.1f",
			snap.NextStorm.SlotTime.In(time.Local).Format("02.01 15:04"),
			snap.NextStorm.Kp,
		)
	case snap.TodayMaxKp != nil:
		card.PeakLine = fmt.Sprintf(
			"Макс сегодня: Kp %.1f в %s",
			snap.TodayMaxKp.Kp,
			snap.TodayMaxKp.SlotTime.In(time.Local).Format("15:04"),
		)
	}

	return card
}

// DetailGeomagnetic рендерит детальную страницу по геомагнитной активности.
func (h *Handler) DetailGeomagnetic(w http.ResponseWriter, r *http.Request) {
	if h.geomagneticService == nil {
		http.Error(w, "Geomagnetic service not configured", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	now := time.Now()
	from := now.Add(-48 * time.Hour)
	to := now.Add(72 * time.Hour)

	detail, err := h.geomagneticService.GetDetail(ctx, from, to)
	if err != nil {
		slog.Error("failed to get geomagnetic detail", "error", err)
		http.Error(w, "Failed to load data", http.StatusInternalServerError)
		return
	}

	// Точки для Chart.js: метка ISO 8601 в UTC + значение Kp + цвет столбца
	type chartPoint struct {
		Time  string  `json:"time"`
		Kp    float32 `json:"kp"`
		Color string  `json:"color"`
	}
	points := make([]chartPoint, 0, len(detail.Kp))
	for _, slot := range detail.Kp {
		points = append(points, chartPoint{
			Time:  slot.SlotTime.UTC().Format(time.RFC3339),
			Kp:    slot.Kp,
			Color: models.ClassifyKp(slot.Kp).HexColor(),
		})
	}
	chartJSON, err := json.Marshal(points)
	if err != nil {
		slog.Error("failed to marshal chart points", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Текущий слот для блока «Сейчас»
	snap, err := h.geomagneticService.GetDashboardSnapshot(ctx, now)
	if err != nil {
		slog.Warn("failed to get geomagnetic snapshot", "error", err)
		snap = &service.DashboardSnapshot{}
	}

	tmpl, err := h.parseTemplate("detail/geomagnetic.html")
	if err != nil {
		slog.Error("failed to parse geomagnetic detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	type dailyRow struct {
		Date  string
		F10   string
		Sn    string
		Ap    string
		MaxKp string
	}
	rows := make([]dailyRow, 0, len(detail.Daily))
	for _, d := range detail.Daily {
		rows = append(rows, dailyRow{
			Date:  d.Date.Format("02.01.2006"),
			F10:   formatNullableFloat(d.F10, "%.1f"),
			Sn:    formatNullableFloat(d.Sn, "%.0f"),
			Ap:    formatNullableFloat(d.Ap, "%.0f"),
			MaxKp: formatNullableFloat(d.MaxKp, "%.0f"),
		})
	}

	card := h.buildGeomagneticCard(ctx)

	data := PageData{
		ActivePage: "dashboard",
		Data: map[string]any{
			"Card":      card,
			"Snapshot":  snap,
			"ChartJSON": string(chartJSON),
			"NowISO":    now.UTC().Format(time.RFC3339),
			"DailyRows": rows,
		},
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("failed to render geomagnetic detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func formatNullableFloat(p *float32, format string) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprintf(format, *p)
}
