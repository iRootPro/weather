package service

import (
	"context"
	"fmt"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

type WeatherService struct {
	repo repository.WeatherRepository
}

func NewWeatherService(repo repository.WeatherRepository) *WeatherService {
	return &WeatherService{repo: repo}
}

func (s *WeatherService) GetCurrent(ctx context.Context) (*models.WeatherData, error) {
	return s.repo.GetLatest(ctx)
}

func (s *WeatherService) GetHistory(ctx context.Context, from, to time.Time, interval string) ([]models.WeatherData, error) {
	if interval == "" || interval == "raw" {
		return s.repo.GetByTimeRange(ctx, from, to)
	}
	return s.repo.GetAggregated(ctx, from, to, interval)
}

func (s *WeatherService) GetStats(ctx context.Context, period string) (*models.WeatherStats, error) {
	now := time.Now()
	var from time.Time

	switch period {
	case "day":
		from = now.AddDate(0, 0, -1)
	case "week":
		from = now.AddDate(0, 0, -7)
	case "month":
		from = now.AddDate(0, -1, 0)
	case "year":
		from = now.AddDate(-1, 0, 0)
	default:
		from = now.AddDate(0, 0, -1) // по умолчанию день
	}

	stats, err := s.repo.GetStats(ctx, from, now)
	if err != nil {
		return nil, err
	}
	stats.Period = period
	return stats, nil
}

func (s *WeatherService) GetChartData(ctx context.Context, from, to time.Time, interval string, fields []string) (*models.ChartData, error) {
	data, err := s.repo.GetAggregated(ctx, from, to, interval)
	if err != nil {
		return nil, err
	}

	chart := &models.ChartData{
		Labels:   make([]string, len(data)),
		Datasets: make(map[string][]float64),
	}

	// Инициализируем datasets для запрошенных полей
	for _, field := range fields {
		chart.Datasets[field] = make([]float64, len(data))
	}

	for i, d := range data {
		chart.Labels[i] = d.Time.Format("2006-01-02 15:04")

		for _, field := range fields {
			var val float64
			switch field {
			case "temp_outdoor":
				if d.TempOutdoor != nil {
					val = float64(*d.TempOutdoor)
				}
			case "temp_indoor":
				if d.TempIndoor != nil {
					val = float64(*d.TempIndoor)
				}
			case "humidity_outdoor":
				if d.HumidityOutdoor != nil {
					val = float64(*d.HumidityOutdoor)
				}
			case "humidity_indoor":
				if d.HumidityIndoor != nil {
					val = float64(*d.HumidityIndoor)
				}
			case "pressure_relative":
				if d.PressureRelative != nil {
					val = float64(*d.PressureRelative)
				}
			case "wind_speed":
				if d.WindSpeed != nil {
					val = float64(*d.WindSpeed)
				}
			case "wind_gust":
				if d.WindGust != nil {
					val = float64(*d.WindGust)
				}
			case "rain_daily":
				if d.RainDaily != nil {
					val = float64(*d.RainDaily)
				}
			case "rain_rate":
				if d.RainRate != nil {
					val = float64(*d.RainRate)
				}
			case "uv_index":
				if d.UVIndex != nil {
					val = float64(*d.UVIndex)
				}
			case "solar_radiation":
				if d.SolarRadiation != nil {
					val = float64(*d.SolarRadiation)
				}
			}
			chart.Datasets[field][i] = val
		}
	}

	return chart, nil
}

func (s *WeatherService) GetRecords(ctx context.Context) (*models.WeatherRecords, error) {
	return s.repo.GetRecords(ctx)
}

// GetCurrentWithHourlyChange returns current data, data from 1 hour ago, and daily min/max
func (s *WeatherService) GetCurrentWithHourlyChange(ctx context.Context) (current *models.WeatherData, hourAgo *models.WeatherData, dailyMinMax *repository.DailyMinMax, err error) {
	current, err = s.repo.GetLatest(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	// Получаем данные за час назад (игнорируем ошибку - данных может не быть)
	targetTime := time.Now().Add(-1 * time.Hour)
	hourAgo, _ = s.repo.GetDataNearTime(ctx, targetTime)

	// Получаем мин/макс за сутки (игнорируем ошибку)
	dailyMinMax, _ = s.repo.GetDailyMinMax(ctx)

	return current, hourAgo, dailyMinMax, nil
}

// GetDataAt returns weather data closest to the specified time
func (s *WeatherService) GetDataAt(ctx context.Context, targetTime time.Time) (*models.WeatherData, error) {
	return s.repo.GetDataNearTime(ctx, targetTime)
}

// Пороговые значения для событий
const (
	RAIN_THRESHOLD            = 0.1  // мм/ч - минимальная интенсивность для "дождя"
	TEMP_CHANGE_THRESHOLD     = 3.0  // °C за час
	WIND_GUST_THRESHOLD       = 12.0 // м/с (было 10.0)
	PRESSURE_CHANGE_THRESHOLD = 3.0  // мм рт.ст. за 3 часа
	PRESSURE_PERIOD_HOURS     = 3    // период для анализа изменения давления

	// Параметры фильтрации дождя
	MIN_RAIN_DURATION_MINUTES = 15 // минимальная длительность дождя
	MIN_RAIN_PAUSE_MINUTES    = 30 // минимальная пауза между дождями (паузы меньше игнорируются)
)

// GetRecentEvents returns detected weather events for the last N hours
func (s *WeatherService) GetRecentEvents(ctx context.Context, hours int) ([]models.WeatherEvent, error) {
	now := time.Now()
	from := now.Add(-time.Duration(hours) * time.Hour)

	// Получаем данные с интервалом 5 минут для анализа
	data, err := s.repo.GetDataForEventDetection(ctx, from, now)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return []models.WeatherEvent{}, nil
	}

	var events []models.WeatherEvent

	// Определяем события дождя
	rainEvents := detectRainEvents(data)
	events = append(events, rainEvents...)

	// Определяем изменения температуры
	tempEvents := detectTemperatureChanges(data)
	events = append(events, tempEvents...)

	// Определяем порывы ветра
	windEvents := detectWindGusts(data)
	events = append(events, windEvents...)

	// Определяем изменения давления
	pressureEvents := detectPressureChanges(data)
	events = append(events, pressureEvents...)

	// Сортируем события по времени (от новых к старым)
	sortEvents(events)

	// Ограничиваем количество событий для виджета
	if len(events) > 7 {
		events = events[:7]
	}

	return events, nil
}

// rainPeriod представляет период дождя
type rainPeriod struct {
	start time.Time
	end   time.Time
}

// detectRainEvents определяет начало и окончание дождя с фильтрацией коротких периодов
func detectRainEvents(data []models.WeatherData) []models.WeatherEvent {
	// 1. Найти все периоды дождя
	rainPeriods := findRainPeriods(data)
	if len(rainPeriods) == 0 {
		return []models.WeatherEvent{}
	}

	// 2. Объединить периоды с короткими паузами
	mergedPeriods := mergeRainPeriodsWithShortPauses(rainPeriods, MIN_RAIN_PAUSE_MINUTES)

	// 3. Отфильтровать короткие дожди
	significantPeriods := filterShortRains(mergedPeriods, MIN_RAIN_DURATION_MINUTES)

	// 4. Создать события для начала и конца каждого периода
	var events []models.WeatherEvent
	for _, period := range significantPeriods {
		// Определяем, идёт ли дождь сейчас
		isOngoing := len(data) > 0 && period.end.Equal(data[len(data)-1].Time) && data[len(data)-1].RainRate != nil && *data[len(data)-1].RainRate >= RAIN_THRESHOLD

		if isOngoing {
			// Дождь всё ещё идёт - показываем только событие начала
			duration := period.end.Sub(period.start)
			events = append(events, models.WeatherEvent{
				Type:        "rain_start",
				Time:        period.start,
				Value:       0,
				Change:      duration.Hours(),
				Description: fmt.Sprintf("Дождь идёт (%s)", formatRainDuration(duration)),
				Icon:        "🌧️",
			})
		} else {
			// Дождь закончился - показываем событие конца
			duration := period.end.Sub(period.start)
			events = append(events, models.WeatherEvent{
				Type:        "rain_end",
				Time:        period.end,
				Value:       0,
				Change:      duration.Hours(),
				Description: fmt.Sprintf("Дождь прошёл (%s)", formatRainDuration(duration)),
				Icon:        "☁️",
			})
		}
	}

	return events
}

// findRainPeriods находит все периоды с RainRate > RAIN_THRESHOLD
func findRainPeriods(data []models.WeatherData) []rainPeriod {
	var periods []rainPeriod
	var currentPeriod *rainPeriod

	for _, d := range data {
		isRaining := d.RainRate != nil && *d.RainRate >= RAIN_THRESHOLD

		if isRaining {
			if currentPeriod == nil {
				// Начало нового периода
				currentPeriod = &rainPeriod{start: d.Time, end: d.Time}
			} else {
				// Продолжение периода
				currentPeriod.end = d.Time
			}
		} else {
			if currentPeriod != nil {
				// Конец периода
				periods = append(periods, *currentPeriod)
				currentPeriod = nil
			}
		}
	}

	// Если период не закончился
	if currentPeriod != nil {
		periods = append(periods, *currentPeriod)
	}

	return periods
}

// mergeRainPeriodsWithShortPauses объединяет периоды с паузами < minPauseMinutes
func mergeRainPeriodsWithShortPauses(periods []rainPeriod, minPauseMinutes int) []rainPeriod {
	if len(periods) <= 1 {
		return periods
	}

	var merged []rainPeriod
	current := periods[0]

	for i := 1; i < len(periods); i++ {
		pauseDuration := periods[i].start.Sub(current.end)

		if pauseDuration < time.Duration(minPauseMinutes)*time.Minute {
			// Пауза короткая - объединяем периоды
			current.end = periods[i].end
		} else {
			// Пауза длинная - сохраняем текущий период и начинаем новый
			merged = append(merged, current)
			current = periods[i]
		}
	}

	// Добавляем последний период
	merged = append(merged, current)

	return merged
}

// filterShortRains удаляет дожди длительностью < minDurationMinutes
func filterShortRains(periods []rainPeriod, minDurationMinutes int) []rainPeriod {
	var filtered []rainPeriod

	for _, period := range periods {
		duration := period.end.Sub(period.start)
		if duration >= time.Duration(minDurationMinutes)*time.Minute {
			filtered = append(filtered, period)
		}
	}

	return filtered
}

// detectTemperatureChanges определяет резкие изменения температуры
func detectTemperatureChanges(data []models.WeatherData) []models.WeatherEvent {
	var events []models.WeatherEvent

	// Проверяем изменения за час (12 точек по 5 минут)
	for i := 12; i < len(data); i++ {
		curr := data[i]
		prev := data[i-12] // час назад

		if curr.TempOutdoor == nil || prev.TempOutdoor == nil {
			continue
		}

		change := *curr.TempOutdoor - *prev.TempOutdoor
		currTemp := float64(*curr.TempOutdoor)
		prevTemp := float64(*prev.TempOutdoor)

		if change >= TEMP_CHANGE_THRESHOLD {
			// Температура выросла
			events = append(events, models.WeatherEvent{
				Type:        "temp_rise",
				Time:        curr.Time,
				Value:       currTemp,
				ValueFrom:   prevTemp,
				Change:      float64(change),
				Period:      "за час",
				Description: fmt.Sprintf("Потеплело на %.1f°C", change),
				Details:     fmt.Sprintf("%.1f → %.1f°C за час", prevTemp, currTemp),
				Icon:        "🌡️",
			})
		} else if change <= -TEMP_CHANGE_THRESHOLD {
			// Температура упала
			events = append(events, models.WeatherEvent{
				Type:        "temp_drop",
				Time:        curr.Time,
				Value:       currTemp,
				ValueFrom:   prevTemp,
				Change:      float64(change),
				Period:      "за час",
				Description: fmt.Sprintf("Похолодало на %.1f°C", -change),
				Details:     fmt.Sprintf("%.1f → %.1f°C за час", prevTemp, currTemp),
				Icon:        "🥶",
			})
		}
	}

	// Группируем близкие по времени события
	return groupSimilarEvents(events, 30*time.Minute)
}

// detectWindGusts определяет сильные порывы ветра
func detectWindGusts(data []models.WeatherData) []models.WeatherEvent {
	var events []models.WeatherEvent

	for _, d := range data {
		if d.WindGust != nil && *d.WindGust >= WIND_GUST_THRESHOLD {
			events = append(events, models.WeatherEvent{
				Type:        "wind_gust",
				Time:        d.Time,
				Value:       float64(*d.WindGust),
				Change:      0,
				Description: fmt.Sprintf("Порыв ветра %.1f м/с", *d.WindGust),
				Icon:        "💨",
			})
		}
	}

	// Группируем близкие порывы и берем максимальный
	return groupWindGusts(events, 30*time.Minute)
}

// detectPressureChanges определяет резкие изменения давления
func detectPressureChanges(data []models.WeatherData) []models.WeatherEvent {
	var events []models.WeatherEvent

	// Проверяем изменения за 3 часа (36 точек по 5 минут)
	for i := 36; i < len(data); i++ {
		curr := data[i]
		prev := data[i-36] // 3 часа назад

		if curr.PressureRelative == nil || prev.PressureRelative == nil {
			continue
		}

		change := *curr.PressureRelative - *prev.PressureRelative
		currPress := float64(*curr.PressureRelative)
		prevPress := float64(*prev.PressureRelative)

		if change >= PRESSURE_CHANGE_THRESHOLD {
			// Давление выросло
			events = append(events, models.WeatherEvent{
				Type:        "pressure_rise",
				Time:        curr.Time,
				Value:       currPress,
				ValueFrom:   prevPress,
				Change:      float64(change),
				Period:      "за 3 часа",
				Description: fmt.Sprintf("Давление растёт (+%.1f мм)", change),
				Details:     fmt.Sprintf("%.0f → %.0f мм за 3 часа", prevPress, currPress),
				Icon:        "⬆️",
			})
		} else if change <= -PRESSURE_CHANGE_THRESHOLD {
			// Давление упало
			events = append(events, models.WeatherEvent{
				Type:        "pressure_drop",
				Time:        curr.Time,
				Value:       currPress,
				ValueFrom:   prevPress,
				Change:      float64(change),
				Period:      "за 3 часа",
				Description: fmt.Sprintf("Давление падает (%.1f мм)", change),
				Details:     fmt.Sprintf("%.0f → %.0f мм за 3 часа", prevPress, currPress),
				Icon:        "⬇️",
			})
		}
	}

	// Группируем близкие по времени события
	return groupSimilarEvents(events, 60*time.Minute)
}

// formatRainDuration форматирует длительность дождя (только время, без текста)
func formatRainDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dч %dм", hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dч", hours)
	} else if minutes > 0 {
		return fmt.Sprintf("%dм", minutes)
	}
	return "менее минуты"
}

