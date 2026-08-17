package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

// GetArchive returns daily station measurements and a comparison with the preceding equivalent calendar period.
// The anchor is normalized to the first day of its month in the configured station timezone.
func (s *WeatherService) GetArchive(ctx context.Context, period string, anchor time.Time) (*models.WeatherArchive, error) {
	loc := s.location
	if loc == nil {
		loc = time.Local
	}

	now := time.Now().In(loc)
	start, end, previousStart, previousEnd, label, err := archivePeriodBounds(period, anchor, now, loc)
	if err != nil {
		return nil, err
	}

	days, err := s.repo.GetDailyInsights(ctx, start, end, s.timezone)
	if err != nil {
		return nil, err
	}
	previousDays, err := s.repo.GetDailyInsights(ctx, previousStart, previousEnd, s.timezone)
	if err != nil {
		return nil, err
	}

	daysInPeriod := calendarDaysInRange(start, end, loc)
	archiveDays := make([]models.WeatherArchiveDay, 0, len(days))
	for _, day := range days {
		archiveDays = append(archiveDays, models.WeatherArchiveDay{
			Date:      day.Date,
			TempMin:   day.TempMin,
			TempAvg:   day.TempAvg,
			TempMax:   day.TempMax,
			RainTotal: day.RainTotal,
		})
	}

	return &models.WeatherArchive{
		Period:    period,
		Label:     label,
		StartDate: start,
		EndDate:   archiveDisplayEnd(end, loc),
		Summary:   buildArchiveSummary(days, daysInPeriod),
		Comparison: models.WeatherArchiveComparison{
			Label:     "предыдущий аналогичный период",
			Available: len(previousDays) > 0,
			Summary:   buildArchiveSummary(previousDays, calendarDaysInRange(previousStart, previousEnd, loc)),
		},
		Days: archiveDays,
	}, nil
}

func archivePeriodBounds(period string, anchor, now time.Time, loc *time.Location) (start, end, previousStart, previousEnd time.Time, label string, err error) {
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	if anchor.IsZero() {
		anchor = currentMonth
	} else {
		anchor = anchor.In(loc)
		anchor = time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, loc)
		if anchor.After(currentMonth) {
			anchor = currentMonth
		}
	}

	switch period {
	case "month":
		start = anchor
		end = start.AddDate(0, 1, 0)
		previousStart = start.AddDate(0, -1, 0)
		previousEnd = start
		label = russianMonthYear(start)
	case "season":
		start, end = seasonBounds(anchor, loc)
		previousStart, previousEnd = seasonBounds(start.AddDate(0, -1, 0), loc)
		seasonYear, seasonCode := seasonIDFromStart(start)
		label = seasonLabel(seasonYear, seasonCode)
	case "year":
		start = time.Date(anchor.Year(), time.January, 1, 0, 0, 0, 0, loc)
		end = start.AddDate(1, 0, 0)
		previousStart = start.AddDate(-1, 0, 0)
		previousEnd = start
		label = fmt.Sprintf("%d год", start.Year())
	default:
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, "", fmt.Errorf("unsupported archive period %q", period)
	}

	if end.After(now) {
		end = now
	}

	// A current, still-running period is compared with the same number of
	// calendar days immediately preceding it, not with an entire prior period.
	// A completed period keeps its full preceding equivalent period.
	previousEnd = previousStart.AddDate(0, 0, calendarDaysInRange(start, end, loc))
	return start, end, previousStart, previousEnd, label, nil
}

func calendarDaysInRange(start, end time.Time, loc *time.Location) int {
	if !end.After(start) {
		return 0
	}
	lastDay := dayStart(end, loc)
	if end.Equal(lastDay) {
		lastDay = lastDay.AddDate(0, 0, -1)
	}
	return daysBetween(dayStart(start, loc), lastDay.AddDate(0, 0, 1))
}

func archiveDisplayEnd(end time.Time, loc *time.Location) time.Time {
	day := dayStart(end, loc)
	if end.Equal(day) {
		return day.Add(-time.Nanosecond)
	}
	return end
}

func buildArchiveSummary(days []models.DailyWeatherInsight, daysInPeriod int) models.WeatherArchiveSummary {
	result := models.WeatherArchiveSummary{
		DaysInPeriod: daysInPeriod,
		DaysWithData: len(days),
	}
	if daysInPeriod > 0 {
		result.CoveragePercent = int(math.Round(float64(len(days)) / float64(daysInPeriod) * 100))
	}

	var tempSum float64
	var tempCount int
	var min, max float64
	var hasMin, hasMax bool

	for _, day := range days {
		if day.TempAvg != nil {
			tempSum += float64(*day.TempAvg)
			tempCount++
		}
		if day.TempMin != nil && (!hasMin || float64(*day.TempMin) < min) {
			min = float64(*day.TempMin)
			hasMin = true
		}
		if day.TempMax != nil && (!hasMax || float64(*day.TempMax) > max) {
			max = float64(*day.TempMax)
			hasMax = true
		}
		if day.RainTotal != nil {
			result.RainTotal += float64(*day.RainTotal)
		}
	}

	if tempCount > 0 {
		average := tempSum / float64(tempCount)
		result.TempAvg = &average
	}
	if hasMin {
		result.TempMin = &min
	}
	if hasMax {
		result.TempMax = &max
	}
	return result
}
