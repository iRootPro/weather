package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

var ErrInvalidInsightSeason = errors.New("invalid insight season")

const (
	wetDayRainThreshold   = 0.2
	rainyDayRainThreshold = 1.0
	heavyRainThreshold    = 10.0
	hotDayTempThreshold   = 30.0
	veryHotTempThreshold  = 35.0
	frostDayTempThreshold = 0.0
	windyGustThreshold    = 10.0
	strongGustThreshold   = 15.0
	sunnySolarThreshold   = 500.0
	cloudySolarThreshold  = 150.0
	highUVThreshold       = 6.0
)

// GetInsights returns human-friendly monthly weather analytics for the current month.
func (s *WeatherService) GetInsights(ctx context.Context) (*models.WeatherInsightsPage, error) {
	return s.GetInsightsForMonth(ctx, time.Time{})
}

// GetInsightsForMonth returns human-friendly weather analytics for the selected calendar month.
// If month is zero or in the future, the current month is used.
func (s *WeatherService) GetInsightsForMonth(ctx context.Context, month time.Time) (*models.WeatherInsightsPage, error) {
	loc := s.location
	if loc == nil {
		loc = time.Local
	}

	now := time.Now().In(loc)
	actualCurrentStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	selectedStart := actualCurrentStart
	if !month.IsZero() {
		month = month.In(loc)
		candidate := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, loc)
		if !candidate.After(actualCurrentStart) {
			selectedStart = candidate
		}
	}
	currentStart := selectedStart
	currentEnd := currentStart.AddDate(0, 1, 0)
	isCurrentMonth := currentStart.Equal(actualCurrentStart)
	periodEnd := currentEnd
	if isCurrentMonth {
		periodEnd = now
	}
	previousStart := currentStart.AddDate(0, -1, 0)
	previousEnd := currentStart

	currentDays, err := s.repo.GetDailyInsights(ctx, currentStart, periodEnd, s.timezone)
	if err != nil {
		return nil, err
	}
	previousDays, err := s.repo.GetDailyInsights(ctx, previousStart, previousEnd, s.timezone)
	if err != nil {
		return nil, err
	}

	rollingAnchor := dayStart(periodEnd, loc).AddDate(0, 0, 1)
	if !isCurrentMonth {
		rollingAnchor = currentEnd
	}
	rollingStart := rollingAnchor.AddDate(0, 0, -59)
	rollingDays, err := s.repo.GetDailyInsights(ctx, rollingStart, rollingAnchor, s.timezone)
	if err != nil {
		return nil, err
	}

	archiveDays := make([]models.DailyWeatherInsight, 0, 366)
	for year := currentStart.Year() - 10; year < currentStart.Year(); year++ {
		archiveMonthStart := time.Date(year, currentStart.Month(), 1, 0, 0, 0, 0, loc)
		archiveMonthEnd := archiveMonthStart.AddDate(0, 1, 0)
		days, err := s.repo.GetDailyInsights(ctx, archiveMonthStart, archiveMonthEnd, s.timezone)
		if err != nil {
			return nil, err
		}
		archiveDays = append(archiveDays, days...)
	}

	seasonStart, seasonEnd := seasonBounds(currentStart, loc)
	seasonPeriodEnd := minTime(periodEnd, seasonEnd)
	seasonDays, err := s.repo.GetDailyInsights(ctx, seasonStart, seasonPeriodEnd, s.timezone)
	if err != nil {
		return nil, err
	}

	daysInSelectedPeriod := daysBetween(currentStart, currentEnd)
	analysisDate := currentEnd.Add(-time.Nanosecond)
	if isCurrentMonth {
		daysInSelectedPeriod = now.Day()
		analysisDate = now
	}
	previousCompareDays := minInt(daysInSelectedPeriod, daysBetween(previousStart, previousEnd))
	previousSameEnd := previousStart.AddDate(0, 0, previousCompareDays)
	previousSameDays := filterDaysBefore(previousDays, previousSameEnd)

	selectedMonthLabel := russianMonthYear(currentStart)
	currentTitle := selectedMonthLabel
	if isCurrentMonth {
		currentTitle = "Этот месяц"
	}
	current := buildMonthlyInsights(currentTitle, currentStart, periodEnd, currentDays, daysInSelectedPeriod)
	previous := buildMonthlyInsights("Прошлый месяц · справочно", previousStart, previousEnd.Add(-time.Nanosecond), previousDays, daysBetween(previousStart, previousEnd))
	previousSame := buildMonthlyInsights(fmt.Sprintf("Прошлый месяц к %d числу", previousCompareDays), previousStart, previousSameEnd.Add(-time.Nanosecond), previousSameDays, previousCompareDays)
	seasonCurrent := buildMonthlyInsights("Текущий сезон", seasonStart, seasonPeriodEnd, seasonDays, maxInt(1, daysBetween(seasonStart, seasonPeriodEnd)))
	season := buildSeasonContext(analysisDate, seasonStart, seasonEnd, seasonCurrent)
	selectedSeasonYear, selectedSeasonCode := seasonIDFromStart(seasonStart)
	actualSeasonStart, _ := seasonBounds(now, loc)
	actualSeasonYear, actualSeasonCode := seasonIDFromStart(actualSeasonStart)
	sameMonthBenchmark := buildSameMonthBenchmark(analysisDate, current, archiveDays, loc)
	last7Title := "Последние 7 дней"
	last30Title := "Последние 30 дней"
	if !isCurrentMonth {
		last7Title = "Финальные 7 дней месяца"
		last30Title = "Последние 30 дней периода"
	}
	last7Days := buildRollingPeriod(last7Title, "Сравнение с предыдущими 7 днями", rollingDays, rollingAnchor.Add(-time.Nanosecond), 7, loc)
	last30Days := buildRollingPeriod(last30Title, "Сравнение с предыдущими 30 днями", rollingDays, rollingAnchor.Add(-time.Nanosecond), 30, loc)

	allDays := append([]models.DailyWeatherInsight{}, rollingDays...)
	allDays = append(allDays, currentDays...)
	currentDryStreak, lastRainDate, hasLastRain := calculateRainRecency(allDays, loc)

	mainInsight, stories := buildInsightStories(current, previous, previousSame, sameMonthBenchmark, last7Days, season, currentDryStreak, lastRainDate, hasLastRain, "месяца")
	bestDay, worstDay := findNotableDays(currentDays)
	dayTypes, dominantDayType := buildDayTypeSummaries(currentDays)
	windInsight := buildWindInsight(current)
	uvInsight := buildUVInsight(currentDays, current)
	timeline := buildTimelineEvents(current, currentDays, bestDay, worstDay)

	page := &models.WeatherInsightsPage{
		GeneratedAt:               now,
		IsSeason:                  false,
		CurrentPeriodLabel:        "месяц",
		CurrentPeriodGenitive:     "месяца",
		CurrentPeriodPreposition:  "месяце",
		SelectedMonthParam:        currentStart.Format("2006-01"),
		SelectedMonthLabel:        selectedMonthLabel,
		SelectedSeasonParam:       formatSeasonParam(selectedSeasonYear, selectedSeasonCode),
		SelectedSeasonLabel:       seasonLabel(selectedSeasonYear, selectedSeasonCode),
		SeasonOptions:             buildSeasonOptions(actualSeasonYear, actualSeasonCode),
		PreviousMonthParam:        currentStart.AddDate(0, -1, 0).Format("2006-01"),
		NextMonthParam:            currentStart.AddDate(0, 1, 0).Format("2006-01"),
		HasNextMonth:              currentStart.Before(actualCurrentStart),
		IsCurrentMonth:            isCurrentMonth,
		PeriodStatus:              insightPeriodStatus(isCurrentMonth, false),
		CurrentMonth:              current,
		PreviousMonth:             previous,
		PreviousSamePeriod:        previousSame,
		Season:                    season,
		SameMonthBenchmark:        sameMonthBenchmark,
		Last7Days:                 last7Days,
		Last30Days:                last30Days,
		DayTypes:                  dayTypes,
		DominantDayType:           dominantDayType,
		WindInsight:               windInsight,
		UVInsight:                 uvInsight,
		Timeline:                  timeline,
		CurrentDryStreak:          currentDryStreak,
		HasLastRain:               hasLastRain,
		MainInsight:               mainInsight,
		Stories:                   stories,
		MonthProgressPercent:      clampPercent(float64(daysInSelectedPeriod) / float64(daysBetween(currentStart, currentEnd)) * 100),
		RainVsPreviousSamePercent: clampPercent(ratioPercent(current.RainTotal, previousSame.RainTotal)),
		RainVsPreviousFullPercent: clampPercent(ratioPercent(current.RainTotal, previous.RainTotal)),
		RainiestDaySharePercent:   rainiestSharePercent(current),
		ComfortPercent:            clampPercent(ratioPercent(float64(current.ComfortableDays), float64(maxInt(current.DaysWithData, 1)))),
		SunnyPercent:              clampPercent(ratioPercent(float64(current.SunnyDays), float64(maxInt(current.DaysWithData, 1)))),
		RainDaysPercent:           clampPercent(ratioPercent(float64(current.RainDays), float64(maxInt(current.DaysWithData, 1)))),
		Calendar:                  buildCalendar(currentStart, currentEnd, currentDays, now, loc),
		RainChartData:             buildRainChartData(currentDays, previousDays, archiveDays, daysBetween(currentStart, currentEnd), daysBetween(previousStart, previousEnd), daysInSelectedPeriod, currentStart.Month(), loc),
		BestDay:                   bestDay,
		WorstDay:                  worstDay,
	}
	if hasLastRain {
		page.LastRainDate = lastRainDate
		page.DaysSinceLastRain = int(dayStart(analysisDate, loc).Sub(dayStart(lastRainDate, loc)).Hours() / 24)
	}

	return page, nil
}

