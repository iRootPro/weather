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

	previousCompareDays := minInt(now.Day(), daysBetween(previousStart, previousEnd))
	previousSameEnd := previousStart.AddDate(0, 0, previousCompareDays)
	previousSameDays := filterDaysBefore(previousDays, previousSameEnd)

	current := buildMonthlyInsights("Этот месяц", currentStart, now, currentDays, now.Day())
	previous := buildMonthlyInsights("Прошлый месяц", previousStart, previousEnd.Add(-time.Nanosecond), previousDays, daysBetween(previousStart, previousEnd))
	previousSame := buildMonthlyInsights(fmt.Sprintf("Прошлый месяц к %d числу", previousCompareDays), previousStart, previousSameEnd.Add(-time.Nanosecond), previousSameDays, previousCompareDays)

	allDays := append([]models.DailyWeatherInsight{}, previousDays...)
	allDays = append(allDays, currentDays...)
	currentDryStreak, lastRainDate, hasLastRain := calculateRainRecency(allDays, loc)

	mainInsight, stories := buildInsightStories(current, previous, previousSame, currentDryStreak, lastRainDate, hasLastRain)
	bestDay, worstDay := findNotableDays(currentDays)

	page := &models.WeatherInsightsPage{
		GeneratedAt:        now,
		CurrentMonth:       current,
		PreviousMonth:      previous,
		PreviousSamePeriod: previousSame,
		CurrentDryStreak:   currentDryStreak,
		HasLastRain:        hasLastRain,
		MainInsight:        mainInsight,
		Stories:            stories,
		Calendar:           buildCalendar(currentStart, currentEnd, currentDays, now, loc),
		RainChartData:      buildRainChartData(currentDays, previousDays, daysBetween(currentStart, currentEnd), daysBetween(previousStart, previousEnd), now.Day()),
		BestDay:            bestDay,
		WorstDay:           worstDay,
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

func buildRainChartData(currentDays, previousDays []models.DailyWeatherInsight, currentMonthDays, previousMonthDays, currentDay int) map[string]interface{} {
	maxDays := maxInt(currentMonthDays, previousMonthDays)
	labels := make([]string, maxDays)
	current := make([]interface{}, maxDays)
	previous := make([]interface{}, maxDays)
	currentByDay := make(map[int]float64, len(currentDays))
	previousByDay := make(map[int]float64, len(previousDays))

	for _, day := range currentDays {
		currentByDay[day.Date.Day()] = value32(day.RainTotal)
	}
	for _, day := range previousDays {
		previousByDay[day.Date.Day()] = value32(day.RainTotal)
	}

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
	}

	return map[string]interface{}{
		"labels":   labels,
		"current":  current,
		"previous": previous,
	}
}

func buildInsightStories(current, previous, previousSame models.MonthlyWeatherInsights, dryStreak int, lastRain time.Time, hasLastRain bool) (models.WeatherInsightStory, []models.WeatherInsightStory) {
	stories := make([]models.WeatherInsightStory, 0, 6)

	main := models.WeatherInsightStory{
		Icon:  "📊",
		Title: "Месяц набирает статистику",
		Text:  fmt.Sprintf("За %d дней собрано %.1f мм осадков и %d дождливых дней.", current.DaysWithData, current.RainTotal, current.RainDays),
	}

	if current.RainTotal > previous.RainTotal && current.RainDays < previous.RainDays {
		main = models.WeatherInsightStory{
			Icon:  "🌧️",
			Title: "Дождей меньше, но воды уже больше",
			Text:  fmt.Sprintf("Дождливых дней %d против %d, зато осадков уже %.1f мм против %.1f мм за весь прошлый месяц.", current.RainDays, previous.RainDays, current.RainTotal, previous.RainTotal),
		}
	} else if current.RainTotal > previous.RainTotal {
		main = models.WeatherInsightStory{
			Icon:  "💧",
			Title: "Месяц уже обогнал прошлый по осадкам",
			Text:  fmt.Sprintf("Выпало %.1f мм — это больше, чем %.1f мм за весь прошлый месяц.", current.RainTotal, previous.RainTotal),
		}
	} else if current.RainTotal > previousSame.RainTotal {
		main = models.WeatherInsightStory{
			Icon:  "📈",
			Title: "Темп осадков выше прошлого месяца",
			Text:  fmt.Sprintf("К этому же числу прошлого месяца было %.1f мм, сейчас уже %.1f мм.", previousSame.RainTotal, current.RainTotal),
		}
	}

	stories = append(stories, models.WeatherInsightStory{
		Icon:  "🗓️",
		Title: "Честное сравнение к той же дате",
		Text:  fmt.Sprintf("Сейчас %.1f мм за %d дней, в прошлом месяце к этому дню было %.1f мм.", current.RainTotal, current.DaysInPeriod, previousSame.RainTotal),
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
