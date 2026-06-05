package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

const quietPriorityThreshold = 20

// DashboardService собирает умный snapshot главного экрана: не просто метрики,
// а отсортированную по важности ленту того, что требует внимания сейчас.
type DashboardService struct {
	weatherService     *WeatherService
	forecastService    *ForecastService
	geomagneticService *GeomagneticService
	hydroService       *HydroService
}

func NewDashboardService(weatherService *WeatherService, forecastService *ForecastService, geomagneticService *GeomagneticService, hydroService *HydroService) *DashboardService {
	return &DashboardService{
		weatherService:     weatherService,
		forecastService:    forecastService,
		geomagneticService: geomagneticService,
		hydroService:       hydroService,
	}
}

func (s *DashboardService) GetSnapshot(ctx context.Context) (*models.DashboardSnapshot, error) {
	now := time.Now()
	snapshot := &models.DashboardSnapshot{
		GeneratedAt: now,
		Quiet: models.QuietSummary{
			Title: "Остальное спокойно",
			Items: []string{},
		},
	}

	var allCards []models.AttentionCard
	var current *models.WeatherData
	var hourAgo *models.WeatherData

	if s.weatherService != nil {
		var err error
		current, hourAgo, _, err = s.weatherService.GetCurrentWithHourlyChange(ctx)
		if err != nil || current == nil {
			snapshot.StationStatus = models.StationStatus{
				OK:       false,
				Label:    "нет свежих данных метеостанции",
				Severity: string(models.DashboardSeverityDanger),
			}
			allCards = append(allCards, models.AttentionCard{
				ID:       "station-no-data",
				Domain:   "station",
				Title:    "Нет данных метеостанции",
				Subtitle: "Не удалось получить последние измерения",
				Severity: string(models.DashboardSeverityDanger),
				Priority: 95,
				Reason:   "последние данные недоступны",
				Icon:     "⚠️",
			})
		} else {
			snapshot.StationStatus = buildStationStatus(current, now)
			if card := buildStationFreshnessCard(snapshot.StationStatus); card != nil {
				allCards = append(allCards, *card)
			}
			allCards = append(allCards, buildCurrentWeatherCard(current, hourAgo))
			allCards = append(allCards, buildWindCard(current))
			allCards = append(allCards, buildRainCard(current, nil))
			if card := buildPressureCard(current, hourAgo); card != nil {
				allCards = append(allCards, *card)
			}
			if card := buildUVCard(current, now); card != nil {
				allCards = append(allCards, *card)
			}
		}

		if events, err := s.weatherService.GetRecentEvents(ctx, 24); err == nil {
			allCards = append(allCards, buildEventCards(events, now)...)
		}
	} else {
		snapshot.StationStatus = models.StationStatus{OK: false, Label: "weather service не настроен", Severity: string(models.DashboardSeverityDanger)}
	}

	if s.forecastService != nil {
		if forecast, err := s.forecastService.GetHourlyForecast(ctx, 6); err == nil {
			if card := buildForecastRainCard(forecast, current); card != nil {
				allCards = append(allCards, *card)
			}
		}
	}

	if s.geomagneticService != nil {
		if card := s.buildGeomagneticAttentionCard(ctx, now); card != nil {
			allCards = append(allCards, *card)
		}
	}

	if s.hydroService != nil {
		if card := s.buildHydroAttentionCard(ctx, now); card != nil {
			allCards = append(allCards, *card)
		}
	}

	cards, quiet := splitAndSortCards(allCards)
	snapshot.Cards = cards
	snapshot.Quiet.Items = quiet
	snapshot.Headline = buildHeadline(cards)
	return snapshot, nil
}

