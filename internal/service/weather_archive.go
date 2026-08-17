package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

var (
	ErrInvalidArchivePeriod = errors.New("invalid archive period")
	ErrInvalidArchiveRange  = errors.New("invalid archive date range")
)

const archiveFirstYear = 2023

// GetArchive returns station observations for a calendar period or an inclusive custom range.
// Comparisons always use a previous equivalent period; this is not a climatic normal.
func (s *WeatherService) GetArchive(ctx context.Context, period, metric, monthParam, seasonParam, yearParam, fromParam, toParam string) (*models.WeatherArchivePage, error) {
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

	start, end, label, previousLabel, err := resolveArchivePeriod(period, monthParam, seasonParam, yearParam, fromParam, toParam, now, loc)
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

	previousStart, previousEnd := archivePreviousPeriod(period, start, end, daysInPeriod, loc)
	previousDays, err := s.repo.GetDailyInsights(ctx, previousStart, previousEnd, s.timezone)
	if err != nil {
		return nil, err
	}

	summary := buildArchiveSummary(currentDays, daysInPeriod)
	previous := buildArchiveSummary(previousDays, daysBetween(previousStart, previousEnd))
	page := &models.WeatherArchivePage{
		GeneratedAt:         now,
		Period:              period,
		Metric:              metric,
		PeriodLabel:         label,
		FromParam:           start.Format("2006-01-02"),
		ToParam:             calendarEnd.AddDate(0, 0, -1).Format("2006-01-02"),
		MonthParam:          start.Format("2006-01"),
		SeasonParam:         seasonParamForDate(start, loc),
		YearParam:           start.Year(),
		SeasonOptions:       archiveSeasonOptions(now, loc),
		YearOptions:         archiveYearOptions(now.Year()),
		Summary:             summary,
		PreviousSummary:     previous,
		PreviousPeriodLabel: previousLabel,
		HasPreviousPeriod:   previous.DaysWithData > 0,
		TemperatureDelta:    summary.TempAvg - previous.TempAvg,
		PrecipitationDelta:  summary.RainTotal - previous.RainTotal,
		WindGustDelta:       summary.WindGustMax - previous.WindGustMax,
		HumidityDelta:       summary.HumidityAvg - previous.HumidityAvg,
		PressureDelta:       summary.PressureAvg - previous.PressureAvg,
		Daily:               currentDays,
		Chart:               buildArchiveChart(currentDays, metric),
	}
	return page, nil
}

func resolveArchivePeriod(period, monthParam, seasonParam, yearParam, fromParam, toParam string, now time.Time, loc *time.Location) (time.Time, time.Time, string, string, error) {
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	switch period {
	case "month":
		start := currentMonth
		if monthParam != "" {
			parsed, err := time.ParseInLocation("2006-01", monthParam, loc)
			if err != nil {
				return time.Time{}, time.Time{}, "", "", ErrInvalidArchivePeriod
			}
			start = parsed
		}
		if start.After(currentMonth) {
			return time.Time{}, time.Time{}, "", "", ErrInvalidArchiveRange
		}
		return start, start.AddDate(0, 1, 0), russianMonthYear(start), "Тот же месяц год назад", nil
	case "season":
		start, end := seasonBounds(now, loc)
		selectedYear, selectedCode := seasonIDFromStart(start)
		if seasonParam != "" {
			parsedYear, parsedCode, err := parseSeasonParam(seasonParam)
			if err != nil {
				return time.Time{}, time.Time{}, "", "", ErrInvalidArchivePeriod
			}
			candidateStart, candidateEnd := seasonBoundsByID(parsedYear, parsedCode, loc)
			if candidateStart.After(start) {
				return time.Time{}, time.Time{}, "", "", ErrInvalidArchiveRange
			}
			start, end, selectedYear, selectedCode = candidateStart, candidateEnd, parsedYear, parsedCode
		}
		return start, end, seasonLabel(selectedYear, selectedCode), "Тот же сезон год назад", nil
	case "year":
		year := now.Year()
		if yearParam != "" {
			if _, err := fmt.Sscanf(yearParam, "%d", &year); err != nil || year < archiveFirstYear || year > now.Year() {
				return time.Time{}, time.Time{}, "", "", ErrInvalidArchivePeriod
			}
		}
		start := time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(1, 0, 0), fmt.Sprintf("%d год", year), "Предыдущий год", nil
	case "range":
		start, errStart := time.ParseInLocation("2006-01-02", fromParam, loc)
		endDate, errEnd := time.ParseInLocation("2006-01-02", toParam, loc)
		if errStart != nil || errEnd != nil || endDate.Before(start) || daysBetween(start, endDate.AddDate(0, 0, 1)) > 366 {
			return time.Time{}, time.Time{}, "", "", ErrInvalidArchiveRange
		}
		return start, endDate.AddDate(0, 0, 1), fmt.Sprintf("%s — %s", start.Format("02.01.2006"), endDate.Format("02.01.2006")), "Предыдущий равный диапазон", nil
	default:
		return time.Time{}, time.Time{}, "", "", ErrInvalidArchivePeriod
	}
}

