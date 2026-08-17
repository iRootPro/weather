package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

var (
	ErrInvalidArchivePeriod = errors.New("invalid archive period")
	ErrInvalidArchiveRange  = errors.New("invalid archive date range")
	ErrInvalidArchiveSearch = errors.New("invalid archive day search")
)

const archiveAvailabilityYears = 10

// GetArchive returns station observations for a calendar period or an inclusive custom range.
func (s *WeatherService) GetArchive(ctx context.Context, period, metric, monthParam, seasonParam, yearParam, fromParam, toParam, searchField, searchComparison, searchThreshold string) (*models.WeatherArchivePage, error) {
	loc := s.location
	if loc == nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	currentDayEnd := dayStart(now, loc).AddDate(0, 0, 1)
	period = strings.ToLower(strings.TrimSpace(period))
	if period == "" {
		period = "month"
	}
	metric = normalizeArchiveMetric(metric)
	search, err := resolveArchiveDaySearch(searchField, searchComparison, searchThreshold)
	if err != nil {
		return nil, err
	}

	start, end, label, err := resolveArchivePeriod(period, monthParam, seasonParam, yearParam, fromParam, toParam, now, loc)
	if err != nil {
		return nil, err
	}
	if !start.Before(currentDayEnd) {
		return nil, ErrInvalidArchiveRange
	}

	dataEnd := minTime(end, now)
	calendarEnd := minTime(end, currentDayEnd)
	if !dataEnd.After(start) {
		return nil, ErrInvalidArchiveRange
	}
	daysInPeriod := daysBetween(start, calendarEnd)
	if daysInPeriod < 1 {
		return nil, ErrInvalidArchiveRange
	}

	currentDays, err := s.repo.GetDailyInsights(ctx, start, dataEnd, s.timezone)
	if err != nil {
		return nil, err
	}
	availabilityStart := time.Date(now.Year()-archiveAvailabilityYears, time.January, 1, 0, 0, 0, 0, loc)
	availabilityDays, err := s.repo.GetDailyInsights(ctx, availabilityStart, now, s.timezone)
	if err != nil {
		return nil, err
	}
	firstDate, lastDate := archiveDateBounds(availabilityDays, now, loc)
	coverage := buildArchiveCoverage(start, calendarEnd, currentDays, availabilityDays, loc)
	events := filterArchiveEvents(buildArchiveEvents(currentDays), metric)
	displayDays := filterArchiveDays(currentDays, search)
	search.MatchedDays = len(displayDays)

	page := &models.WeatherArchivePage{
		GeneratedAt:    now,
		Period:         period,
		Metric:         metric,
		PeriodLabel:    label,
		FromParam:      start.Format("2006-01-02"),
		ToParam:        calendarEnd.AddDate(0, 0, -1).Format("2006-01-02"),
		FirstDateParam: firstDate.Format("2006-01-02"),
		LastDateParam:  lastDate.Format("2006-01-02"),
		MonthParam:     start.Format("2006-01"),
		SeasonParam:    seasonParamForDate(start, loc),
		YearParam:      start.Year(),
		SeasonOptions:  archiveSeasonOptions(availabilityDays, now, loc),
		YearOptions:    archiveYearOptions(availabilityDays, now),
		Summary:        buildArchiveSummary(currentDays, daysInPeriod),
		Coverage:       coverage,
		Events:         events,
		Search:         search,
		Daily:          displayDays,
	}
	return page, nil
}

