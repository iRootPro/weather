package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

type WaterLevelCardData struct {
	HasData           bool
	StationName       string
	ObjectName        string
	ObservedAt        string
	LevelM            float32
	RelativeLevelCm   string
	ChangeText        string
	ChangeClass       string
	DayChangeText     string
	LeadText          string
	StatusLabel       string
	StatusGradient    string
	StatusText        string
	ToPrevention      string
	ToDanger          string
	RiskPct           int
	RiskBarClass      string
	RiskHeadline      string
	RiskCaption       string
	ThresholdReached  bool
	ThresholdTitle    string
	ThresholdValue    string
	ThresholdDetail   string
	AbsoluteLevelText string
	StatusNote        string
	TrendText         string
	TrendClass        string
	Sparkline         WaterSparklineData
	Upstream          []WaterLevelMiniData
}

type WaterLevelMiniData struct {
	StationName       string
	ObjectName        string
	Role              string
	ObservedAt        string
	LevelM            float32
	ChangeText        string
	ChangeClass       string
	DayChangeText     string
	StatusLabel       string
	StatusText        string
	ToPrevention      string
	ToDanger          string
	RiskPct           int
	RiskBarClass      string
	RiskHeadline      string
	RiskCaption       string
	AbsoluteLevelText string
	TrendText         string
	TrendClass        string
	Sparkline         WaterSparklineData
}

type WaterSparklineData struct {
	HasData          bool
	Points           string
	FillPoints       string
	LastX            int
	LastY            int
	MinText          string
	MaxText          string
	ThresholdY       int
	HasThresholdLine bool
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
		HasData:           true,
		ObservedAt:        snap.Current.ObservedAt.In(time.Local).Format("02.01 15:04"),
		LevelM:            snap.Current.LevelBSM,
		AbsoluteLevelText: fmt.Sprintf("%.3f м, Балтийская система высот", snap.Current.LevelBSM),
		StatusLabel:       snap.Status.Label(),
		StatusGradient:    snap.Status.TailwindGradient(),
		StatusText:        snap.Status.TextColor(),
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
		card.DayChangeText = formatSignedFloat(*snap.Change24hM*100, "%.0f см")
	}
	if snap.RelativeLevelCm != nil {
		card.RelativeLevelCm = fmt.Sprintf("%.0f см над нулём поста", *snap.RelativeLevelCm)
	}
	if snap.Current.LeadText != nil && *snap.Current.LeadText != "?" && *snap.Current.LeadText != "---" {
		card.LeadText = *snap.Current.LeadText
	}
	if snap.ToPreventionM != nil {
		card.ToPrevention = formatDistanceToThreshold(*snap.ToPreventionM)
	}
	if snap.ToDangerM != nil {
		card.ToDanger = formatDistanceToThreshold(*snap.ToDangerM)
	}
	if snap.ToPreventionM != nil {
		card.RiskPct = riskPercent(*snap.ToPreventionM, snap.Status)
		card.RiskBarClass = riskBarClass(snap.Status, card.RiskPct)
		card.RiskHeadline, card.RiskCaption = riskHeadline(*snap.ToPreventionM)
		card.ThresholdReached, card.ThresholdTitle, card.ThresholdValue, card.ThresholdDetail = thresholdBlock(*snap.ToPreventionM, card.RiskPct)
	} else {
		card.RiskHeadline = fmt.Sprintf("%.3f м", snap.Current.LevelBSM)
		card.RiskCaption = "текущая отметка уровня"
	}
	card.TrendText, card.TrendClass = trendLabel(snap.Current.ChangeCmPerHour, snap.Change24hM)
	card.StatusNote = hydroStatusNote(snap.Status, snap.Current.ChangeCmPerHour, snap.Change24hM)
	upstream, err := h.hydroService.GetUpstreamSnapshots(r.Context(), time.Now())
	if err != nil {
		slog.Warn("failed to get upstream hydro snapshots", "error", err)
	} else {
		to := time.Now()
		from := to.Add(-24 * time.Hour)
		for _, item := range upstream {
			if mini := buildWaterLevelMini(item); mini != nil {
				if item.Current != nil {
					readings, err := h.hydroService.GetRangeForStation(r.Context(), item.Current.StationUUID, from, to)
					if err != nil {
						slog.Warn("failed to get upstream hydro sparkline", "station_uuid", item.Current.StationUUID, "error", err)
					} else {
						mini.Sparkline = buildWaterSparkline(readings, item.Gauge)
					}
				}
				card.Upstream = append(card.Upstream, *mini)
			}
		}
	}
	return card
}