// GetInsightsForSeason returns a full-season weather report.
// seasonParam format: YYYY-winter, YYYY-spring, YYYY-summer or YYYY-autumn.
// Winter uses the civil year in which January and February fall: 2026-winter = Dec 2025 — Feb 2026.
func (s *WeatherService) GetInsightsForSeason(ctx context.Context, seasonParam string) (*models.WeatherInsightsPage, error) {
	loc := s.location
	if loc == nil {
		loc = time.Local
	}

	now := time.Now().In(loc)
	actualCurrentStart, _ := seasonBounds(now, loc)
	actualCurrentID, actualCurrentCode := seasonIDFromStart(actualCurrentStart)
	selectedYear, selectedCode := actualCurrentID, actualCurrentCode
	if seasonParam != "" {
		parsedYear, parsedCode, err := parseSeasonParam(seasonParam)
		if err != nil {
			return nil, err
		}
		candidateStart, _ := seasonBoundsByID(parsedYear, parsedCode, loc)
		if !candidateStart.After(actualCurrentStart) {
			selectedYear, selectedCode = parsedYear, parsedCode
		}
	}

	currentStart, currentEnd := seasonBoundsByID(selectedYear, selectedCode, loc)
	isCurrentSeason := currentStart.Equal(actualCurrentStart)
	periodEnd := currentEnd
	if isCurrentSeason {
		periodEnd = now
	}
	analysisDate := currentEnd.Add(-time.Nanosecond)
	daysInSelectedPeriod := daysBetween(currentStart, currentEnd)
	if isCurrentSeason {
		analysisDate = now
		daysInSelectedPeriod = maxInt(1, daysBetween(currentStart, dayStart(now, loc).AddDate(0, 0, 1)))
	}

	previousYear, previousCode := shiftSeasonID(selectedYear, selectedCode, -1)
	previousStart, previousEnd := seasonBoundsByID(previousYear, previousCode, loc)
	previousCompareDays := minInt(daysInSelectedPeriod, daysBetween(previousStart, previousEnd))
	previousSameEnd := previousStart.AddDate(0, 0, previousCompareDays)

	currentDays, err := s.repo.GetDailyInsights(ctx, currentStart, periodEnd, s.timezone)
	if err != nil {
		return nil, err
	}
	previousDays, err := s.repo.GetDailyInsights(ctx, previousStart, previousEnd, s.timezone)
	if err != nil {
		return nil, err
	}
	previousSameDays := filterDaysBefore(previousDays, previousSameEnd)

	rollingAnchor := dayStart(periodEnd, loc).AddDate(0, 0, 1)
	if !isCurrentSeason {
		rollingAnchor = currentEnd
	}
	rollingStart := rollingAnchor.AddDate(0, 0, -59)
	rollingDays, err := s.repo.GetDailyInsights(ctx, rollingStart, rollingAnchor, s.timezone)
	if err != nil {
		return nil, err
	}

	archiveDays := make([]models.DailyWeatherInsight, 0, 920)
	for year := selectedYear - 10; year < selectedYear; year++ {
		archiveStart, archiveEnd := seasonBoundsByID(year, selectedCode, loc)
		days, err := s.repo.GetDailyInsights(ctx, archiveStart, archiveEnd, s.timezone)
		if err != nil {
			return nil, err
		}
		archiveDays = append(archiveDays, days...)
	}

	selectedSeasonLabel := seasonLabel(selectedYear, selectedCode)
	currentTitle := selectedSeasonLabel
	if isCurrentSeason {
		currentTitle = "Этот сезон"
	}
	current := buildMonthlyInsights(currentTitle, currentStart, periodEnd, currentDays, daysInSelectedPeriod)
	previous := buildMonthlyInsights("Прошлый сезон · справочно", previousStart, previousEnd.Add(-time.Nanosecond), previousDays, daysBetween(previousStart, previousEnd))
	previousSame := buildMonthlyInsights(fmt.Sprintf("Прошлый сезон к %d дню", previousCompareDays), previousStart, previousSameEnd.Add(-time.Nanosecond), previousSameDays, previousCompareDays)
	season := buildSeasonContext(analysisDate, currentStart, currentEnd, current)
	benchmark := buildSameSeasonBenchmark(current, archiveDays, selectedYear, selectedCode, daysInSelectedPeriod, loc)
	last7Title := "Последние 7 дней"
	last30Title := "Последние 30 дней"
	if !isCurrentSeason {
		last7Title = "Финальные 7 дней сезона"
		last30Title = "Последние 30 дней сезона"
	}
	last7Days := buildRollingPeriod(last7Title, "Сравнение с предыдущими 7 днями", rollingDays, rollingAnchor.Add(-time.Nanosecond), 7, loc)
	last30Days := buildRollingPeriod(last30Title, "Сравнение с предыдущими 30 днями", rollingDays, rollingAnchor.Add(-time.Nanosecond), 30, loc)

	allDays := append([]models.DailyWeatherInsight{}, rollingDays...)
	allDays = append(allDays, currentDays...)
	currentDryStreak, lastRainDate, hasLastRain := calculateRainRecency(allDays, loc)

	mainInsight, stories := buildInsightStories(current, previous, previousSame, benchmark, last7Days, season, currentDryStreak, lastRainDate, hasLastRain, "сезона")
	bestDay, worstDay := findNotableDays(currentDays)
	dayTypes, dominantDayType := buildDayTypeSummaries(currentDays)
	windInsight := buildWindInsight(current)
	uvInsight := buildUVInsight(currentDays, current)
	timeline := buildTimelineEvents(current, currentDays, bestDay, worstDay)
	seasonMonths := buildSeasonMonthCards(currentStart, periodEnd, currentDays, loc)
	previousSeasonParam := formatSeasonParam(previousYear, previousCode)
	nextYear, nextCode := shiftSeasonID(selectedYear, selectedCode, 1)
	nextStart, _ := seasonBoundsByID(nextYear, nextCode, loc)

	page := &models.WeatherInsightsPage{
		GeneratedAt:               now,
		IsSeason:                  true,
		CurrentPeriodLabel:        "сезон",
		CurrentPeriodGenitive:     "сезона",
		CurrentPeriodPreposition:  "сезоне",
		SelectedMonthParam:        currentStart.Format("2006-01"),
		SelectedMonthLabel:        russianMonthYear(currentStart),
		PreviousMonthParam:        currentStart.AddDate(0, -1, 0).Format("2006-01"),
		NextMonthParam:            currentStart.AddDate(0, 1, 0).Format("2006-01"),
		HasNextMonth:              false,
		IsCurrentMonth:            isCurrentSeason,
		SelectedSeasonParam:       formatSeasonParam(selectedYear, selectedCode),
		SelectedSeasonLabel:       selectedSeasonLabel,
		PreviousSeasonParam:       previousSeasonParam,
		NextSeasonParam:           formatSeasonParam(nextYear, nextCode),
		HasNextSeason:             nextStart.Before(actualCurrentStart) || nextStart.Equal(actualCurrentStart),
		SeasonOptions:             ensureSeasonOption(buildSeasonOptions(actualCurrentID, actualCurrentCode), selectedYear, selectedCode),
		PeriodStatus:              insightPeriodStatus(isCurrentSeason, true),
		CurrentMonth:              current,
		PreviousMonth:             previous,
		PreviousSamePeriod:        previousSame,
		Season:                    season,
		SameMonthBenchmark:        benchmark,
		Last7Days:                 last7Days,
		Last30Days:                last30Days,
		DayTypes:                  dayTypes,
		DominantDayType:           dominantDayType,
		WindInsight:               windInsight,
		UVInsight:                 uvInsight,
		Timeline:                  timeline,
		SeasonMonths:              seasonMonths,
		CurrentDryStreak:          currentDryStreak,
		HasLastRain:               hasLastRain,
		MainInsight:               mainInsight,
		Stories:                   stories,
		MonthProgressPercent:      clampPercent(float64(daysInSelectedPeriod) / float64(daysBetween(currentStart, currentEnd)) * 100),
		RainVsPreviousSamePercent: clampPercent(ratioPercent(current.RainTotal, previousSame.RainTotal)),
		RainVsPreviousFullPercent: clampPercent(ratioPercent(current.RainTotal, previous.RainTotal)),
		RainiestDaySharePercent:   rainiestSharePercent(current),
		ComfortPercent:            clampPercent(ratioPercent(float64(current.ComfortableDays), float64(maxInt(current.DaysWithData, 1)))),
		SunnyPercent:              clampPercent(ratioPercent(float64(current.SunnyDays), float64(maxInt(current.DaysWithData, 1)))),
		RainDaysPercent:           clampPercent(ratioPercent(float64(current.RainDays), float64(maxInt(current.DaysWithData, 1)))),
		Calendar:                  nil,
		RainChartData:             buildSeasonRainChartData(currentDays, previousDays, archiveDays, currentStart, currentEnd, periodEnd, previousStart, previousEnd, selectedCode, loc),
		BestDay:                   bestDay,
		WorstDay:                  worstDay,
	}
	if hasLastRain {
		page.LastRainDate = lastRainDate
		page.DaysSinceLastRain = int(dayStart(analysisDate, loc).Sub(dayStart(lastRainDate, loc)).Hours() / 24)
	}

	return page, nil
}