func resolveArchivePeriod(period, monthParam, seasonParam, yearParam, fromParam, toParam string, now time.Time, loc *time.Location) (time.Time, time.Time, string, error) {
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	switch period {
	case "month":
		start := currentMonth
		if monthParam != "" {
			parsed, err := time.ParseInLocation("2006-01", monthParam, loc)
			if err != nil {
				return time.Time{}, time.Time{}, "", ErrInvalidArchivePeriod
			}
			start = parsed
		}
		if start.After(currentMonth) {
			return time.Time{}, time.Time{}, "", ErrInvalidArchiveRange
		}
		return start, start.AddDate(0, 1, 0), russianMonthYear(start), nil
	case "season":
		start, end := seasonBounds(now, loc)
		selectedYear, selectedCode := seasonIDFromStart(start)
		if seasonParam != "" {
			parsedYear, parsedCode, err := parseSeasonParam(seasonParam)
			if err != nil {
				return time.Time{}, time.Time{}, "", ErrInvalidArchivePeriod
			}
			candidateStart, candidateEnd := seasonBoundsByID(parsedYear, parsedCode, loc)
			if candidateStart.After(start) {
				return time.Time{}, time.Time{}, "", ErrInvalidArchiveRange
			}
			start, end, selectedYear, selectedCode = candidateStart, candidateEnd, parsedYear, parsedCode
		}
		return start, end, seasonLabel(selectedYear, selectedCode), nil
	case "year":
		year := now.Year()
		if yearParam != "" {
			if _, err := fmt.Sscanf(yearParam, "%d", &year); err != nil || year < 2000 || year > now.Year() {
				return time.Time{}, time.Time{}, "", ErrInvalidArchivePeriod
			}
		}
		start := time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(1, 0, 0), fmt.Sprintf("%d год", year), nil
	case "range":
		start, errStart := time.ParseInLocation("2006-01-02", fromParam, loc)
		endDate, errEnd := time.ParseInLocation("2006-01-02", toParam, loc)
		if errStart != nil || errEnd != nil || endDate.Before(start) || daysBetween(start, endDate.AddDate(0, 0, 1)) > 366 {
			return time.Time{}, time.Time{}, "", ErrInvalidArchiveRange
		}
		return start, endDate.AddDate(0, 0, 1), fmt.Sprintf("%s — %s", start.Format("02.01.2006"), endDate.Format("02.01.2006")), nil
	default:
		return time.Time{}, time.Time{}, "", ErrInvalidArchivePeriod
	}
}

func archiveDateBounds(days []models.DailyWeatherInsight, now time.Time, loc *time.Location) (time.Time, time.Time) {
	if len(days) == 0 {
		return dayStart(now, loc), dayStart(now, loc)
	}
	return days[0].Date.In(loc), days[len(days)-1].Date.In(loc)
}

func archiveSeasonOptions(days []models.DailyWeatherInsight, now time.Time, loc *time.Location) []models.WeatherInsightsPeriodOption {
	available := make(map[string]bool)
	for _, day := range days {
		available[seasonParamForDate(day.Date, loc)] = true
	}
	options := make([]models.WeatherInsightsPeriodOption, 0, len(available))
	start, _ := seasonBounds(now, loc)
	for len(options) < len(available) {
		year, code := seasonIDFromStart(start)
		param := formatSeasonParam(year, code)
		if available[param] {
			options = append(options, models.WeatherInsightsPeriodOption{Value: param, Label: seasonLabel(year, code)})
		}
		start = start.AddDate(0, -3, 0)
	}
	return options
}

func archiveYearOptions(days []models.DailyWeatherInsight, now time.Time) []int {
	available := make(map[int]bool)
	for _, day := range days {
		available[day.Date.Year()] = true
	}
	years := make([]int, 0, len(available))
	for year := now.Year(); year >= 2000; year-- {
		if available[year] {
			years = append(years, year)
		}
	}
	return years
}

func seasonParamForDate(date time.Time, loc *time.Location) string {
	start, _ := seasonBounds(date, loc)
	year, code := seasonIDFromStart(start)
	return formatSeasonParam(year, code)
}

func buildArchiveCoverage(start, end time.Time, periodDays, availabilityDays []models.DailyWeatherInsight, loc *time.Location) models.WeatherArchiveCoverage {
	coverage := models.WeatherArchiveCoverage{
		ExpectedDays: daysBetween(start, end),
		CoveredDays:  len(periodDays),
		Gaps:         make([]models.WeatherArchiveGap, 0),
	}
	if len(availabilityDays) > 0 {
		coverage.FirstObserved = availabilityDays[0].Date.In(loc)
		coverage.LastObserved = availabilityDays[len(availabilityDays)-1].Date.In(loc)
	}

	available := make(map[string]bool, len(periodDays))
	for _, day := range periodDays {
		available[day.Date.In(loc).Format("2006-01-02")] = true
	}
	var gapStart time.Time
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		if available[day.Format("2006-01-02")] {
			if !gapStart.IsZero() {
				gapDays := daysBetween(gapStart, day)
				coverage.Gaps = append(coverage.Gaps, models.WeatherArchiveGap{From: gapStart, To: day.AddDate(0, 0, -1), Days: gapDays})
				coverage.LongestGapDays = maxInt(coverage.LongestGapDays, gapDays)
				gapStart = time.Time{}
			}
			continue
		}
		coverage.MissingDays++
		if gapStart.IsZero() {
			gapStart = day
		}
	}
	if !gapStart.IsZero() {
		gapDays := daysBetween(gapStart, end)
		coverage.Gaps = append(coverage.Gaps, models.WeatherArchiveGap{From: gapStart, To: end.AddDate(0, 0, -1), Days: gapDays})
		coverage.LongestGapDays = maxInt(coverage.LongestGapDays, gapDays)
	}
	return coverage
}