// groupSimilarEvents группирует похожие события, оставляя самое значимое
func groupSimilarEvents(events []models.WeatherEvent, window time.Duration) []models.WeatherEvent {
	if len(events) == 0 {
		return events
	}

	// Сортируем по времени
	sortEvents(events)

	var grouped []models.WeatherEvent
	i := 0

	for i < len(events) {
		// Начинаем новую группу
		maxEvent := events[i]
		j := i + 1

		// Ищем все события в пределах окна
		for j < len(events) && events[j].Time.Sub(events[i].Time) <= window {
			// Берем событие с максимальным изменением
			if abs(events[j].Change) > abs(maxEvent.Change) {
				maxEvent = events[j]
			}
			j++
		}

		grouped = append(grouped, maxEvent)
		i = j
	}

	return grouped
}

// groupWindGusts группирует порывы ветра, оставляя максимальный в окне
func groupWindGusts(events []models.WeatherEvent, window time.Duration) []models.WeatherEvent {
	if len(events) == 0 {
		return events
	}

	// Сортируем по времени
	sortEvents(events)

	var grouped []models.WeatherEvent
	i := 0

	for i < len(events) {
		// Начинаем новую группу
		maxEvent := events[i]
		j := i + 1

		// Ищем все порывы в пределах окна
		for j < len(events) && events[j].Time.Sub(events[i].Time) <= window {
			// Берем максимальный порыв
			if events[j].Value > maxEvent.Value {
				maxEvent = events[j]
			}
			j++
		}

		grouped = append(grouped, maxEvent)
		i = j
	}

	return grouped
}