func buildMonthlyInsights(title string, start, end time.Time, days []models.DailyWeatherInsight, daysInPeriod int) models.MonthlyWeatherInsights {
	result := models.MonthlyWeatherInsights{
		Title:        title,
		StartDate:    start,
		EndDate:      end,
		DaysInPeriod: daysInPeriod,
		DaysWithData: len(days),
	}

	var tempSum float64
	var tempCount int

	for _, day := range days {
		rain := value32(day.RainTotal)
		gust := value32(day.WindGustMax)
		solar := value32(day.SolarRadiationMax)
		uv := value32(day.UVIndexMax)
		tempMin := value32(day.TempMin)
		tempMax := value32(day.TempMax)
		tempAvg := value32(day.TempAvg)
		rainRate := value32(day.RainRateMax)

		if day.RainTotal != nil {
			result.RainTotal += rain
			if rain >= wetDayRainThreshold {
				result.WetDays++
			} else {
				result.DryDays++
			}
			if rain >= rainyDayRainThreshold {
				result.RainDays++
			}
			if rain >= wetDayRainThreshold && (result.MaxRainDay == nil || rain > result.MaxRainDay.Value) {
				result.MaxRainDay = &models.DayInsightValue{Date: day.Date, Value: rain}
			}
		} else {
			result.DryDays++
		}

		if day.RainRateMax != nil && rainRate > result.MaxRainRate {
			result.MaxRainRate = rainRate
			result.MaxRainRateAt = &models.DayInsightValue{Date: day.Date, Value: rainRate}
		}

		if day.TempAvg != nil {
			tempSum += tempAvg
			tempCount++
		}
		if day.TempMax != nil {
			if tempMax >= hotDayTempThreshold {
				result.HotDays++
			}
			if tempMax >= veryHotTempThreshold {
				result.VeryHotDays++
			}
			if result.MaxTempDay == nil || tempMax > result.MaxTempDay.Value {
				result.MaxTempDay = &models.DayInsightValue{Date: day.Date, Value: tempMax}
			}
		}
		if day.TempMin != nil {
			if tempMin < frostDayTempThreshold {
				result.FrostDays++
			}
			if result.MinTempDay == nil || tempMin < result.MinTempDay.Value {
				result.MinTempDay = &models.DayInsightValue{Date: day.Date, Value: tempMin}
			}
		}

		if day.WindGustMax != nil {
			if gust >= windyGustThreshold {
				result.WindyDays++
			}
			if gust >= strongGustThreshold {
				result.StrongWindDays++
			}
			if result.MaxWindGustDay == nil || gust > result.MaxWindGustDay.Value {
				result.MaxWindGustDay = &models.DayInsightValue{Date: day.Date, Value: gust}
			}
		}

		if day.SolarRadiationMax != nil {
			if solar >= sunnySolarThreshold {
				result.SunnyDays++
			}
			if solar < cloudySolarThreshold {
				result.CloudyDays++
			}
			if result.SunniestDay == nil || solar > result.SunniestDay.Value {
				result.SunniestDay = &models.DayInsightValue{Date: day.Date, Value: solar}
			}
		}
		if day.UVIndexMax != nil && uv >= highUVThreshold {
			result.HighUVDays++
		}

		if day.TempAvg != nil && tempAvg >= 18 && tempAvg <= 26 && rain < wetDayRainThreshold && gust < 8 {
			result.ComfortableDays++
		}
	}

	if tempCount > 0 {
		result.AvgTemp = tempSum / float64(tempCount)
	}

	return result
}

func buildCalendar(start, end time.Time, days []models.DailyWeatherInsight, now time.Time, loc *time.Location) []models.CalendarWeatherDay {
	dayByNumber := make(map[int]models.DailyWeatherInsight, len(days))
	for _, day := range days {
		dayByNumber[day.Date.In(loc).Day()] = day
	}

	cells := make([]models.CalendarWeatherDay, 0, 42)
	firstWeekdayOffset := (int(start.Weekday()) + 6) % 7 // Monday = 0
	for i := 0; i < firstWeekdayOffset; i++ {
		cells = append(cells, models.CalendarWeatherDay{IsBlank: true})
	}

	daysInMonth := daysBetween(start, end)
	for dayNum := 1; dayNum <= daysInMonth; dayNum++ {
		date := time.Date(start.Year(), start.Month(), dayNum, 0, 0, 0, 0, loc)
		cell := models.CalendarWeatherDay{
			Date:     date,
			Day:      dayNum,
			IsToday:  sameDay(date, now),
			IsFuture: date.After(dayStart(now, loc)),
		}
		if insight, ok := dayByNumber[dayNum]; ok {
			cell.HasData = true
			cell.TempMin = value32(insight.TempMin)
			cell.TempMax = value32(insight.TempMax)
			cell.RainTotal = value32(insight.RainTotal)
			cell.WindGust = value32(insight.WindGustMax)
			cell.SolarMax = value32(insight.SolarRadiationMax)
			cell.RainLevel = rainLevel(cell.RainTotal)
			cell.Badges = dayBadges(insight)
			cell.CardClass = calendarClass(insight)
			cell.Summary = fmt.Sprintf("%s: %.1f…%.1f°C, дождь %.1f мм, порыв %.1f м/с", formatInsightDate(date), cell.TempMin, cell.TempMax, cell.RainTotal, cell.WindGust)
		} else if cell.IsFuture {
			cell.CardClass = "bg-gray-50 text-gray-300 dark:bg-gray-800/50 dark:text-gray-600 border-gray-100 dark:border-gray-700"
			cell.Summary = "Будущий день"
		} else {
			cell.CardClass = "bg-gray-50 dark:bg-gray-800/50 border-gray-100 dark:border-gray-700"
			cell.Summary = "Нет данных"
		}
		cells = append(cells, cell)
	}
	return cells
}