func buildStationStatus(current *models.WeatherData, now time.Time) models.StationStatus {
	age := now.Sub(current.Time)
	ageMinutes := int(age.Minutes())
	lastSeen := current.Time
	status := models.StationStatus{
		OK:         age < 30*time.Minute,
		LastSeenAt: &lastSeen,
		AgeMinutes: &ageMinutes,
		Severity:   string(models.DashboardSeverityNormal),
		Label:      "данные свежие",
	}
	switch {
	case age >= time.Hour:
		status.OK = false
		status.Severity = string(models.DashboardSeverityDanger)
		status.Label = fmt.Sprintf("данные устарели на %d мин", ageMinutes)
	case age >= 30*time.Minute:
		status.OK = false
		status.Severity = string(models.DashboardSeverityWarning)
		status.Label = fmt.Sprintf("данные устарели на %d мин", ageMinutes)
	case age >= 10*time.Minute:
		status.OK = true
		status.Severity = string(models.DashboardSeverityInfo)
		status.Label = fmt.Sprintf("последнее обновление %d мин назад", ageMinutes)
	}
	return status
}

func buildStationFreshnessCard(status models.StationStatus) *models.AttentionCard {
	if status.AgeMinutes == nil || *status.AgeMinutes < 10 {
		return nil
	}
	priority := 35
	if *status.AgeMinutes >= 30 {
		priority = 78
	}
	if *status.AgeMinutes >= 60 {
		priority = 95
	}
	return &models.AttentionCard{
		ID:       "station-freshness",
		Domain:   "station",
		Title:    "Данные метеостанции устарели",
		Subtitle: status.Label,
		Severity: status.Severity,
		Priority: priority,
		Reason:   "свежесть данных влияет на доверие к остальным показателям",
		Icon:     "⚠️",
	}
}

func buildCurrentWeatherCard(current *models.WeatherData, hourAgo *models.WeatherData) models.AttentionCard {
	title := "Текущая погода"
	subtitle := "Последние данные метеостанции"
	value := "—"
	severity := models.DashboardSeverityNormal
	priority := 45
	icon := "🌤️"

	if current.TempOutdoor != nil {
		value = fmt.Sprintf("%.1f", *current.TempOutdoor)
		title = weatherComfortTitle(*current.TempOutdoor)
		switch {
		case *current.TempOutdoor >= 35:
			severity = models.DashboardSeverityDanger
			priority = 82
			icon = "🔥"
			subtitle = "Очень жарко"
		case *current.TempOutdoor >= 30:
			severity = models.DashboardSeverityWarning
			priority = 68
			icon = "🥵"
			subtitle = "Жарко"
		case *current.TempOutdoor <= -10:
			severity = models.DashboardSeverityWarning
			priority = 65
			icon = "🥶"
			subtitle = "Сильный мороз"
		case *current.TempOutdoor <= 0:
			severity = models.DashboardSeverityInfo
			priority = 52
			icon = "❄️"
			subtitle = "Холодно"
		}
	}

	if current.TempFeelsLike != nil && current.TempOutdoor != nil {
		subtitle = fmt.Sprintf("Ощущается как %.1f°", *current.TempFeelsLike)
	}
	if hourAgo != nil && current.TempOutdoor != nil && hourAgo.TempOutdoor != nil {
		change := *current.TempOutdoor - *hourAgo.TempOutdoor
		if math.Abs(float64(change)) >= 3 {
			priority = maxIntDashboard(priority, 72)
			severity = models.DashboardSeverityWarning
			if change > 0 {
				subtitle = fmt.Sprintf("Быстро теплеет: +%.1f° за час", change)
			} else {
				subtitle = fmt.Sprintf("Быстро холодает: %.1f° за час", change)
			}
		}
	}

	return models.AttentionCard{
		ID:        "weather-current",
		Domain:    "weather",
		Title:     title,
		Subtitle:  subtitle,
		Value:     value,
		Unit:      "°C",
		Severity:  string(severity),
		Priority:  models.ClampPriority(priority),
		Reason:    "базовое текущее состояние погоды",
		Icon:      icon,
		DetailURL: "/detail/temperature",
	}
}

