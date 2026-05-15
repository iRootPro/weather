package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

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

// GetInsights returns human-friendly monthly weather analytics for the web page.
func (s *WeatherService) GetInsights(ctx context.Context) (*models.WeatherInsightsPage, error) {
	loc := s.location
	if loc == nil {
		loc = time.Local
	}

	now := time.Now().In(loc)
	currentStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	currentEnd := currentStart.AddDate(0, 1, 0)
	previousStart := currentStart.AddDate(0, -1, 0)
	previousEnd := currentStart

	currentDays, err := s.repo.GetDailyInsights(ctx, currentStart, now, s.timezone)
	if err != nil {
		return nil, err
	}
	previousDays, err := s.repo.GetDailyInsights(ctx, previousStart, previousEnd, s.timezone)
	if err != nil {
		return nil, err
	}

	rollingStart := dayStart(now, loc).AddDate(0, 0, -59)
	rollingDays, err := s.repo.GetDailyInsights(ctx, rollingStart, now, s.timezone)
	if err != nil {
		return nil, err
	}

	archiveDays := make([]models.DailyWeatherInsight, 0, 366)
	for year := now.Year() - 10; year < now.Year(); year++ {
		archiveMonthStart := time.Date(year, now.Month(), 1, 0, 0, 0, 0, loc)
		archiveMonthEnd := archiveMonthStart.AddDate(0, 1, 0)
		days, err := s.repo.GetDailyInsights(ctx, archiveMonthStart, archiveMonthEnd, s.timezone)
		if err != nil {
			return nil, err
		}
		archiveDays = append(archiveDays, days...)
	}

	seasonStart, seasonEnd := seasonBounds(now, loc)
	seasonDays, err := s.repo.GetDailyInsights(ctx, seasonStart, now, s.timezone)
	if err != nil {
		return nil, err
	}

	previousCompareDays := minInt(now.Day(), daysBetween(previousStart, previousEnd))
	previousSameEnd := previousStart.AddDate(0, 0, previousCompareDays)
	previousSameDays := filterDaysBefore(previousDays, previousSameEnd)

	current := buildMonthlyInsights("Этот месяц", currentStart, now, currentDays, now.Day())
	previous := buildMonthlyInsights("Прошлый месяц · справочно", previousStart, previousEnd.Add(-time.Nanosecond), previousDays, daysBetween(previousStart, previousEnd))
	previousSame := buildMonthlyInsights(fmt.Sprintf("Прошлый месяц к %d числу", previousCompareDays), previousStart, previousSameEnd.Add(-time.Nanosecond), previousSameDays, previousCompareDays)
	seasonCurrent := buildMonthlyInsights("Текущий сезон", seasonStart, now, seasonDays, maxInt(1, daysBetween(seasonStart, now)))
	season := buildSeasonContext(now, seasonStart, seasonEnd, seasonCurrent)
	sameMonthBenchmark := buildSameMonthBenchmark(now, current, archiveDays, loc)
	last7Days := buildRollingPeriod("Последние 7 дней", "Сравнение с предыдущими 7 днями", rollingDays, now, 7, loc)
	last30Days := buildRollingPeriod("Последние 30 дней", "Сравнение с предыдущими 30 днями", rollingDays, now, 30, loc)

	allDays := append([]models.DailyWeatherInsight{}, rollingDays...)
	allDays = append(allDays, currentDays...)
	currentDryStreak, lastRainDate, hasLastRain := calculateRainRecency(allDays, loc)

	mainInsight, stories := buildInsightStories(current, previous, previousSame, sameMonthBenchmark, last7Days, season, currentDryStreak, lastRainDate, hasLastRain)
	bestDay, worstDay := findNotableDays(currentDays)

	page := &models.WeatherInsightsPage{
		GeneratedAt:               now,
		CurrentMonth:              current,
		PreviousMonth:             previous,
		PreviousSamePeriod:        previousSame,
		Season:                    season,
		SameMonthBenchmark:        sameMonthBenchmark,
		Last7Days:                 last7Days,
		Last30Days:                last30Days,
		CurrentDryStreak:          currentDryStreak,
		HasLastRain:               hasLastRain,
		MainInsight:               mainInsight,
		Stories:                   stories,
		MonthProgressPercent:      clampPercent(float64(now.Day()) / float64(daysBetween(currentStart, currentEnd)) * 100),
		RainVsPreviousSamePercent: clampPercent(ratioPercent(current.RainTotal, previousSame.RainTotal)),
		RainVsPreviousFullPercent: clampPercent(ratioPercent(current.RainTotal, previous.RainTotal)),
		RainiestDaySharePercent:   rainiestSharePercent(current),
		ComfortPercent:            clampPercent(ratioPercent(float64(current.ComfortableDays), float64(maxInt(current.DaysWithData, 1)))),
		SunnyPercent:              clampPercent(ratioPercent(float64(current.SunnyDays), float64(maxInt(current.DaysWithData, 1)))),
		RainDaysPercent:           clampPercent(ratioPercent(float64(current.RainDays), float64(maxInt(current.DaysWithData, 1)))),
		Calendar:                  buildCalendar(currentStart, currentEnd, currentDays, now, loc),
		RainChartData:             buildRainChartData(currentDays, previousDays, archiveDays, daysBetween(currentStart, currentEnd), daysBetween(previousStart, previousEnd), now.Day(), now.Month(), loc),
		BestDay:                   bestDay,
		WorstDay:                  worstDay,
	}
	if hasLastRain {
		page.LastRainDate = lastRainDate
		page.DaysSinceLastRain = int(dayStart(now, loc).Sub(dayStart(lastRainDate, loc)).Hours() / 24)
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

func buildInsightStories(current, previous, previousSame models.MonthlyWeatherInsights, benchmark models.WeatherArchiveBenchmark, last7 models.RollingWeatherPeriod, season models.WeatherSeasonContext, dryStreak int, lastRain time.Time, hasLastRain bool) (models.WeatherInsightStory, []models.WeatherInsightStory) {
	stories := make([]models.WeatherInsightStory, 0, 7)

	main := models.WeatherInsightStory{
		Icon:  season.Icon,
		Title: season.FocusTitle,
		Text:  fmt.Sprintf("%s За %d дней месяца собрано %.1f мм осадков и %d дождливых дней.", season.FocusText, current.DaysWithData, current.RainTotal, current.RainDays),
	}

	if benchmark.Available && math.Abs(float64(benchmark.RainDeltaPercent)) >= 25 {
		icon := "💧"
		title := "Месяц влажнее сезонной нормы"
		if benchmark.RainDeltaPercent < 0 {
			icon = "🌤️"
			title = "Месяц суше сезонной нормы"
		}
		main = models.WeatherInsightStory{
			Icon:  icon,
			Title: title,
			Text:  fmt.Sprintf("Для этого месяца года норма к текущей дате — около %.1f мм. Сейчас %.1f мм: %s", benchmark.RainTotalAvg, current.RainTotal, benchmark.Verdict),
		}
	} else if last7.Current.RainTotal >= last7.Previous.RainTotal*1.8 && last7.Current.RainTotal >= 5 {
		main = models.WeatherInsightStory{
			Icon:  "🌧️",
			Title: "Последняя неделя резко влажнее",
			Text:  fmt.Sprintf("За последние 7 дней выпало %.1f мм против %.1f мм неделей ранее. Это лучше отражает текущую погоду, чем сравнение с прошлым месяцем.", last7.Current.RainTotal, last7.Previous.RainTotal),
		}
	} else if current.RainTotal > previousSame.RainTotal {
		main = models.WeatherInsightStory{
			Icon:  "📈",
			Title: "Темп осадков выше прошлого месяца",
			Text:  fmt.Sprintf("Это справочное сравнение: к этому же числу прошлого месяца было %.1f мм, сейчас %.1f мм. Сезонный контекст важнее.", previousSame.RainTotal, current.RainTotal),
		}
	}

	stories = append(stories, models.WeatherInsightStory{
		Icon:  season.Icon,
		Title: season.Title,
		Text:  season.Description,
	})

	if benchmark.Available {
		stories = append(stories, models.WeatherInsightStory{
			Icon:  "📚",
			Title: "Сравнение с такими же месяцами",
			Text:  fmt.Sprintf("Архивных сезонов: %d. Осадки: %.1f мм против средней %.1f мм к этой дате.", benchmark.SampleSize, current.RainTotal, benchmark.RainTotalAvg),
		})
	} else {
		stories = append(stories, models.WeatherInsightStory{
			Icon:  "📚",
			Title: "Архив ещё копится",
			Text:  benchmark.StatusText,
		})
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
			Title: "Самый дождливый день месяца",
			Text:  fmt.Sprintf("%s — %.1f мм осадков, около %.0f%% месячной суммы.", formatInsightDate(current.MaxRainDay.Date), current.MaxRainDay.Value, share),
		})
	}

	if current.ComfortableDays > 0 {
		stories = append(stories, models.WeatherInsightStory{
			Icon:  "🚶",
			Title: "Дни для прогулок",
			Text:  fmt.Sprintf("Комфортных дней в этом месяце: %d из %d.", current.ComfortableDays, current.DaysWithData),
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