func buildSeasonContext(now, start, end time.Time, seasonSummary models.MonthlyWeatherInsights) models.WeatherSeasonContext {
	name, code, icon := seasonName(now.Month())
	progress := clampPercent(float64(daysBetween(start, now)) / float64(maxInt(daysBetween(start, end), 1)) * 100)
	ctx := models.WeatherSeasonContext{
		Code:            code,
		Name:            name,
		Title:           fmt.Sprintf("Сезонный контекст: %s", name),
		Icon:            icon,
		ProgressPercent: progress,
	}

	switch code {
	case "winter":
		ctx.Description = fmt.Sprintf("Зимой важнее смотреть на морозные дни, оттепели, ветер и резкие перепады давления. Осадки сравнивать с соседним месяцем особенно опасно: декабрь, январь и февраль ведут себя по-разному.")
		ctx.FocusTitle = "Зимний режим наблюдений"
		ctx.FocusText = fmt.Sprintf("За сезон уже %d морозных дней и %d дней с сильными порывами.", seasonSummary.FrostDays, seasonSummary.StrongWindDays)
	case "spring":
		ctx.Description = "Весной нормальны быстрые смены сценария: сухие окна, ливни, первые жаркие дни и большие перепады температуры. Поэтому главный ориентир — такие же весенние месяцы и последние недели."
		ctx.FocusTitle = "Весна с переменным характером"
		ctx.FocusText = fmt.Sprintf("За сезон уже %d дождливых дней, %d солнечных и %d жарких.", seasonSummary.RainDays, seasonSummary.SunnyDays, seasonSummary.HotDays)
	case "summer":
		ctx.Description = "Летом важнее жара, UV, ливневые пики и комфортные дни. Соседний месяц может быть слабой базой: июнь, июль и август часто отличаются по режиму осадков."
		ctx.FocusTitle = "Летний профиль: жара и ливни"
		ctx.FocusText = fmt.Sprintf("За сезон уже %d жарких дней, %d дней с высоким UV и %.1f мм осадков.", seasonSummary.HotDays, seasonSummary.HighUVDays, seasonSummary.RainTotal)
	default:
		ctx.Description = "Осенью важны влажность, пасмурность, ветер и редкие сухие окна. Сравнение с летними месяцами может искажать выводы, поэтому смотрим на осенний контекст."
		ctx.FocusTitle = "Осенний профиль: влажность и ветер"
		ctx.FocusText = fmt.Sprintf("За сезон уже %d пасмурных дней, %d ветреных и %.1f мм осадков.", seasonSummary.CloudyDays, seasonSummary.WindyDays, seasonSummary.RainTotal)
	}

	return ctx
}

func buildSameMonthBenchmark(now time.Time, current models.MonthlyWeatherInsights, archiveDays []models.DailyWeatherInsight, loc *time.Location) models.WeatherArchiveBenchmark {
	benchmark := models.WeatherArchiveBenchmark{
		Title:      "Такие же месяцы в архиве",
		Subtitle:   fmt.Sprintf("%s к %d числу прошлых лет", monthName(now.Month()), now.Day()),
		StatusText: "Пока недостаточно прошлых лет с данными по этому месяцу. Сравнение с прошлым месяцем остаётся справочным, а не главным выводом.",
	}

	byYear := make(map[int][]models.DailyWeatherInsight)
	for _, day := range archiveDays {
		local := day.Date.In(loc)
		if local.Month() != now.Month() || local.Day() > now.Day() || local.Year() == now.Year() {
			continue
		}
		byYear[local.Year()] = append(byYear[local.Year()], day)
	}

	minDays := maxInt(3, int(math.Ceil(float64(now.Day())*0.65)))
	samples := make([]models.MonthlyWeatherInsights, 0, len(byYear))
	for year, days := range byYear {
		if len(days) < minDays {
			continue
		}
		start := time.Date(year, now.Month(), 1, 0, 0, 0, 0, loc)
		end := time.Date(year, now.Month(), minInt(now.Day(), daysInMonth(year, now.Month(), loc)), 23, 59, 59, 0, loc)
		samples = append(samples, buildMonthlyInsights(fmt.Sprintf("%d", year), start, end, days, now.Day()))
	}

	benchmark.SampleSize = len(samples)
	if len(samples) == 0 {
		return benchmark
	}

	for _, sample := range samples {
		benchmark.RainTotalAvg += sample.RainTotal
		benchmark.RainDaysAvg += float64(sample.RainDays)
		benchmark.AvgTempAvg += sample.AvgTemp
	}
	benchmark.Available = true
	benchmark.Reliable = len(samples) >= 2
	benchmark.RainTotalAvg /= float64(len(samples))
	benchmark.RainDaysAvg /= float64(len(samples))
	benchmark.AvgTempAvg /= float64(len(samples))
	benchmark.RainRatioPercent = clampPercent(ratioPercent(current.RainTotal, benchmark.RainTotalAvg))
	benchmark.RainDeltaPercent = int(math.Round((current.RainTotal - benchmark.RainTotalAvg) / math.Max(benchmark.RainTotalAvg, 0.1) * 100))
	benchmark.TempDelta = math.Round((current.AvgTemp-benchmark.AvgTempAvg)*10) / 10
	benchmark.StatusText = fmt.Sprintf("Есть %d архивных сравнений для этого месяца года.", len(samples))
	if !benchmark.Reliable {
		benchmark.StatusText = "Есть только один похожий месяц в архиве — выводы показываем осторожно."
	}

	switch {
	case benchmark.RainDeltaPercent >= 40:
		benchmark.Verdict = "заметно влажнее обычного для этого месяца"
	case benchmark.RainDeltaPercent >= 15:
		benchmark.Verdict = "немного влажнее обычного для этого месяца"
	case benchmark.RainDeltaPercent <= -40:
		benchmark.Verdict = "заметно суше обычного для этого месяца"
	case benchmark.RainDeltaPercent <= -15:
		benchmark.Verdict = "немного суше обычного для этого месяца"
	default:
		benchmark.Verdict = "примерно в сезонном коридоре"
	}

	return benchmark
}

func buildSameSeasonBenchmark(current models.MonthlyWeatherInsights, archiveDays []models.DailyWeatherInsight, seasonYear int, seasonCode string, compareDays int, loc *time.Location) models.WeatherArchiveBenchmark {
	benchmark := models.WeatherArchiveBenchmark{
		Title:      "Такие же сезоны в архиве",
		Subtitle:   fmt.Sprintf("%s к %d дню сезона прошлых лет", seasonNameByCode(seasonCode), compareDays),
		StatusText: "Пока недостаточно прошлых лет с данными по этому сезону. Сравнение с прошлым сезоном остаётся справочным, а не главным выводом.",
	}

	bySeason := make(map[int][]models.DailyWeatherInsight)
	for _, day := range archiveDays {
		local := day.Date.In(loc)
		year, code := seasonIDForDate(local)
		if code != seasonCode || year == seasonYear {
			continue
		}
		start, _ := seasonBoundsByID(year, code, loc)
		if daysBetween(start, dayStart(local, loc)) >= compareDays {
			continue
		}
		bySeason[year] = append(bySeason[year], day)
	}

	minDays := maxInt(14, int(math.Ceil(float64(compareDays)*0.65)))
	samples := make([]models.MonthlyWeatherInsights, 0, len(bySeason))
	for year, days := range bySeason {
		if len(days) < minDays {
			continue
		}
		start, _ := seasonBoundsByID(year, seasonCode, loc)
		end := start.AddDate(0, 0, compareDays).Add(-time.Nanosecond)
		samples = append(samples, buildMonthlyInsights(fmt.Sprintf("%d", year), start, end, days, compareDays))
	}

	benchmark.SampleSize = len(samples)
	if len(samples) == 0 {
		return benchmark
	}

	for _, sample := range samples {
		benchmark.RainTotalAvg += sample.RainTotal
		benchmark.RainDaysAvg += float64(sample.RainDays)
		benchmark.AvgTempAvg += sample.AvgTemp
	}
	benchmark.Available = true
	benchmark.Reliable = len(samples) >= 2
	benchmark.RainTotalAvg /= float64(len(samples))
	benchmark.RainDaysAvg /= float64(len(samples))
	benchmark.AvgTempAvg /= float64(len(samples))
	benchmark.RainRatioPercent = clampPercent(ratioPercent(current.RainTotal, benchmark.RainTotalAvg))
	benchmark.RainDeltaPercent = int(math.Round((current.RainTotal - benchmark.RainTotalAvg) / math.Max(benchmark.RainTotalAvg, 0.1) * 100))
	benchmark.TempDelta = math.Round((current.AvgTemp-benchmark.AvgTempAvg)*10) / 10
	benchmark.StatusText = fmt.Sprintf("Есть %d архивных сравнений для этого сезона.", len(samples))
	if !benchmark.Reliable {
		benchmark.StatusText = "Есть только один похожий сезон в архиве — выводы показываем осторожно."
	}

	suffix := "для этого сезона"
	switch {
	case benchmark.RainDeltaPercent >= 40:
		benchmark.Verdict = "заметно влажнее обычного " + suffix
	case benchmark.RainDeltaPercent >= 15:
		benchmark.Verdict = "немного влажнее обычного " + suffix
	case benchmark.RainDeltaPercent <= -40:
		benchmark.Verdict = "заметно суше обычного " + suffix
	case benchmark.RainDeltaPercent <= -15:
		benchmark.Verdict = "немного суше обычного " + suffix
	default:
		benchmark.Verdict = "примерно в сезонном коридоре"
	}

	return benchmark
}