func buildWindCard(current *models.WeatherData) models.AttentionCard {
	value := "—"
	subtitle := "Ветер слабый"
	severity := models.DashboardSeverityCalm
	priority := 12
	icon := "🍃"
	if current != nil && current.WindSpeed != nil {
		value = fmt.Sprintf("%.1f", *current.WindSpeed)
	}
	gust := float32(0)
	if current != nil && current.WindGust != nil {
		gust = *current.WindGust
	}
	speed := float32(0)
	if current != nil && current.WindSpeed != nil {
		speed = *current.WindSpeed
	}
	maxWind := maxFloat32Dashboard(speed, gust)
	switch {
	case maxWind >= 17:
		severity = models.DashboardSeverityDanger
		priority = 92
		subtitle = fmt.Sprintf("Штормовые порывы до %.1f м/с", maxWind)
		icon = "🌪️"
	case maxWind >= 15:
		severity = models.DashboardSeverityWarning
		priority = 82
		subtitle = fmt.Sprintf("Очень сильные порывы до %.1f м/с", maxWind)
		icon = "💨"
	case maxWind >= 10:
		severity = models.DashboardSeverityWarning
		priority = 70
		subtitle = fmt.Sprintf("Сильные порывы до %.1f м/с", maxWind)
		icon = "💨"
	case maxWind >= 5:
		severity = models.DashboardSeverityInfo
		priority = 36
		subtitle = "Ветрено"
		icon = "🌬️"
	}
	return models.AttentionCard{
		ID:        "wind-current",
		Domain:    "wind",
		Title:     "Ветер",
		Subtitle:  subtitle,
		Value:     value,
		Unit:      "м/с",
		Severity:  string(severity),
		Priority:  priority,
		Reason:    "порывы ветра влияют на безопасность и комфорт",
		Icon:      icon,
		DetailURL: "/detail/wind",
	}
}

func buildRainCard(current *models.WeatherData, _ []models.HourlyForecast) models.AttentionCard {
	value := "0.0"
	subtitle := "Дождя сейчас нет"
	severity := models.DashboardSeverityCalm
	priority := 10
	icon := "☁️"
	if current != nil && current.RainRate != nil {
		value = fmt.Sprintf("%.1f", *current.RainRate)
		switch {
		case *current.RainRate >= 7.5:
			severity = models.DashboardSeverityDanger
			priority = 88
			subtitle = "Ливень сейчас"
			icon = "⛈️"
		case *current.RainRate >= 2.5:
			severity = models.DashboardSeverityWarning
			priority = 74
			subtitle = "Сильный дождь сейчас"
			icon = "🌧️"
		case *current.RainRate >= 0.1:
			severity = models.DashboardSeverityInfo
			priority = 58
			subtitle = "Дождь идёт сейчас"
			icon = "🌧️"
		}
	}
	return models.AttentionCard{
		ID:        "rain-current",
		Domain:    "rain",
		Title:     "Осадки",
		Subtitle:  subtitle,
		Value:     value,
		Unit:      "мм/ч",
		Severity:  string(severity),
		Priority:  priority,
		Reason:    "текущая интенсивность дождя",
		Icon:      icon,
		DetailURL: "/detail/rain",
	}
}

