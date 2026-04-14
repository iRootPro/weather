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

// SparkBar — один бар компактной столбчатой диаграммы для дашборда.
type SparkBar struct {
	HeightPct int    // высота в процентах (1..100)
	Color     string // CSS-цвет
	Title     string // подсказка по hover (например, "13.04 12:00 · Kp 0.7")
	Empty     bool   // true → плейсхолдер для отсутствующего слота
}

// GeomagneticCardData — данные для карточки на дашборде, готовые к рендеру
// (без условий внутри шаблона).
type GeomagneticCardData struct {
	HasData        bool
	Kp             float32
	StatusLabel    string
	StatusHeading  string // "Спокойно" / "Возмущение" / "Магнитная буря G3" — крупно для пользователя
	StatusGradient string
	StatusText     string
	SubLine        string // поясняющая подпись под крупным статусом — объясняет, о чём блок
	PeakLine       string // готовая строка «Прогноз: …» или «Макс сегодня: …», либо пустая
	Sparkline      []SparkBar
}

// statusHeading возвращает крупную подпись для пользователя:
// «Спокойно» / «Возмущение» / «Магнитная буря G1»…
func statusHeading(status models.KpStatus, kp float32) string {
	switch status {
	case models.KpStorm:
		if gLevel, _, ok := models.StormLevel(kp); ok {
			return "Магнитная буря " + gLevel
		}
		return "Магнитная буря"
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
	now := time.Now()
	snap, err := h.geomagneticService.GetDashboardSnapshot(ctx, now)
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
		SubLine:        "Магнитное поле Земли",
		Sparkline:      h.buildSparkline(ctx, now),
	}

	switch {
	case snap.NextStorm != nil:
		when := snap.NextStorm.SlotTime.In(time.Local).Format("02.01 в 15:04")
		if gLevel, desc, ok := models.StormLevel(snap.NextStorm.Kp); ok {
			card.PeakLine = fmt.Sprintf("Прогноз: буря %s «%s» %s", gLevel, desc, when)
		}
	case snap.TodayMaxKp != nil:
		when := snap.TodayMaxKp.SlotTime.In(time.Local).Format("15:04")
		if gLevel, _, ok := models.StormLevel(snap.TodayMaxKp.Kp); ok {
			card.PeakLine = fmt.Sprintf("Макс сегодня: буря %s в %s", gLevel, when)
		} else if models.ClassifyKp(snap.TodayMaxKp.Kp) == models.KpUnsettled {
			card.PeakLine = fmt.Sprintf("Макс сегодня: возмущение в %s", when)
		}
	}

	return card
}

// buildSparkline собирает 16 фиксированных 3-часовых слотов (последние 48 часов).
// Если для какого-то слота данных нет — отдаёт Empty=true.
func (h *Handler) buildSparkline(ctx context.Context, now time.Time) []SparkBar {
	const slotCount = 16
	const slotDur = 3 * time.Hour

	// Якорь — ближайший прошедший 3-часовой слот по локальному времени.
	local := now.In(time.Local)
	hour := (local.Hour() / 3) * 3
	anchor := time.Date(local.Year(), local.Month(), local.Day(), hour, 0, 0, 0, local.Location())
	from := anchor.Add(-time.Duration(slotCount-1) * slotDur)
	to := anchor.Add(slotDur - time.Second)

	rows, err := h.geomagneticService.GetDetail(ctx, from, to)
	if err != nil || rows == nil {
		return nil
	}

	// Индексируем по началу слота (UTC) для быстрого поиска.
	byTime := make(map[int64]float32, len(rows.Kp))
	for _, k := range rows.Kp {
		byTime[k.SlotTime.UTC().Truncate(slotDur).Unix()] = k.Kp
	}

	bars := make([]SparkBar, 0, slotCount)
	for i := range slotCount {
		slotStart := from.Add(time.Duration(i) * slotDur)
		key := slotStart.UTC().Truncate(slotDur).Unix()
		kp, ok := byTime[key]
		if !ok {
			bars = append(bars, SparkBar{Empty: true})
			continue
		}
		// Высота в процентах от Kp = 9. Минимум 4%, чтобы было видно нулевые.
		clamped := kp
		if clamped < 0 {
			clamped = 0
		}
		if clamped > 9 {
			clamped = 9
		}
		pct := int(float32(clamped)/9*96) + 4
		bars = append(bars, SparkBar{
			HeightPct: pct,
			Color:     models.ClassifyKp(kp).HexColor(),
			Title:     fmt.Sprintf("%s · Kp %.1f", slotStart.Format("02.01 15:04"), kp),
		})
	}
	return bars
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