func buildSeasonMonthCards(seasonStart, periodEnd time.Time, days []models.DailyWeatherInsight, loc *time.Location) []models.MonthlyWeatherInsights {
	cards := make([]models.MonthlyWeatherInsights, 0, 3)
	for i := 0; i < 3; i++ {
		monthStart := seasonStart.AddDate(0, i, 0)
		monthEnd := monthStart.AddDate(0, 1, 0)
		queryEnd := minTime(monthEnd, periodEnd)
		monthDays := []models.DailyWeatherInsight{}
		daysInPeriod := daysBetween(monthStart, monthEnd)
		endDate := monthEnd.Add(-time.Nanosecond)
		if queryEnd.After(monthStart) {
			monthDays = filterDaysBetween(days, monthStart, queryEnd, loc)
			endDate = queryEnd.Add(-time.Nanosecond)
			if queryEnd.Before(monthEnd) {
				daysInPeriod = maxInt(1, daysBetween(monthStart, dayStart(queryEnd, loc).AddDate(0, 0, 1)))
			}
		}
		cards = append(cards, buildMonthlyInsights(russianMonthYear(monthStart), monthStart, endDate, monthDays, daysInPeriod))
	}
	return cards
}

func buildRollingPeriod(title, subtitle string, days []models.DailyWeatherInsight, now time.Time, length int, loc *time.Location) models.RollingWeatherPeriod {
	end := dayStart(now, loc).AddDate(0, 0, 1)
	currentStart := end.AddDate(0, 0, -length)
	previousStart := currentStart.AddDate(0, 0, -length)
	currentDays := filterDaysBetween(days, currentStart, end, loc)
	previousDays := filterDaysBetween(days, previousStart, currentStart, loc)

	current := buildMonthlyInsights(title, currentStart, end.Add(-time.Nanosecond), currentDays, length)
	previous := buildMonthlyInsights("Предыдущий период", previousStart, currentStart.Add(-time.Nanosecond), previousDays, length)
	result := models.RollingWeatherPeriod{
		Title:            title,
		Subtitle:         subtitle,
		Current:          current,
		Previous:         previous,
		RainDeltaPercent: int(math.Round((current.RainTotal - previous.RainTotal) / math.Max(previous.RainTotal, 0.1) * 100)),
		TempDelta:        math.Round((current.AvgTemp-previous.AvgTemp)*10) / 10,
	}

	switch {
	case current.RainTotal >= previous.RainTotal*1.5 && current.RainTotal >= 2:
		result.Verdict = fmt.Sprintf("Сейчас влажнее: %.1f мм против %.1f мм в предыдущем таком же периоде.", current.RainTotal, previous.RainTotal)
	case previous.RainTotal >= current.RainTotal*1.5 && previous.RainTotal >= 2:
		result.Verdict = fmt.Sprintf("Сейчас суше: %.1f мм против %.1f мм в предыдущем таком же периоде.", current.RainTotal, previous.RainTotal)
	case math.Abs(result.TempDelta) >= 2:
		result.Verdict = fmt.Sprintf("По осадкам период близок, но средняя температура изменилась на %.1f°C.", result.TempDelta)
	default:
		result.Verdict = "Ближайший период без резкого перелома: осадки и температура близки к предыдущему отрезку."
	}

	return result
}

func buildRainChartData(currentDays, previousDays, archiveDays []models.DailyWeatherInsight, currentMonthDays, previousMonthDays, currentDay int, currentMonth time.Month, loc *time.Location) map[string]interface{} {
	maxDays := maxInt(currentMonthDays, previousMonthDays)
	labels := make([]string, maxDays)
	current := make([]interface{}, maxDays)
	previous := make([]interface{}, maxDays)
	archive := make([]interface{}, maxDays)
	currentByDay := make(map[int]float64, len(currentDays))
	previousByDay := make(map[int]float64, len(previousDays))

	for _, day := range currentDays {
		currentByDay[day.Date.In(loc).Day()] = value32(day.RainTotal)
	}
	for _, day := range previousDays {
		previousByDay[day.Date.In(loc).Day()] = value32(day.RainTotal)
	}
	archiveByDay := averageArchiveRainCumulativeByDay(archiveDays, currentMonth, maxDays, loc)

	var currentSum, previousSum float64
	for i := 1; i <= maxDays; i++ {
		labels[i-1] = fmt.Sprintf("%d", i)
		if i <= currentDay {
			currentSum += currentByDay[i]
			current[i-1] = math.Round(currentSum*10) / 10
		} else {
			current[i-1] = nil
		}
		if i <= previousMonthDays {
			previousSum += previousByDay[i]
			previous[i-1] = math.Round(previousSum*10) / 10
		} else {
			previous[i-1] = nil
		}
		if value, ok := archiveByDay[i]; ok {
			archive[i-1] = math.Round(value*10) / 10
		} else {
			archive[i-1] = nil
		}
	}

	return map[string]interface{}{
		"labels":   labels,
		"current":  current,
		"previous": previous,
		"archive":  archive,
	}
}

func buildDayTypeSummaries(days []models.DailyWeatherInsight) ([]models.WeatherDayTypeSummary, models.WeatherDayTypeSummary) {
	defs := map[string]models.WeatherDayTypeSummary{
		"storm":       {Code: "storm", Label: "ливневые", Icon: "🌧️", Description: "дни с сильным дождём", Class: "bg-blue-50 text-blue-800 dark:bg-blue-900/20 dark:text-blue-200"},
		"windy":       {Code: "windy", Label: "ветреные", Icon: "💨", Description: "порывы заметно мешали", Class: "bg-teal-50 text-teal-800 dark:bg-teal-900/20 dark:text-teal-200"},
		"hot":         {Code: "hot", Label: "жаркие", Icon: "🔥", Description: "максимум ≥30°C", Class: "bg-red-50 text-red-800 dark:bg-red-900/20 dark:text-red-200"},
		"comfortable": {Code: "comfortable", Label: "комфортные", Icon: "🚶", Description: "сухо, умеренно тепло и без сильного ветра", Class: "bg-emerald-50 text-emerald-800 dark:bg-emerald-900/20 dark:text-emerald-200"},
		"sunny":       {Code: "sunny", Label: "солнечные", Icon: "☀️", Description: "яркие сухие дни", Class: "bg-yellow-50 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-200"},
		"wet":         {Code: "wet", Label: "влажные", Icon: "💧", Description: "осадки без ливневого пика", Class: "bg-cyan-50 text-cyan-800 dark:bg-cyan-900/20 dark:text-cyan-200"},
		"cloudy":      {Code: "cloudy", Label: "пасмурные", Icon: "☁️", Description: "мало солнечной радиации", Class: "bg-slate-50 text-slate-800 dark:bg-slate-700/40 dark:text-slate-200"},
		"calm":        {Code: "calm", Label: "спокойные", Icon: "▫️", Description: "без яркого погодного акцента", Class: "bg-gray-50 text-gray-800 dark:bg-gray-700/40 dark:text-gray-200"},
	}
	order := []string{"storm", "windy", "hot", "comfortable", "sunny", "wet", "cloudy", "calm"}
	counts := make(map[string]int, len(order))
	for _, day := range days {
		counts[classifyDayType(day)]++
	}

	result := make([]models.WeatherDayTypeSummary, 0, len(order))
	dominant := defs["calm"]
	for _, code := range order {
		item := defs[code]
		item.Count = counts[code]
		item.Percent = clampPercent(ratioPercent(float64(item.Count), float64(maxInt(len(days), 1))))
		if item.Count > 0 {
			result = append(result, item)
		}
		if item.Count > dominant.Count {
			dominant = item
		}
	}
	return result, dominant
}

func classifyDayType(day models.DailyWeatherInsight) string {
	rain := value32(day.RainTotal)
	gust := value32(day.WindGustMax)
	tempMax := value32(day.TempMax)
	tempAvg := value32(day.TempAvg)
	solar := value32(day.SolarRadiationMax)

	switch {
	case rain >= heavyRainThreshold:
		return "storm"
	case gust >= strongGustThreshold:
		return "windy"
	case tempMax >= hotDayTempThreshold:
		return "hot"
	case tempAvg >= 18 && tempAvg <= 26 && rain < wetDayRainThreshold && gust < 8:
		return "comfortable"
	case solar >= sunnySolarThreshold && rain < wetDayRainThreshold:
		return "sunny"
	case rain >= wetDayRainThreshold:
		return "wet"
	case day.SolarRadiationMax != nil && solar < cloudySolarThreshold:
		return "cloudy"
	default:
		return "calm"
	}
}