func buildForecastRainCard(forecast []models.HourlyForecast, current *models.WeatherData) *models.AttentionCard {
	if current != nil && current.RainRate != nil && *current.RainRate >= 0.1 {
		return nil
	}
	var best *models.HourlyForecast
	for i := range forecast {
		f := forecast[i]
		if f.PrecipitationProbability >= 50 || f.Precipitation >= 0.2 {
			if best == nil || f.PrecipitationProbability > best.PrecipitationProbability || f.Precipitation > best.Precipitation {
				best = &f
			}
		}
	}
	if best == nil {
		return nil
	}
	priority := 50
	severity := models.DashboardSeverityInfo
	if best.PrecipitationProbability >= 80 || best.Precipitation >= 2 {
		priority = 66
		severity = models.DashboardSeverityWarning
	}
	return &models.AttentionCard{
		ID:        "rain-forecast",
		Domain:    "forecast",
		Title:     "Возможен дождь",
		Subtitle:  fmt.Sprintf("Прогноз на %s: вероятность %d%%", best.Time.In(time.Local).Format("15:04"), best.PrecipitationProbability),
		Value:     fmt.Sprintf("%d", best.PrecipitationProbability),
		Unit:      "%",
		Severity:  string(severity),
		Priority:  priority,
		Reason:    "в прогнозе на ближайшие часы есть осадки",
		Icon:      "🌧️",
		DetailURL: "/",
	}
}

func buildPressureCard(current, hourAgo *models.WeatherData) *models.AttentionCard {
	if current == nil || hourAgo == nil || current.PressureRelative == nil || hourAgo.PressureRelative == nil {
		return nil
	}
	change := *current.PressureRelative - *hourAgo.PressureRelative
	if math.Abs(float64(change)) < 1.5 {
		return nil
	}
	priority := 48
	severity := models.DashboardSeverityInfo
	if math.Abs(float64(change)) >= 3 {
		priority = 72
		severity = models.DashboardSeverityWarning
	}
	title := "Давление растёт"
	icon := "⬆️"
	if change < 0 {
		title = "Давление падает"
		icon = "⬇️"
	}
	return &models.AttentionCard{
		ID:        "pressure-trend",
		Domain:    "pressure",
		Title:     title,
		Subtitle:  fmt.Sprintf("%+.1f мм за час", change),
		Value:     fmt.Sprintf("%.0f", *current.PressureRelative),
		Unit:      "мм",
		Severity:  string(severity),
		Priority:  priority,
		Reason:    "быстрое изменение давления может означать смену погоды",
		Icon:      icon,
		DetailURL: "/detail/pressure",
	}
}

func buildUVCard(current *models.WeatherData, now time.Time) *models.AttentionCard {
	if current == nil || current.UVIndex == nil {
		return nil
	}
	hour := now.In(time.Local).Hour()
	if hour < 8 || hour > 19 || *current.UVIndex < 6 {
		return nil
	}
	priority := 64
	severity := models.DashboardSeverityWarning
	if *current.UVIndex >= 8 {
		priority = 78
	}
	return &models.AttentionCard{
		ID:        "uv-high",
		Domain:    "solar",
		Title:     "Высокий UV",
		Subtitle:  "Лучше избегать прямого солнца",
		Value:     fmt.Sprintf("%.0f", *current.UVIndex),
		Unit:      "UV",
		Severity:  string(severity),
		Priority:  priority,
		Reason:    "UV-индекс выше безопасного уровня",
		Icon:      "☀️",
		DetailURL: "/detail/solar",
	}
}

func buildEventCards(events []models.WeatherEvent, now time.Time) []models.AttentionCard {
	cards := make([]models.AttentionCard, 0, 3)
	for _, event := range events {
		if len(cards) >= 3 {
			break
		}
		if now.Sub(event.Time) > 6*time.Hour {
			continue
		}
		priority := 0
		severity := models.DashboardSeverityInfo
		domain := "weather"
		detailURL := "/"
		switch event.Type {
		case "wind_gust":
			priority = 70
			domain = "wind"
			detailURL = "/detail/wind"
			if event.Value >= 15 {
				priority = 82
				severity = models.DashboardSeverityWarning
			}
		case "pressure_drop", "pressure_rise":
			priority = 64
			domain = "pressure"
			detailURL = "/detail/pressure"
		case "temp_drop", "temp_rise":
			priority = 62
			domain = "weather"
			detailURL = "/detail/temperature"
		case "rain_start":
			priority = 62
			domain = "rain"
			detailURL = "/detail/rain"
		default:
			continue
		}
		cards = append(cards, models.AttentionCard{
			ID:        "event-" + event.Type,
			Domain:    domain,
			Title:     event.Description,
			Subtitle:  event.Details,
			Value:     formatEventValue(event),
			Severity:  string(severity),
			Priority:  priority,
			Reason:    "недавнее погодное событие",
			Icon:      event.Icon,
			DetailURL: detailURL,
		})
	}
	return cards
}