func archivePreviousPeriod(period string, start, end time.Time, days int, loc *time.Location) (time.Time, time.Time) {
	switch period {
	case "month":
		previousStart := start.AddDate(-1, 0, 0)
		previousEnd := minTime(previousStart.AddDate(0, 0, days), previousStart.AddDate(0, 1, 0))
		return previousStart, previousEnd
	case "season":
		year, code := seasonIDFromStart(start)
		previousStart, previousFullEnd := seasonBoundsByID(year-1, code, loc)
		return previousStart, minTime(previousStart.AddDate(0, 0, days), previousFullEnd)
	case "year":
		previousStart := start.AddDate(-1, 0, 0)
		return previousStart, previousStart.AddDate(0, 0, days)
	default: // custom range: immediately preceding range of equal length
		return start.AddDate(0, 0, -days), start
	}
}

func archiveSeasonOptions(now time.Time, loc *time.Location) []models.WeatherInsightsPeriodOption {
	start, _ := seasonBounds(now, loc)
	year, code := seasonIDFromStart(start)
	return buildSeasonOptions(year, code)
}

func archiveYearOptions(currentYear int) []int {
	years := make([]int, 0, currentYear-archiveFirstYear+1)
	for year := currentYear; year >= archiveFirstYear; year-- {
		years = append(years, year)
	}
	return years
}

func seasonParamForDate(date time.Time, loc *time.Location) string {
	start, _ := seasonBounds(date, loc)
	year, code := seasonIDFromStart(start)
	return formatSeasonParam(year, code)
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

func buildArchiveChart(days []models.DailyWeatherInsight, metric string) models.WeatherArchiveChart {
	chart := models.WeatherArchiveChart{Labels: make([]string, 0, len(days)), Values: make([]float64, 0, len(days)), Type: "line"}
	if metric == "precipitation" {
		chart.Label, chart.Unit, chart.Type = "Осадки за сутки", "мм", "bar"
	} else if metric == "wind" {
		chart.Label, chart.Unit = "Максимальный порыв", "м/с"
	} else if metric == "air" {
		chart.Label, chart.Unit = "Среднее давление", "гПа"
	} else if metric == "sun" {
		chart.Label, chart.Unit = "Максимальный UV", "UV"
	} else {
		chart.Label, chart.Unit = "Средняя температура", "°C"
		chart.Secondary = make([]float64, 0, len(days))
	}
	for _, day := range days {
		chart.Labels = append(chart.Labels, day.Date.Format("02.01"))
		switch metric {
		case "precipitation":
			chart.Values = append(chart.Values, archiveFloat32(day.RainTotal))
		case "wind":
			chart.Values = append(chart.Values, archiveFloat32(day.WindGustMax))
		case "air":
			chart.Values = append(chart.Values, archiveFloat32(day.PressureAvg))
		case "sun":
			chart.Values = append(chart.Values, archiveFloat32(day.UVIndexMax))
		default:
			chart.Values = append(chart.Values, archiveFloat32(day.TempAvg))
			chart.Secondary = append(chart.Secondary, archiveFloat32(day.TempMax))
		}
	}
	return chart
}

func archiveFloat32(value *float32) float64 {
	if value == nil {
		return 0
	}
	return float64(*value)
}