func buildWindInsight(current models.MonthlyWeatherInsights) models.WeatherFactorInsight {
	insight := models.WeatherFactorInsight{
		Icon:       "💨",
		Title:      "Ветер",
		Value:      fmt.Sprintf("%d", current.WindyDays),
		Detail:     "ветреных дней",
		Advice:     "Ветер почти не вмешивался в сценарий месяца.",
		Level:      "low",
		LevelLabel: "спокойно",
	}
	if current.MaxWindGustDay != nil {
		insight.Detail = fmt.Sprintf("максимальный порыв %.1f м/с — %s", current.MaxWindGustDay.Value, formatInsightDate(current.MaxWindGustDay.Date))
	}
	if current.StrongWindDays > 0 {
		insight.Value = fmt.Sprintf("%d", current.StrongWindDays)
		insight.Advice = "Были дни, когда порывы уже могли мешать прогулкам, велосипеду и лёгким конструкциям."
		insight.Level = "high"
		insight.LevelLabel = "порывисто"
	} else if current.WindyDays > 0 {
		insight.Advice = "Ветер был заметен, но без частых сильных порывов."
		insight.Level = "medium"
		insight.LevelLabel = "заметно"
	}
	return insight
}

func buildUVInsight(days []models.DailyWeatherInsight, current models.MonthlyWeatherInsights) models.WeatherFactorInsight {
	maxUV := 0.0
	var maxUVDate time.Time
	for _, day := range days {
		uv := value32(day.UVIndexMax)
		if day.UVIndexMax != nil && uv > maxUV {
			maxUV = uv
			maxUVDate = day.Date
		}
	}

	insight := models.WeatherFactorInsight{
		Icon:       "🕶️",
		Title:      "UV и солнце",
		Value:      fmt.Sprintf("%d", current.HighUVDays),
		Detail:     "дней с высоким UV",
		Advice:     "UV пока не был главным фактором месяца.",
		Level:      "low",
		LevelLabel: "мягко",
	}
	if !maxUVDate.IsZero() {
		insight.Detail = fmt.Sprintf("максимум UV %.1f — %s", maxUV, formatInsightDate(maxUVDate))
	}
	if current.HighUVDays >= 5 {
		insight.Advice = "Солнцезащита была практичной необходимостью: очки, крем и тень в середине дня."
		insight.Level = "high"
		insight.LevelLabel = "активно"
	} else if current.HighUVDays > 0 || current.SunnyDays >= 5 {
		insight.Advice = "Были яркие дни: для долгих прогулок лучше учитывать солнце, даже если температура комфортная."
		insight.Level = "medium"
		insight.LevelLabel = "ярко"
	}
	return insight
}

func buildTimelineEvents(current models.MonthlyWeatherInsights, days []models.DailyWeatherInsight, bestDay, worstDay *models.NotableWeatherDay) []models.WeatherTimelineEvent {
	events := make([]models.WeatherTimelineEvent, 0, 8)
	seen := make(map[string]bool)
	add := func(date time.Time, icon, title, description, category string, severity int) {
		if date.IsZero() {
			return
		}
		key := fmt.Sprintf("%s:%s", date.Format("2006-01-02"), category)
		if seen[key] {
			return
		}
		seen[key] = true
		events = append(events, models.WeatherTimelineEvent{Date: date, Icon: icon, Title: title, Description: description, Category: category, Severity: severity})
	}

	if current.MaxRainDay != nil {
		add(current.MaxRainDay.Date, "🌧️", "Главный дождь", fmt.Sprintf("%.1f мм за день", current.MaxRainDay.Value), "rain", 90)
	}
	if current.MaxWindGustDay != nil {
		add(current.MaxWindGustDay.Date, "💨", "Самый сильный порыв", fmt.Sprintf("%.1f м/с", current.MaxWindGustDay.Value), "wind", 75)
	}
	if current.MaxTempDay != nil {
		add(current.MaxTempDay.Date, "🔥", "Самый жаркий день", fmt.Sprintf("%.1f°C", current.MaxTempDay.Value), "heat", 70)
	}
	if current.MinTempDay != nil {
		add(current.MinTempDay.Date, "🧊", "Самая холодная ночь", fmt.Sprintf("%.1f°C", current.MinTempDay.Value), "cold", 65)
	}
	if current.SunniestDay != nil {
		add(current.SunniestDay.Date, "☀️", "Самый солнечный день", fmt.Sprintf("%.0f Вт/м²", current.SunniestDay.Value), "sun", 55)
	}
	if maxUVDate, maxUV := maxUVDay(days); !maxUVDate.IsZero() {
		add(maxUVDate, "🕶️", "Пик UV", fmt.Sprintf("UV %.1f", maxUV), "uv", 60)
	}
	if bestDay != nil {
		add(bestDay.Date, bestDay.Icon, bestDay.Title, bestDay.Description, "best", bestDay.Score)
	}
	if worstDay != nil {
		add(worstDay.Date, worstDay.Icon, worstDay.Title, worstDay.Description, "worst", 100-worstDay.Score)
	}

	sortTimelineEvents(events)
	if len(events) > 7 {
		return events[:7]
	}
	return events
}

func maxUVDay(days []models.DailyWeatherInsight) (time.Time, float64) {
	maxUV := 0.0
	var date time.Time
	for _, day := range days {
		uv := value32(day.UVIndexMax)
		if day.UVIndexMax != nil && uv > maxUV {
			maxUV = uv
			date = day.Date
		}
	}
	return date, maxUV
}

func sortTimelineEvents(events []models.WeatherTimelineEvent) {
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].Date.Before(events[i].Date) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}
}

func buildInsightStories(current, previous, previousSame models.MonthlyWeatherInsights, benchmark models.WeatherArchiveBenchmark, last7 models.RollingWeatherPeriod, season models.WeatherSeasonContext, dryStreak int, lastRain time.Time, hasLastRain bool, periodGenitive string) (models.WeatherInsightStory, []models.WeatherInsightStory) {
	stories := make([]models.WeatherInsightStory, 0, 7)

	main := models.WeatherInsightStory{
		Icon:  season.Icon,
		Title: season.FocusTitle,
		Text:  fmt.Sprintf("%s За %d дней %s собрано %.1f мм осадков и %d дождливых дней.", season.FocusText, current.DaysWithData, periodGenitive, current.RainTotal, current.RainDays),
	}

	periodNoun := trimGenitivePeriod(periodGenitive)
	if benchmark.Available && math.Abs(float64(benchmark.RainDeltaPercent)) >= 25 {
		icon := "💧"
		title := fmt.Sprintf("%s влажнее сезонной нормы", capitalize(periodNoun))
		if benchmark.RainDeltaPercent < 0 {
			icon = "🌤️"
			title = fmt.Sprintf("%s суше сезонной нормы", capitalize(periodNoun))
		}
		main = models.WeatherInsightStory{
			Icon:  icon,
			Title: title,
			Text:  fmt.Sprintf("Для этого периода года норма к текущей точке — около %.1f мм. Сейчас %.1f мм: %s", benchmark.RainTotalAvg, current.RainTotal, benchmark.Verdict),
		}
	} else if last7.Current.RainTotal >= last7.Previous.RainTotal*1.8 && last7.Current.RainTotal >= 5 {
		main = models.WeatherInsightStory{
			Icon:  "🌧️",
			Title: "Последняя неделя резко влажнее",
			Text:  fmt.Sprintf("За последние 7 дней выпало %.1f мм против %.1f мм неделей ранее. Это лучше отражает текущую погоду, чем сравнение с прошлым %s.", last7.Current.RainTotal, last7.Previous.RainTotal, periodNoun),
		}
	} else if current.RainTotal > previousSame.RainTotal {
		main = models.WeatherInsightStory{
			Icon:  "📈",
			Title: fmt.Sprintf("Темп осадков выше прошлого %s", periodGenitive),
			Text:  fmt.Sprintf("Это справочное сравнение: к той же точке прошлого %s было %.1f мм, сейчас %.1f мм. Сезонный контекст важнее.", periodGenitive, previousSame.RainTotal, current.RainTotal),
		}
	}

	stories = append(stories, models.WeatherInsightStory{
		Icon:  "🔎",
		Title: "Ближайший погодный срез",
		Text:  last7.Verdict,
	})

	if dryStreak > 0 {
		text := fmt.Sprintf("Сейчас %d сухих дней подряд по последним данным.", dryStreak)
		if hasLastRain {
			text = fmt.Sprintf("Последний дождь был %s; сейчас %d сухих дней подряд.", formatInsightDate(lastRain), dryStreak)
		}
		stories = append(stories, models.WeatherInsightStory{Icon: "🌤️", Title: "Сухая серия", Text: text})
	} else if hasLastRain {
		stories = append(stories, models.WeatherInsightStory{Icon: "🌦️", Title: "Сухой серии нет", Text: fmt.Sprintf("Дождь был сегодня (%s), поэтому счётчик сухих дней пока на нуле.", formatInsightDate(lastRain))})
	}

	if current.MaxRainDay != nil {
		share := 0.0
		if current.RainTotal > 0 {
			share = current.MaxRainDay.Value / current.RainTotal * 100
		}
		stories = append(stories, models.WeatherInsightStory{
			Icon:  "🏆",
			Title: fmt.Sprintf("Самый дождливый день %s", periodGenitive),
			Text:  fmt.Sprintf("%s — %.1f мм осадков, около %.0f%% суммы %s.", formatInsightDate(current.MaxRainDay.Date), current.MaxRainDay.Value, share, periodGenitive),
		})
	}

	if current.ComfortableDays > 0 {
		stories = append(stories, models.WeatherInsightStory{
			Icon:  "🚶",
			Title: "Дни для прогулок",
			Text:  fmt.Sprintf("Комфортных дней за период: %d из %d.", current.ComfortableDays, current.DaysWithData),
		})
	}

	return main, stories
}