func buildWaterLevelMini(snap *models.HydroSnapshot) *WaterLevelMiniData {
	if snap == nil || !snap.HasData || snap.Current == nil {
		return nil
	}
	mini := &WaterLevelMiniData{
		ObservedAt:        snap.Current.ObservedAt.In(time.Local).Format("02.01 15:04"),
		LevelM:            snap.Current.LevelBSM,
		AbsoluteLevelText: fmt.Sprintf("%.3f м, Балтийская система высот", snap.Current.LevelBSM),
		StatusLabel:       snap.Status.Label(),
		StatusText:        snap.Status.TextColor(),
		RiskBarClass:      riskBarClass(snap.Status, 0),
	}
	if snap.Gauge != nil {
		mini.StationName = snap.Gauge.HolderName
		if snap.Gauge.Locality != nil && *snap.Gauge.Locality != "" {
			if mini.StationName != "" {
				mini.StationName += " · " + *snap.Gauge.Locality
			} else {
				mini.StationName = *snap.Gauge.Locality
			}
		}
		if mini.StationName == "" {
			mini.StationName = snap.Gauge.Name
		}
		mini.ObjectName = snap.Gauge.MonitoringObject
		mini.Role = hydroStationRole(mini.ObjectName)
	}
	if snap.Current.ChangeCmPerHour != nil {
		mini.ChangeText = formatSignedFloat(*snap.Current.ChangeCmPerHour, "%.0f см/ч")
		mini.ChangeClass = changeClass(*snap.Current.ChangeCmPerHour)
	} else if snap.ChangeM != nil {
		cm := *snap.ChangeM * 100
		mini.ChangeText = formatSignedFloat(cm, "%.0f см")
		mini.ChangeClass = changeClass(cm)
	}
	if snap.Change24hM != nil {
		mini.DayChangeText = formatSignedFloat(*snap.Change24hM*100, "%.0f см")
	}
	if snap.ToPreventionM != nil {
		mini.ToPrevention = formatDistanceToThreshold(*snap.ToPreventionM)
		mini.RiskPct = riskPercent(*snap.ToPreventionM, snap.Status)
		mini.RiskBarClass = riskBarClass(snap.Status, mini.RiskPct)
		mini.RiskHeadline, mini.RiskCaption = riskHeadline(*snap.ToPreventionM)
	} else {
		mini.RiskHeadline = fmt.Sprintf("%.3f м", snap.Current.LevelBSM)
		mini.RiskCaption = "текущая отметка уровня"
	}
	if snap.ToDangerM != nil {
		mini.ToDanger = formatDistanceToThreshold(*snap.ToDangerM)
	}
	mini.TrendText, mini.TrendClass = trendLabel(snap.Current.ChangeCmPerHour, snap.Change24hM)
	return mini
}

func hydroStationRole(objectName string) string {
	if strings.Contains(strings.ToLower(objectName), "кубан") {
		return "выше по руслу"
	}
	return "приток бассейна"
}

func trendLabel(changePerHour *float32, change24hM *float32) (string, string) {
	if changePerHour != nil {
		v := *changePerHour
		switch {
		case v >= 3:
			return "быстрый рост ↑", "text-red-700 dark:text-red-300"
		case v >= 1:
			return "растёт ↑", "text-red-600 dark:text-red-300"
		case v <= -3:
			return "быстро снижается ↓", "text-green-700 dark:text-green-300"
		case v <= -1:
			return "снижается ↓", "text-green-600 dark:text-green-300"
		default:
			return "стабильно →", "text-gray-600 dark:text-gray-300"
		}
	}
	if change24hM != nil {
		cm := *change24hM * 100
		switch {
		case cm >= 20:
			return "быстрый рост ↑", "text-red-700 dark:text-red-300"
		case cm >= 5:
			return "растёт ↑", "text-red-600 dark:text-red-300"
		case cm <= -20:
			return "быстро снижается ↓", "text-green-700 dark:text-green-300"
		case cm <= -5:
			return "снижается ↓", "text-green-600 dark:text-green-300"
		default:
			return "стабильно →", "text-gray-600 dark:text-gray-300"
		}
	}
	return "наблюдение", "text-gray-600 dark:text-gray-300"
}