func (s *DashboardService) buildGeomagneticAttentionCard(ctx context.Context, now time.Time) *models.AttentionCard {
	snap, err := s.geomagneticService.GetDashboardSnapshot(ctx, now)
	if err != nil || snap == nil || !snap.HasData || snap.Current == nil {
		return nil
	}
	status := snap.Status
	priority := 10
	severity := models.DashboardSeverityCalm
	subtitle := "Магнитное поле спокойно"
	title := "Геомагнитная активность"
	if snap.Current.Kp >= 5 {
		priority = 86
		severity = models.DashboardSeverityDanger
		if gLevel, desc, ok := models.StormLevel(snap.Current.Kp); ok {
			title = "Магнитная буря " + gLevel
			subtitle = desc + " магнитная буря"
		} else {
			title = "Магнитная буря"
			subtitle = "Kp выше порога бури"
		}
	} else if snap.Current.Kp >= 4 {
		priority = 62
		severity = models.DashboardSeverityWarning
		title = "Геомагнитное возмущение"
		subtitle = "Kp близок к уровню магнитной бури"
	}
	if snap.NextStorm != nil && snap.NextStorm.SlotTime.After(now) {
		priority = maxIntDashboard(priority, 72)
		if severity == models.DashboardSeverityCalm {
			severity = models.DashboardSeverityWarning
		}
		when := snap.NextStorm.SlotTime.In(time.Local).Format("15:04")
		if gLevel, _, ok := models.StormLevel(snap.NextStorm.Kp); ok {
			subtitle = fmt.Sprintf("В прогнозе буря %s к %s", gLevel, when)
		}
	}
	return &models.AttentionCard{
		ID:        "geomagnetic-current",
		Domain:    "geomagnetic",
		Title:     title,
		Subtitle:  subtitle,
		Value:     fmt.Sprintf("%.1f", snap.Current.Kp),
		Unit:      "Kp",
		Severity:  string(severity),
		Priority:  models.ClampPriority(priority),
		Reason:    "геомагнитная активность по индексу Kp",
		Icon:      status.Emoji(),
		DetailURL: "/detail/geomagnetic",
	}
}