func findNotableDays(days []models.DailyWeatherInsight) (*models.NotableWeatherDay, *models.NotableWeatherDay) {
	var best *models.NotableWeatherDay
	var worst *models.NotableWeatherDay
	for _, day := range days {
		score := comfortScore(day)
		desc := notableDescription(day, score)
		candidate := &models.NotableWeatherDay{
			Icon:        "🚶",
			Title:       "Лучший день для прогулки",
			Date:        day.Date,
			Description: desc,
			Score:       score,
		}
		badCandidate := &models.NotableWeatherDay{
			Icon:        "🌪️",
			Title:       "Самый тяжёлый день",
			Date:        day.Date,
			Description: desc,
			Score:       score,
		}
		if best == nil || score > best.Score {
			best = candidate
		}
		if worst == nil || score < worst.Score {
			worst = badCandidate
		}
	}
	return best, worst
}

func comfortScore(day models.DailyWeatherInsight) int {
	score := 100
	tempAvg := value32(day.TempAvg)
	rain := value32(day.RainTotal)
	gust := value32(day.WindGustMax)
	uv := value32(day.UVIndexMax)

	if day.TempAvg != nil {
		score -= int(math.Abs(tempAvg-22) * 2)
	}
	score -= int(math.Min(rain*1.5, 30))
	if gust > 6 {
		score -= int((gust - 6) * 2)
	}
	if uv > 7 {
		score -= int((uv - 7) * 3)
	}
	return maxInt(0, minInt(100, score))
}

func notableDescription(day models.DailyWeatherInsight, score int) string {
	return fmt.Sprintf("Индекс %d/100: средняя %.1f°C, осадки %.1f мм, порыв %.1f м/с.", score, value32(day.TempAvg), value32(day.RainTotal), value32(day.WindGustMax))
}

func calculateRainRecency(days []models.DailyWeatherInsight, loc *time.Location) (dryStreak int, lastRain time.Time, hasLastRain bool) {
	for i := len(days) - 1; i >= 0; i-- {
		rain := value32(days[i].RainTotal)
		if rain >= wetDayRainThreshold {
			return dryStreak, days[i].Date.In(loc), true
		}
		dryStreak++
	}
	return dryStreak, time.Time{}, false
}

func dayBadges(day models.DailyWeatherInsight) []string {
	badges := make([]string, 0, 3)
	rain := value32(day.RainTotal)
	if rain >= heavyRainThreshold {
		badges = append(badges, "🌧️")
	} else if rain >= wetDayRainThreshold {
		badges = append(badges, "💧")
	}
	if value32(day.WindGustMax) >= windyGustThreshold {
		badges = append(badges, "💨")
	}
	if value32(day.TempMax) >= hotDayTempThreshold {
		badges = append(badges, "🔥")
	}
	if value32(day.SolarRadiationMax) >= sunnySolarThreshold {
		badges = append(badges, "☀️")
	}
	return badges
}

func calendarClass(day models.DailyWeatherInsight) string {
	rain := value32(day.RainTotal)
	tempMax := value32(day.TempMax)
	solar := value32(day.SolarRadiationMax)
	switch {
	case rain >= heavyRainThreshold:
		return "bg-blue-100 dark:bg-blue-900/40 border-blue-300 dark:border-blue-700"
	case rain >= wetDayRainThreshold:
		return "bg-cyan-50 dark:bg-cyan-900/30 border-cyan-200 dark:border-cyan-800"
	case tempMax >= hotDayTempThreshold:
		return "bg-red-50 dark:bg-red-900/25 border-red-200 dark:border-red-800"
	case solar >= sunnySolarThreshold:
		return "bg-yellow-50 dark:bg-yellow-900/25 border-yellow-200 dark:border-yellow-800"
	default:
		return "bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700"
	}
}

func rainLevel(rain float64) string {
	switch {
	case rain >= heavyRainThreshold:
		return "heavy"
	case rain >= rainyDayRainThreshold:
		return "rain"
	case rain >= wetDayRainThreshold:
		return "wet"
	default:
		return "none"
	}
}

func filterDaysBefore(days []models.DailyWeatherInsight, end time.Time) []models.DailyWeatherInsight {
	result := make([]models.DailyWeatherInsight, 0, len(days))
	for _, day := range days {
		if day.Date.Before(end) {
			result = append(result, day)
		}
	}
	return result
}

func filterDaysBetween(days []models.DailyWeatherInsight, start, end time.Time, loc *time.Location) []models.DailyWeatherInsight {
	result := make([]models.DailyWeatherInsight, 0, len(days))
	for _, day := range days {
		date := dayStart(day.Date, loc)
		if !date.Before(start) && date.Before(end) {
			result = append(result, day)
		}
	}
	return result
}

func buildSeasonRainChartData(currentDays, previousDays, archiveDays []models.DailyWeatherInsight, currentStart, currentEnd, currentPeriodEnd, previousStart, previousEnd time.Time, seasonCode string, loc *time.Location) map[string]interface{} {
	maxDays := maxInt(daysBetween(currentStart, currentEnd), daysBetween(previousStart, previousEnd))
	currentLimit := daysBetween(currentStart, minTime(dayStart(currentPeriodEnd, loc).AddDate(0, 0, 1), currentEnd))
	if currentPeriodEnd.Equal(currentEnd) || currentPeriodEnd.After(currentEnd) {
		currentLimit = daysBetween(currentStart, currentEnd)
	}
	labels := make([]string, maxDays)
	current := make([]interface{}, maxDays)
	previous := make([]interface{}, maxDays)
	archive := make([]interface{}, maxDays)

	currentByOffset := rainByOffset(currentDays, currentStart, loc)
	previousByOffset := rainByOffset(previousDays, previousStart, loc)
	archiveByOffset := averageArchiveSeasonRainCumulativeByOffset(archiveDays, seasonCode, maxDays, loc)

	var currentSum, previousSum float64
	for i := 0; i < maxDays; i++ {
		labels[i] = currentStart.AddDate(0, 0, i).Format("02.01")
		if i < currentLimit {
			currentSum += currentByOffset[i]
			current[i] = math.Round(currentSum*10) / 10
		} else {
			current[i] = nil
		}
		if i < daysBetween(previousStart, previousEnd) {
			previousSum += previousByOffset[i]
			previous[i] = math.Round(previousSum*10) / 10
		} else {
			previous[i] = nil
		}
		if value, ok := archiveByOffset[i]; ok {
			archive[i] = math.Round(value*10) / 10
		} else {
			archive[i] = nil
		}
	}

	return map[string]interface{}{
		"labels":   labels,
		"current":  current,
		"previous": previous,
		"archive":  archive,
	}
}

func rainByOffset(days []models.DailyWeatherInsight, start time.Time, loc *time.Location) map[int]float64 {
	result := make(map[int]float64, len(days))
	start = dayStart(start, loc)
	for _, day := range days {
		offset := daysBetween(start, dayStart(day.Date, loc))
		if offset >= 0 {
			result[offset] = value32(day.RainTotal)
		}
	}
	return result
}