// sortEvents сортирует события по времени (от новых к старым)
func sortEvents(events []models.WeatherEvent) {
	// Простая сортировка пузырьком (достаточно для небольшого количества событий)
	for i := 0; i < len(events)-1; i++ {
		for j := 0; j < len(events)-i-1; j++ {
			if events[j].Time.Before(events[j+1].Time) {
				events[j], events[j+1] = events[j+1], events[j]
			}
		}
	}
}

// abs возвращает абсолютное значение
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// GetLatest возвращает последние данные о погоде
func (s *WeatherService) GetLatest(ctx context.Context) (*models.WeatherData, error) {
	return s.repo.GetLatest(ctx)
}

// GetDataNearTime возвращает данные о погоде около указанного времени
func (s *WeatherService) GetDataNearTime(ctx context.Context, targetTime time.Time) (*models.WeatherData, error) {
	return s.repo.GetDataNearTime(ctx, targetTime)
}

// GetMinMaxInRange возвращает минимальную и максимальную температуру в указанном диапазоне
func (s *WeatherService) GetMinMaxInRange(ctx context.Context, from, to time.Time) (*repository.DailyMinMax, error) {
	// Получаем статистику за период
	stats, err := s.repo.GetStats(ctx, from, to)
	if err != nil {
		return nil, err
	}

	return &repository.DailyMinMax{
		TempMin: stats.TempOutdoorMin,
		TempMax: stats.TempOutdoorMax,
	}, nil
}

// GetDailyMinMax возвращает минимальную и максимальную температуру за сегодня
func (s *WeatherService) GetDailyMinMax(ctx context.Context) (*repository.DailyMinMax, error) {
	return s.repo.GetDailyMinMax(ctx)
}