func hydroStatusNote(status models.HydroStatus, changePerHour *float32, change24hM *float32) string {
	trend := summaryTrendPhrase(changePerHour, change24hM)
	switch status {
	case models.HydroStatusDanger:
		return "Опасный уровень уже достигнут. Смотрите динамику и предупреждения служб."
	case models.HydroStatusPrevention:
		return "Неблагоприятный уровень уже достигнут. Следующий ориентир — опасный уровень."
	case models.HydroStatusNear:
		return "Порог почти рядом. Важнее всего сейчас — скорость роста и посты выше по течению."
	default:
		if strings.Contains(trend, "растёт") {
			return "Пока ниже порога, но уровень растёт — следим за динамикой."
		}
		if strings.Contains(trend, "снижается") {
			return "Уровень ниже порога и снижается."
		}
		return "Уровень ниже порога, резких изменений нет."
	}
}

func summaryTrendPhrase(changePerHour *float32, change24hM *float32) string {
	if changePerHour != nil {
		v := *changePerHour
		switch {
		case v >= 3:
			return "быстро растёт"
		case v >= 1:
			return "растёт"
		case v <= -3:
			return "быстро снижается"
		case v <= -1:
			return "снижается"
		default:
			return "почти без изменений"
		}
	}
	if change24hM != nil {
		cm := *change24hM * 100
		switch {
		case cm >= 20:
			return "быстро растёт"
		case cm >= 5:
			return "растёт"
		case cm <= -20:
			return "быстро снижается"
		case cm <= -5:
			return "снижается"
		default:
			return "почти без изменений"
		}
	}
	return "под наблюдением"
}