func buildArchiveEvents(days []models.DailyWeatherInsight) []models.WeatherArchiveEvent {
	var hottest, coldest, wettest, windiest, sunniest *models.WeatherArchiveEvent
	for _, day := range days {
		if day.TempMax != nil && (hottest == nil || float64(*day.TempMax) > hottest.Value) {
			hottest = &models.WeatherArchiveEvent{Group: "temperature", Icon: "🌡️", Title: "Самый жаркий день", Date: day.Date, Value: float64(*day.TempMax), Unit: "°C"}
		}
		if day.TempMin != nil && (coldest == nil || float64(*day.TempMin) < coldest.Value) {
			coldest = &models.WeatherArchiveEvent{Group: "temperature", Icon: "❄️", Title: "Самая низкая температура", Date: day.Date, Value: float64(*day.TempMin), Unit: "°C"}
		}
		if day.RainTotal != nil && *day.RainTotal > 0 && (wettest == nil || float64(*day.RainTotal) > wettest.Value) {
			wettest = &models.WeatherArchiveEvent{Group: "precipitation", Icon: "🌧️", Title: "Самый дождливый день", Date: day.Date, Value: float64(*day.RainTotal), Unit: "мм"}
		}
		if day.WindGustMax != nil && (windiest == nil || float64(*day.WindGustMax) > windiest.Value) {
			windiest = &models.WeatherArchiveEvent{Group: "wind", Icon: "💨", Title: "Самый сильный порыв", Date: day.Date, Value: float64(*day.WindGustMax), Unit: "м/с"}
		}
		if day.SolarRadiationMax != nil && *day.SolarRadiationMax > 0 && (sunniest == nil || float64(*day.SolarRadiationMax) > sunniest.Value) {
			sunniest = &models.WeatherArchiveEvent{Group: "sun", Icon: "☀️", Title: "Самый солнечный день", Date: day.Date, Value: float64(*day.SolarRadiationMax), Unit: "Вт/м²"}
		}
	}
	events := make([]models.WeatherArchiveEvent, 0, 5)
	for _, event := range []*models.WeatherArchiveEvent{hottest, coldest, wettest, windiest, sunniest} {
		if event != nil {
			events = append(events, *event)
		}
	}
	return events
}