func averageArchiveSeasonRainCumulativeByOffset(days []models.DailyWeatherInsight, seasonCode string, maxDays int, loc *time.Location) map[int]float64 {
	bySeason := make(map[int]map[int]float64)
	for _, day := range days {
		local := day.Date.In(loc)
		seasonYear, code := seasonIDForDate(local)
		if code != seasonCode {
			continue
		}
		start, _ := seasonBoundsByID(seasonYear, seasonCode, loc)
		offset := daysBetween(start, dayStart(local, loc))
		if offset < 0 || offset >= maxDays {
			continue
		}
		if bySeason[seasonYear] == nil {
			bySeason[seasonYear] = make(map[int]float64)
		}
		bySeason[seasonYear][offset] = value32(day.RainTotal)
	}

	result := make(map[int]float64, maxDays)
	if len(bySeason) < 2 {
		return result
	}

	sums := make([]float64, maxDays)
	counts := make([]int, maxDays)
	for _, byOffset := range bySeason {
		if len(byOffset) < 10 {
			continue
		}
		var cumulative float64
		for offset := 0; offset < maxDays; offset++ {
			cumulative += byOffset[offset]
			if _, ok := byOffset[offset]; ok {
				sums[offset] += cumulative
				counts[offset]++
			}
		}
	}

	for offset := 0; offset < maxDays; offset++ {
		if counts[offset] > 0 {
			result[offset] = sums[offset] / float64(counts[offset])
		}
	}
	return result
}

func averageArchiveRainCumulativeByDay(days []models.DailyWeatherInsight, month time.Month, maxDays int, loc *time.Location) map[int]float64 {
	byYear := make(map[int]map[int]float64)
	for _, day := range days {
		local := day.Date.In(loc)
		if local.Month() != month {
			continue
		}
		if byYear[local.Year()] == nil {
			byYear[local.Year()] = make(map[int]float64)
		}
		byYear[local.Year()][local.Day()] = value32(day.RainTotal)
	}

	result := make(map[int]float64, maxDays)
	if len(byYear) < 2 {
		return result
	}

	sums := make([]float64, maxDays+1)
	counts := make([]int, maxDays+1)
	for _, byDay := range byYear {
		if len(byDay) < 3 {
			continue
		}
		var cumulative float64
		for day := 1; day <= maxDays; day++ {
			cumulative += byDay[day]
			if _, ok := byDay[day]; ok {
				sums[day] += cumulative
				counts[day]++
			}
		}
	}

	for day := 1; day <= maxDays; day++ {
		if counts[day] > 0 {
			result[day] = sums[day] / float64(counts[day])
		}
	}
	return result
}

func parseSeasonParam(value string) (int, string, error) {
	if len(value) < 6 {
		return 0, "", ErrInvalidInsightSeason
	}
	var year int
	var code string
	if _, err := fmt.Sscanf(value, "%d-%s", &year, &code); err != nil {
		return 0, "", ErrInvalidInsightSeason
	}
	if !validSeasonCode(code) || year < 2000 || year > 2100 {
		return 0, "", ErrInvalidInsightSeason
	}
	return year, code, nil
}

func validSeasonCode(code string) bool {
	switch code {
	case "winter", "spring", "summer", "autumn":
		return true
	default:
		return false
	}
}

func seasonBoundsByID(year int, code string, loc *time.Location) (time.Time, time.Time) {
	var start time.Time
	switch code {
	case "winter":
		start = time.Date(year-1, time.December, 1, 0, 0, 0, 0, loc)
	case "spring":
		start = time.Date(year, time.March, 1, 0, 0, 0, 0, loc)
	case "summer":
		start = time.Date(year, time.June, 1, 0, 0, 0, 0, loc)
	default:
		start = time.Date(year, time.September, 1, 0, 0, 0, 0, loc)
	}
	return start, start.AddDate(0, 3, 0)
}

func seasonIDFromStart(start time.Time) (int, string) {
	_, code, _ := seasonName(start.Month())
	if code == "winter" {
		return start.Year() + 1, code
	}
	return start.Year(), code
}

func seasonIDForDate(t time.Time) (int, string) {
	_, code, _ := seasonName(t.Month())
	if code == "winter" && t.Month() == time.December {
		return t.Year() + 1, code
	}
	return t.Year(), code
}

func shiftSeasonID(year int, code string, delta int) (int, string) {
	order := []string{"winter", "spring", "summer", "autumn"}
	index := 0
	for i, item := range order {
		if item == code {
			index = i
			break
		}
	}
	absolute := year*4 + index + delta
	newYear := absolute / 4
	newIndex := absolute % 4
	if newIndex < 0 {
		newIndex += 4
		newYear--
	}
	return newYear, order[newIndex]
}

func formatSeasonParam(year int, code string) string {
	return fmt.Sprintf("%04d-%s", year, code)
}

func seasonLabel(year int, code string) string {
	name := seasonNameByCode(code)
	if code == "winter" {
		return fmt.Sprintf("%s %d/%02d", capitalize(name), year-1, year%100)
	}
	return fmt.Sprintf("%s %d", capitalize(name), year)
}

func seasonNameByCode(code string) string {
	switch code {
	case "winter":
		return "зима"
	case "spring":
		return "весна"
	case "summer":
		return "лето"
	default:
		return "осень"
	}
}

func capitalize(value string) string {
	switch value {
	case "месяц":
		return "Месяц"
	case "сезон":
		return "Сезон"
	case "зима":
		return "Зима"
	case "весна":
		return "Весна"
	case "лето":
		return "Лето"
	case "осень":
		return "Осень"
	default:
		return value
	}
}

func ensureSeasonOption(options []models.WeatherInsightsPeriodOption, selectedYear int, selectedCode string) []models.WeatherInsightsPeriodOption {
	selectedValue := formatSeasonParam(selectedYear, selectedCode)
	for _, option := range options {
		if option.Value == selectedValue {
			return options
		}
	}
	return append([]models.WeatherInsightsPeriodOption{{Value: selectedValue, Label: seasonLabel(selectedYear, selectedCode)}}, options...)
}

func buildSeasonOptions(currentYear int, currentCode string) []models.WeatherInsightsPeriodOption {
	options := make([]models.WeatherInsightsPeriodOption, 0, 13)
	year, code := currentYear, currentCode
	for i := 0; i < 13; i++ {
		options = append(options, models.WeatherInsightsPeriodOption{
			Value: formatSeasonParam(year, code),
			Label: seasonLabel(year, code),
		})
		year, code = shiftSeasonID(year, code, -1)
	}
	return options
}

func seasonBounds(t time.Time, loc *time.Location) (time.Time, time.Time) {
	year := t.In(loc).Year()
	switch t.In(loc).Month() {
	case time.December:
		start := time.Date(year, time.December, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 3, 0)
	case time.January, time.February:
		start := time.Date(year-1, time.December, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 3, 0)
	case time.March, time.April, time.May:
		start := time.Date(year, time.March, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 3, 0)
	case time.June, time.July, time.August:
		start := time.Date(year, time.June, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 3, 0)
	default:
		start := time.Date(year, time.September, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 3, 0)
	}
}

func seasonName(month time.Month) (name, code, icon string) {
	switch month {
	case time.December, time.January, time.February:
		return "зима", "winter", "❄️"
	case time.March, time.April, time.May:
		return "весна", "spring", "🌱"
	case time.June, time.July, time.August:
		return "лето", "summer", "☀️"
	default:
		return "осень", "autumn", "🍂"
	}
}

func monthName(month time.Month) string {
	months := []string{"", "январь", "февраль", "март", "апрель", "май", "июнь", "июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь"}
	return months[month]
}

func daysInMonth(year int, month time.Month, loc *time.Location) int {
	start := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	return daysBetween(start, start.AddDate(0, 1, 0))
}

func russianMonthYear(t time.Time) string {
	months := []string{"", "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь", "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"}
	return fmt.Sprintf("%s %d", months[t.Month()], t.Year())
}

func insightPeriodStatus(isCurrent bool, isSeason bool) string {
	period := "месяца"
	if isSeason {
		period = "сезона"
	}
	if isCurrent {
		return fmt.Sprintf("%s в процессе", trimGenitivePeriod(period))
	}
	return fmt.Sprintf("итоговый отчёт %s", period)
}

func trimGenitivePeriod(period string) string {
	if period == "сезона" {
		return "сезон"
	}
	return "месяц"
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func value32(v *float32) float64 {
	if v == nil {
		return 0
	}
	return float64(*v)
}

func daysBetween(from, to time.Time) int {
	return int(to.Sub(from).Hours() / 24)
}

func dayStart(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func ratioPercent(value, baseline float64) float64 {
	if baseline <= 0 {
		if value > 0 {
			return 100
		}
		return 0
	}
	return value / baseline * 100
}

func clampPercent(value float64) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return int(math.Round(value))
}

func rainiestSharePercent(month models.MonthlyWeatherInsights) int {
	if month.MaxRainDay == nil || month.RainTotal <= 0 {
		return 0
	}
	return clampPercent(month.MaxRainDay.Value / month.RainTotal * 100)
}

func formatInsightDate(t time.Time) string {
	months := []string{"", "января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}
	return fmt.Sprintf("%d %s", t.Day(), months[t.Month()])
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