func (h *Handler) WaterLevelWidget(w http.ResponseWriter, r *http.Request) {
	card := h.buildWaterLevelCard(r)
	if !card.HasData {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if h.hydroService != nil {
		to := time.Now()
		from := to.Add(-24 * time.Hour)
		readings, err := h.hydroService.GetRange(r.Context(), from, to)
		if err != nil {
			slog.Warn("failed to get hydro sparkline", "error", err)
		} else {
			gauge, _ := h.hydroService.GetGauge(r.Context())
			card.Sparkline = buildWaterSparkline(readings, gauge)
		}
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
	readings = filterHydroOutliers(readings)
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

func filterHydroOutliers(readings []models.HydroLevelReading) []models.HydroLevelReading {
	if len(readings) < 4 {
		return readings
	}
	levels := make([]float32, 0, len(readings))
	for _, r := range readings {
		levels = append(levels, r.LevelBSM)
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i] < levels[j] })
	median := levels[len(levels)/2]
	q1 := levels[len(levels)/4]
	q3 := levels[(len(levels)*3)/4]
	iqr := q3 - q1
	limit := iqr * 6
	if limit < 0.75 {
		limit = 0.75
	}
	if limit > 3 {
		limit = 3
	}
	out := make([]models.HydroLevelReading, 0, len(readings))
	for _, r := range readings {
		if absFloat32(r.LevelBSM-median) <= limit {
			out = append(out, r)
		}
	}
	if len(out) < 2 {
		return readings
	}
	return out
}

func buildWaterSparkline(readings []models.HydroLevelReading, gauge *models.HydroGauge) WaterSparklineData {
	readings = filterHydroOutliers(readings)
	if len(readings) < 2 {
		return WaterSparklineData{}
	}

	// Для компактного SVG оставляем не больше 72 точек: форма сохраняется, HTML не раздувается.
	step := 1
	if len(readings) > 72 {
		step = (len(readings) + 71) / 72
	}
	sampled := make([]models.HydroLevelReading, 0, len(readings)/step+1)
	for i := 0; i < len(readings); i += step {
		sampled = append(sampled, readings[i])
	}
	if last := readings[len(readings)-1]; sampled[len(sampled)-1].ObservedAt != last.ObservedAt {
		sampled = append(sampled, last)
	}

	minLevel, maxLevel := sampled[0].LevelBSM, sampled[0].LevelBSM
	for _, r := range sampled {
		if r.LevelBSM < minLevel {
			minLevel = r.LevelBSM
		}
		if r.LevelBSM > maxLevel {
			maxLevel = r.LevelBSM
		}
	}
	var prevention *float32
	if gauge != nil && gauge.FloodingPreventionBM != nil {
		prevention = gauge.FloodingPreventionBM
		if *prevention < minLevel {
			minLevel = *prevention
		}
		if *prevention > maxLevel {
			maxLevel = *prevention
		}
	}
	if maxLevel-minLevel < 0.05 {
		mid := (maxLevel + minLevel) / 2
		minLevel = mid - 0.025
		maxLevel = mid + 0.025
	}
	padding := (maxLevel - minLevel) * 0.12
	minLevel -= padding
	maxLevel += padding

	toXY := func(i int, level float32) (int, int) {
		x := 0
		if len(sampled) > 1 {
			x = int(float32(i) / float32(len(sampled)-1) * 100)
		}
		y := int((1 - (level-minLevel)/(maxLevel-minLevel)) * 44)
		if y < 2 {
			y = 2
		}
		if y > 42 {
			y = 42
		}
		return x, y
	}

	points := ""
	for i, r := range sampled {
		x, y := toXY(i, r.LevelBSM)
		if i > 0 {
			points += " "
		}
		points += fmt.Sprintf("%d,%d", x, y)
	}
	lastX, lastY := toXY(len(sampled)-1, sampled[len(sampled)-1].LevelBSM)
	out := WaterSparklineData{
		HasData:    true,
		Points:     points,
		FillPoints: "0,44 " + points + " 100,44",
		LastX:      lastX,
		LastY:      lastY,
		MinText:    fmt.Sprintf("%.3f", minLevel+padding),
		MaxText:    fmt.Sprintf("%.3f", maxLevel-padding),
	}
	if prevention != nil {
		_, y := toXY(0, *prevention)
		out.ThresholdY = y
		out.HasThresholdLine = true
	}
	return out
}

func thresholdBlock(toPreventionM float32, riskPct int) (bool, string, string, string) {
	cm := toPreventionM * 100
	if cm <= 0 {
		return true, "Неблагоприятный порог", "достигнут", "уровень уже выше порога"
	}
	return false, "Запас до неблагоприятного", formatDistanceToThreshold(toPreventionM), fmt.Sprintf("Шкала заполнена на %d%%", riskPct)
}

func riskHeadline(toPreventionM float32) (string, string) {
	cm := toPreventionM * 100
	if cm < 0 {
		return formatUnsignedCm(-cm), "выше неблагоприятного уровня"
	}
	if cm == 0 {
		return "на пороге", "неблагоприятного уровня"
	}
	return formatUnsignedCm(cm), "до неблагоприятного уровня"
}

func riskPercent(toPreventionM float32, status models.HydroStatus) int {
	if status == models.HydroStatusDanger || status == models.HydroStatusPrevention || toPreventionM <= 0 {
		return 100
	}
	cm := toPreventionM * 100
	if cm >= 100 {
		return 0
	}
	pct := int(100 - cm)
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

func riskBarClass(status models.HydroStatus, pct int) string {
	if status == models.HydroStatusDanger {
		return "bg-red-500"
	}
	if status == models.HydroStatusPrevention || status == models.HydroStatusNear || pct >= 80 {
		return "bg-orange-500"
	}
	return "bg-sky-500"
}

func absFloat32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
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
		return "превышен на " + formatUnsignedCm(-v*100)
	}
	return "осталось " + formatUnsignedCm(v*100)
}

func formatUnsignedCm(cm float32) string {
	if cm > 0 && cm < 1 {
		return "менее 1 см"
	}
	return fmt.Sprintf("%.0f см", cm)
}