func (s *DashboardService) buildHydroAttentionCard(ctx context.Context, now time.Time) *models.AttentionCard {
	snap, err := s.hydroService.GetSnapshot(ctx, now)
	if err != nil || snap == nil || !snap.HasData || snap.Current == nil {
		return nil
	}
	priority := 12
	severity := models.DashboardSeverityCalm
	title := "Уровень воды в норме"
	subtitle := "Гидропост без опасных значений"
	icon := "💧"
	switch snap.Status {
	case models.HydroStatusDanger:
		priority = 96
		severity = models.DashboardSeverityDanger
		title = "Опасный уровень воды"
		icon = "🚨"
	case models.HydroStatusPrevention:
		priority = 88
		severity = models.DashboardSeverityDanger
		title = "Неблагоприятный уровень воды"
		icon = "⚠️"
	case models.HydroStatusNear:
		priority = 78
		severity = models.DashboardSeverityWarning
		title = "Вода близко к НЯ"
		icon = "⚠️"
	case models.HydroStatusUnknown:
		priority = 30
		severity = models.DashboardSeverityInfo
		title = "Уровень воды"
		subtitle = "Пороговые значения неизвестны"
	}
	if snap.ToPreventionM != nil {
		if *snap.ToPreventionM >= 0 {
			subtitle = fmt.Sprintf("До неблагоприятного уровня %.2f м", *snap.ToPreventionM)
		} else {
			subtitle = fmt.Sprintf("Неблагоприятный уровень превышен на %.2f м", math.Abs(float64(*snap.ToPreventionM)))
		}
	}
	if snap.Current.ChangeCmPerHour != nil {
		change := *snap.Current.ChangeCmPerHour
		switch {
		case change >= 3:
			priority += 18
			subtitle += fmt.Sprintf(" · быстрый рост %+0.f см/ч", change)
		case change >= 1:
			priority += 8
			subtitle += fmt.Sprintf(" · растёт %+0.f см/ч", change)
		case change <= -3:
			priority -= 8
			subtitle += fmt.Sprintf(" · быстро снижается %.0f см/ч", change)
		case change <= -1:
			priority -= 4
			subtitle += fmt.Sprintf(" · снижается %.0f см/ч", change)
		}
	}
	station := ""
	if snap.Gauge != nil {
		station = snap.Gauge.MonitoringObject
		if station != "" {
			subtitle = station + " · " + subtitle
		}
	}
	return &models.AttentionCard{
		ID:        "hydro-current",
		Domain:    "hydro",
		Title:     title,
		Subtitle:  subtitle,
		Value:     fmt.Sprintf("%.3f", snap.Current.LevelBSM),
		Unit:      "м БС",
		Severity:  string(severity),
		Priority:  models.ClampPriority(priority),
		Reason:    "уровень воды и расстояние до неблагоприятного порога",
		Icon:      icon,
		DetailURL: "/detail/water-level",
	}
}

func splitAndSortCards(allCards []models.AttentionCard) ([]models.AttentionCard, []string) {
	cards := make([]models.AttentionCard, 0, len(allCards))
	quiet := make([]string, 0)
	seenQuiet := map[string]bool{}
	for _, card := range allCards {
		card.Priority = models.ClampPriority(card.Priority)
		if card.Priority <= quietPriorityThreshold {
			label := quietLabel(card)
			if label != "" && !seenQuiet[label] {
				seenQuiet[label] = true
				quiet = append(quiet, label)
			}
			continue
		}
		cards = append(cards, card)
	}
	sort.SliceStable(cards, func(i, j int) bool {
		return cards[i].Priority > cards[j].Priority
	})
	return cards, quiet
}

func buildHeadline(cards []models.AttentionCard) models.DashboardHeadline {
	if len(cards) == 0 || cards[0].Priority < 55 {
		return models.DashboardHeadline{
			Title:    "Сейчас спокойно",
			Summary:  "Нет показателей, которые требуют внимания",
			Severity: string(models.DashboardSeverityCalm),
			Icon:     "🟢",
		}
	}
	card := cards[0]
	return models.DashboardHeadline{
		Title:    "Главное сейчас — " + strings.TrimSuffix(card.Title, "."),
		Summary:  card.Subtitle,
		Severity: card.Severity,
		Icon:     card.Icon,
	}
}

func quietLabel(card models.AttentionCard) string {
	switch card.Domain {
	case "hydro":
		return "вода"
	case "wind":
		return "ветер"
	case "rain":
		return "дождь"
	case "geomagnetic":
		return "геомагнитка"
	case "solar":
		return "UV"
	}
	return ""
}

func weatherComfortTitle(temp float32) string {
	switch {
	case temp >= 35:
		return "Очень жарко"
	case temp >= 30:
		return "Жарко"
	case temp <= -10:
		return "Морозно"
	case temp <= 0:
		return "Холодно"
	case temp >= 18 && temp <= 26:
		return "Комфортно"
	default:
		return "Текущая погода"
	}
}

func formatEventValue(event models.WeatherEvent) string {
	if event.Value == 0 {
		return ""
	}
	return fmt.Sprintf("%.1f", event.Value)
}

func maxIntDashboard(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat32Dashboard(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