func filterArchiveEvents(events []models.WeatherArchiveEvent, metric string) []models.WeatherArchiveEvent {
	if metric == "all" {
		return events
	}
	allowed := map[string]bool{
		"temperature":   metric == "temperature",
		"precipitation": metric == "precipitation",
		"wind":          metric == "wind",
		"sun":           metric == "sun",
	}
	filtered := make([]models.WeatherArchiveEvent, 0, len(events))
	for _, event := range events {
		if allowed[event.Group] {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func resolveArchiveDaySearch(field, comparison, thresholdParam string) (models.WeatherArchiveDaySearch, error) {
	field = strings.ToLower(strings.TrimSpace(field))
	if field == "" {
		return models.WeatherArchiveDaySearch{}, nil
	}
	labels := map[string]string{
		"temp_max": "максимальная температура",
		"temp_min": "минимальная температура",
		"rain":     "осадки за сутки",
		"gust":     "максимальный порыв",
		"uv":       "максимальный UV-индекс",
	}
	label, ok := labels[field]
	if !ok {
		return models.WeatherArchiveDaySearch{}, ErrInvalidArchiveSearch
	}
	comparison = strings.ToLower(strings.TrimSpace(comparison))
	if comparison != "gte" && comparison != "lte" {
		return models.WeatherArchiveDaySearch{}, ErrInvalidArchiveSearch
	}
	threshold, err := strconv.ParseFloat(strings.TrimSpace(thresholdParam), 64)
	if err != nil || math.IsNaN(threshold) || math.IsInf(threshold, 0) {
		return models.WeatherArchiveDaySearch{}, ErrInvalidArchiveSearch
	}
	operator := "≥"
	if comparison == "lte" {
		operator = "≤"
	}
	units := map[string]string{"temp_max": "°C", "temp_min": "°C", "rain": "мм", "gust": "м/с", "uv": ""}
	description := fmt.Sprintf("%s %s %s", label, operator, formatArchiveSearchThreshold(threshold, units[field]))
	return models.WeatherArchiveDaySearch{Active: true, Field: field, Comparison: comparison, Threshold: threshold, Description: description}, nil
}

func formatArchiveSearchThreshold(value float64, unit string) string {
	formatted := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
	if unit != "" {
		formatted += " " + unit
	}
	return formatted
}

func filterArchiveDays(days []models.DailyWeatherInsight, search models.WeatherArchiveDaySearch) []models.DailyWeatherInsight {
	if !search.Active {
		return days
	}
	matches := make([]models.DailyWeatherInsight, 0)
	for _, day := range days {
		value, ok := archiveDaySearchValue(day, search.Field)
		if !ok {
			continue
		}
		matchesCondition := value >= search.Threshold
		if search.Comparison == "lte" {
			matchesCondition = value <= search.Threshold
		}
		if matchesCondition {
			matches = append(matches, day)
		}
	}
	return matches
}

func archiveDaySearchValue(day models.DailyWeatherInsight, field string) (float64, bool) {
	switch field {
	case "temp_max":
		if day.TempMax != nil {
			return float64(*day.TempMax), true
		}
	case "temp_min":
		if day.TempMin != nil {
			return float64(*day.TempMin), true
		}
	case "rain":
		if day.RainTotal != nil {
			return float64(*day.RainTotal), true
		}
	case "gust":
		if day.WindGustMax != nil {
			return float64(*day.WindGustMax), true
		}
	case "uv":
		if day.UVIndexMax != nil {
			return float64(*day.UVIndexMax), true
		}
	}
	return 0, false
}

func normalizeArchiveMetric(metric string) string {
	switch metric {
	case "temperature", "precipitation", "wind", "air", "sun":
		return metric
	default:
		return "all"
	}
}

func buildArchiveSummary(days []models.DailyWeatherInsight, daysInPeriod int) models.WeatherArchiveSummary {
	summary := models.WeatherArchiveSummary{DaysInPeriod: daysInPeriod}
	var tempSum, humiditySum, pressureSum float64
	var tempCount, humidityCount, pressureCount int

	for _, day := range days {
		summary.DaysWithData++
		if day.TempMin != nil && day.TempMax != nil {
			min, max := float64(*day.TempMin), float64(*day.TempMax)
			if !summary.HasTemp || min < summary.TempMin {
				summary.TempMin = min
			}
			if !summary.HasTemp || max > summary.TempMax {
				summary.TempMax = max
			}
			summary.HasTemp = true
		}
		if day.TempAvg != nil {
			tempSum += float64(*day.TempAvg)
			tempCount++
		}
		if day.RainTotal != nil {
			summary.RainTotal += float64(*day.RainTotal)
			if *day.RainTotal >= rainyDayRainThreshold {
				summary.RainDays++
			}
			summary.HasRain = true
		}
		if day.WindSpeedMax != nil {
			summary.WindSpeedMax = math.Max(summary.WindSpeedMax, float64(*day.WindSpeedMax))
			summary.HasWind = true
		}
		if day.WindGustMax != nil {
			summary.WindGustMax = math.Max(summary.WindGustMax, float64(*day.WindGustMax))
			summary.HasWind = true
		}
		if day.HumidityAvg != nil {
			humiditySum += float64(*day.HumidityAvg)
			humidityCount++
		}
		if day.PressureAvg != nil {
			pressureSum += float64(*day.PressureAvg)
			pressureCount++
		}
		if day.HumidityAvg != nil || day.PressureAvg != nil {
			summary.HasAir = true
		}
		if day.UVIndexMax != nil {
			summary.UVIndexMax = math.Max(summary.UVIndexMax, float64(*day.UVIndexMax))
			summary.HasSun = true
		}
		if day.SolarRadiationMax != nil {
			summary.SolarRadiationMax = math.Max(summary.SolarRadiationMax, float64(*day.SolarRadiationMax))
			summary.HasSun = true
		}
	}
	if tempCount > 0 {
		summary.TempAvg = tempSum / float64(tempCount)
	}
	if humidityCount > 0 {
		summary.HumidityAvg = humiditySum / float64(humidityCount)
	}
	if pressureCount > 0 {
		summary.PressureAvg = pressureSum / float64(pressureCount)
	}
	return summary
}
