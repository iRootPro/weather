package web

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/service"
)

// degreesToDirection converts wind direction in degrees to compass direction
func degreesToDirection(degrees int16) string {
	// Normalize to 0-360
	deg := int(degrees) % 360
	if deg < 0 {
		deg += 360
	}

	directions := []string{"С", "СВ", "В", "ЮВ", "Ю", "ЮЗ", "З", "СЗ"}
	// Each direction covers 45 degrees, offset by 22.5 degrees
	index := ((deg + 22) % 360) / 45
	return directions[index]
}

func (h *Handler) parsePartial(name string) (*template.Template, error) {
	partialPath := filepath.Join(h.templatesDir, "partials", name)
	return template.New(name).Funcs(templateFuncs).ParseFiles(partialPath)
}

// CurrentWeatherWidget renders the current weather widget
func (h *Handler) CurrentWeatherWidget(w http.ResponseWriter, r *http.Request) {
	data, hourAgo, dailyMinMax, err := h.weatherService.GetCurrentWithHourlyChange(r.Context())
	if err != nil {
		slog.Error("failed to get current weather", "error", err)
		http.Error(w, "Failed to load weather data", http.StatusInternalServerError)
		return
	}

	// Convert pointer values for template
	templateData := struct {
		Time             string
		TempOutdoor      float32
		TempFeelsLike    float32
		DewPoint         float32
		IsFoggy          bool
		IsRaining        bool
		RainIntensity    string // "дождь", "сильный дождь", "ливень"
		IsWindy          bool
		WindIntensity    string // "ветер", "сильный ветер", "шторм"
		HumidityOutdoor  int16
		PressureRelative float32
		WindSpeed        float32
		WindGust         float32
		WindDirection    int16
		WindDirectionStr string
		RainRate         float32
		RainDaily        float32
		RainMonthly      float32
		UVIndex          float32
		SolarRadiation   float32
		Illuminance      float32 // lux = solar radiation * 120
		// Hourly changes
		TempChange     float32
		HumidityChange int16
		PressureChange float32
		HasHourlyData  bool
		// Daily min/max
		TempMin        float32
		TempMax        float32
		HumidityMin    int16
		HumidityMax    int16
		PressureMin    float32
		PressureMax    float32
		WindMax      float32
		WindGustMax  float32
		HasDailyData bool
	}{
		Time: "Данные на " + data.Time.Format("15:04"),
	}

	// Check if we have hourly comparison data
	templateData.HasHourlyData = hourAgo != nil
	// Check if we have daily min/max data
	templateData.HasDailyData = dailyMinMax != nil

	if data.TempOutdoor != nil {
		templateData.TempOutdoor = *data.TempOutdoor
		if hourAgo != nil && hourAgo.TempOutdoor != nil {
			templateData.TempChange = *data.TempOutdoor - *hourAgo.TempOutdoor
		}
	}
	if data.HumidityOutdoor != nil {
		templateData.HumidityOutdoor = *data.HumidityOutdoor
		if hourAgo != nil && hourAgo.HumidityOutdoor != nil {
			templateData.HumidityChange = *data.HumidityOutdoor - *hourAgo.HumidityOutdoor
		}
	}
	if data.PressureRelative != nil {
		templateData.PressureRelative = *data.PressureRelative
		if hourAgo != nil && hourAgo.PressureRelative != nil {
			templateData.PressureChange = *data.PressureRelative - *hourAgo.PressureRelative
		}
	}
	if data.WindSpeed != nil {
		templateData.WindSpeed = *data.WindSpeed
		// Определяем интенсивность ветра
		if *data.WindSpeed >= 5 {
			templateData.IsWindy = true
			switch {
			case *data.WindSpeed >= 17:
				templateData.WindIntensity = "шторм"
			case *data.WindSpeed >= 10:
				templateData.WindIntensity = "сильный ветер"
			default:
				templateData.WindIntensity = "ветер"
			}
		}
	}
	if data.WindGust != nil {
		templateData.WindGust = *data.WindGust
	}
	if data.WindDirection != nil {
		templateData.WindDirection = *data.WindDirection
		templateData.WindDirectionStr = degreesToDirection(*data.WindDirection)
	}
	if data.RainRate != nil {
		templateData.RainRate = *data.RainRate
		// Определяем интенсивность дождя
		if *data.RainRate > 0 {
			templateData.IsRaining = true
			switch {
			case *data.RainRate >= 7.5:
				templateData.RainIntensity = "ливень"
			case *data.RainRate >= 2.5:
				templateData.RainIntensity = "сильный дождь"
			default:
				templateData.RainIntensity = "дождь"
			}
		}
	}
	if data.RainDaily != nil {
		templateData.RainDaily = *data.RainDaily
	}
	if data.RainMonthly != nil {
		templateData.RainMonthly = *data.RainMonthly
	}
	if data.UVIndex != nil {
		templateData.UVIndex = *data.UVIndex
	}
	if data.SolarRadiation != nil {
		templateData.SolarRadiation = *data.SolarRadiation
		templateData.Illuminance = *data.SolarRadiation * 120 // approximate conversion to lux
	}
	if data.DewPoint != nil {
		templateData.DewPoint = *data.DewPoint
		// Определяем туман если разница между температурой и точкой росы < 2.5°C
		if data.TempOutdoor != nil {
			templateData.IsFoggy = models.IsFoggy(float64(*data.TempOutdoor), float64(*data.DewPoint))
		}
	}
	if data.TempFeelsLike != nil {
		templateData.TempFeelsLike = *data.TempFeelsLike
	} else if data.TempOutdoor != nil {
		// Если нет рассчитанной, используем реальную температуру
		templateData.TempFeelsLike = *data.TempOutdoor
	}

	// Daily min/max
	if dailyMinMax != nil {
		if dailyMinMax.TempMin != nil {
			templateData.TempMin = *dailyMinMax.TempMin
		}
		if dailyMinMax.TempMax != nil {
			templateData.TempMax = *dailyMinMax.TempMax
		}
		if dailyMinMax.HumidityMin != nil {
			templateData.HumidityMin = *dailyMinMax.HumidityMin
		}
		if dailyMinMax.HumidityMax != nil {
			templateData.HumidityMax = *dailyMinMax.HumidityMax
		}
		if dailyMinMax.PressureMin != nil {
			templateData.PressureMin = *dailyMinMax.PressureMin
		}
		if dailyMinMax.PressureMax != nil {
			templateData.PressureMax = *dailyMinMax.PressureMax
		}
		if dailyMinMax.WindMax != nil {
			templateData.WindMax = *dailyMinMax.WindMax
		}
		if dailyMinMax.GustMax != nil {
			templateData.WindGustMax = *dailyMinMax.GustMax
		}
	}

	tmpl, err := h.parsePartial("current_weather.html")
	if err != nil {
		slog.Error("failed to parse current weather template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render current weather widget", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// StatsWidget renders the daily stats widget
func (h *Handler) StatsWidget(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "day"
	}

	stats, err := h.weatherService.GetStats(r.Context(), period)
	if err != nil {
		slog.Error("failed to get stats", "error", err)
		http.Error(w, "Failed to load stats", http.StatusInternalServerError)
		return
	}

	// Convert pointer values for template
	templateData := struct {
		TempOutdoorMin      float32
		TempOutdoorMax      float32
		TempOutdoorAvg      float32
		PressureRelativeMin float32
		PressureRelativeMax float32
		PressureRelativeAvg float32
		WindSpeedMax        float32
		RainTotal           float32
	}{}

	if stats.TempOutdoorMin != nil {
		templateData.TempOutdoorMin = *stats.TempOutdoorMin
	}
	if stats.TempOutdoorMax != nil {
		templateData.TempOutdoorMax = *stats.TempOutdoorMax
	}
	if stats.TempOutdoorAvg != nil {
		templateData.TempOutdoorAvg = *stats.TempOutdoorAvg
	}
	if stats.PressureRelativeMin != nil {
		templateData.PressureRelativeMin = *stats.PressureRelativeMin
	}
	if stats.PressureRelativeMax != nil {
		templateData.PressureRelativeMax = *stats.PressureRelativeMax
	}
	if stats.PressureRelativeAvg != nil {
		templateData.PressureRelativeAvg = *stats.PressureRelativeAvg
	}
	if stats.WindSpeedMax != nil {
		templateData.WindSpeedMax = *stats.WindSpeedMax
	}
	if stats.RainTotal != nil {
		templateData.RainTotal = *stats.RainTotal
	}

	tmpl, err := h.parsePartial("daily_stats.html")
	if err != nil {
		slog.Error("failed to parse stats template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render stats widget", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// formatDuration formats duration as "Xч Yмин"
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dч %dмин", hours, minutes)
}

// formatDurationChange formats duration change with sign and appropriate units
func formatDurationChange(d time.Duration) string {
	sign := ""
	if d > 0 {
		sign = "+"
	}

	totalSeconds := int(d.Seconds())
	if totalSeconds < 0 {
		totalSeconds = -totalSeconds
	}

	minutes := totalSeconds / 60
	seconds := totalSeconds % 60

	if minutes == 0 {
		return fmt.Sprintf("%s%dсек", sign, int(d.Seconds()))
	}
	if seconds == 0 {
		return fmt.Sprintf("%s%dмин", sign, int(d.Minutes()))
	}
	return fmt.Sprintf("%s%dмин %dсек", sign, int(d.Minutes()), seconds)
}

// SunTimesWidget renders the sun and moon times widget
func (h *Handler) SunTimesWidget(w http.ResponseWriter, r *http.Request) {
	slog.Info("SunTimesWidget called")

	if h.sunService == nil {
		slog.Error("sun service is nil")
		http.Error(w, "Sun service not configured", http.StatusInternalServerError)
		return
	}

	sunTimes := h.sunService.GetTodaySunTimesWithComparison()

	// Get moon data if moon service is available
	var moonData *service.MoonData
	if h.moonService != nil {
		moonData = h.moonService.GetTodayMoonData()
	}

	templateData := struct {
		Date             string
		Dawn             string
		Sunrise          string
		Sunset           string
		Dusk             string
		DayLength        string
		LightLength      string
		DayChangeDay     string
		DayChangeWeek    string
		DayChangeMonth   string
		LightChangeDay   string
		LightChangeWeek  string
		LightChangeMonth string
		// For CSS classes (positive = growing day)
		DayChangePositive   bool
		LightChangePositive bool
		// Moon data
		HasMoonData         bool
		MoonPhase           string
		MoonPhaseIcon       string
		MoonIllumination    float64
		MoonAge             float64
		Moonrise            string
		Moonset             string
		DaysToNextPhase     float64
		NextPhaseName       string
	}{
		Date:                time.Now().Format("2 января"),
		Dawn:                sunTimes.Dawn.Format("15:04"),
		Sunrise:             sunTimes.Sunrise.Format("15:04"),
		Sunset:              sunTimes.Sunset.Format("15:04"),
		Dusk:                sunTimes.Dusk.Format("15:04"),
		DayLength:           formatDuration(sunTimes.DayLength),
		LightLength:         formatDuration(sunTimes.LightLength),
		DayChangeDay:        formatDurationChange(sunTimes.DayChangeDay),
		DayChangeWeek:       formatDurationChange(sunTimes.DayChangeWeek),
		DayChangeMonth:      formatDurationChange(sunTimes.DayChangeMonth),
		LightChangeDay:      formatDurationChange(sunTimes.LightChangeDay),
		LightChangeWeek:     formatDurationChange(sunTimes.LightChangeWeek),
		LightChangeMonth:    formatDurationChange(sunTimes.LightChangeMonth),
		DayChangePositive:   sunTimes.DayChangeDay >= 0,
		LightChangePositive: sunTimes.LightChangeDay >= 0,
		HasMoonData:         moonData != nil,
	}

	// Add moon data if available
	if moonData != nil {
		templateData.MoonPhase = moonData.PhaseName
		templateData.MoonPhaseIcon = moonData.PhaseIcon
		templateData.MoonIllumination = moonData.Illumination
		templateData.MoonAge = moonData.Age
		templateData.Moonrise = moonData.Moonrise.Format("15:04")
		templateData.Moonset = moonData.Moonset.Format("15:04")

		// Determine next major phase (full moon or new moon)
		// Synodic month is 29.53 days, full moon is at ~14.765 days
		const fullMoonAge = 14.765
		const synodicMonth = 29.53

		if moonData.Age < fullMoonAge {
			// Waxing moon - show days to full moon
			templateData.DaysToNextPhase = fullMoonAge - moonData.Age
			templateData.NextPhaseName = "До полнолуния"
		} else {
			// Waning moon - show days to new moon
			templateData.DaysToNextPhase = synodicMonth - moonData.Age
			templateData.NextPhaseName = "До новолуния"
		}
	}

	tmpl, err := h.parsePartial("sun_times.html")
	if err != nil {
		slog.Error("failed to parse sun times template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render sun times widget", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// WeatherEventsWidget renders the weather events widget
func (h *Handler) WeatherEventsWidget(w http.ResponseWriter, r *http.Request) {
	hours := 24 // показываем события за последние 24 часа
	events, err := h.weatherService.GetRecentEvents(r.Context(), hours)
	if err != nil {
		slog.Error("failed to get weather events", "error", err)
		http.Error(w, "Failed to load weather events", http.StatusInternalServerError)
		return
	}

	templateData := struct {
		Events   []models.WeatherEvent
		NoEvents bool
	}{
		Events:   events,
		NoEvents: len(events) == 0,
	}

	tmpl, err := h.parsePartial("weather_events.html")
	if err != nil {
		slog.Error("failed to parse weather events template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render weather events widget", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ForecastWidget renders the weather forecast widget
func (h *Handler) ForecastWidget(w http.ResponseWriter, r *http.Request) {
	if h.forecastService == nil {
		slog.Error("forecast service is nil")
		http.Error(w, "Forecast service not configured", http.StatusInternalServerError)
		return
	}

	now := time.Now()

	// Получаем почасовой прогноз на следующие 12 часов (чтобы гарантированно было 3-4 карточки)
	hourlyForecast, err := h.forecastService.GetHourlyForecast(r.Context(), 12)
	if err != nil {
		slog.Error("failed to get hourly forecast", "error", err)
		http.Error(w, "Failed to load forecast", http.StatusInternalServerError)
		return
	}

	// Получаем дневной прогноз начиная с завтра
	dailyForecast, err := h.forecastService.GetDailyForecast(r.Context(), 6)
	if err != nil {
		slog.Error("failed to get daily forecast", "error", err)
		http.Error(w, "Failed to load forecast", http.StatusInternalServerError)
		return
	}

	// Фильтруем дневной прогноз: исключаем сегодня
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	filteredDaily := make([]models.DailyForecast, 0)
	for _, df := range dailyForecast {
		if df.Date.After(tomorrow) || df.Date.Equal(tomorrow) {
			filteredDaily = append(filteredDaily, df)
		}
	}
	// Не ограничиваем количество дней - нужно столько, сколько нужно для 9 карточек

	// Единый тип карточки для всех прогнозов
	type ForecastCard struct {
		IsHourly                 bool   // true = почасовой, false = дневной
		Label                    string // "15:00" или "Пн"
		Icon                     string
		TempMain                 string // "-3°" или "-8/-2°"
		TempSecondary            string // "ощущ. -7°" для часов, пусто для дней
		WeatherDescription       string // "Снег", "Дождь", "Облачно" и т.д.
		PrecipitationProbability int16
		HasPrecipitation         bool
	}

	cards := make([]ForecastCard, 0)

	// Добавляем почасовые карточки (каждые 3 часа, максимум 3 карточки)
	hourCount := 0
	maxHours := 3
	for i, hf := range hourlyForecast {
		if hourCount >= maxHours {
			break
		}
		// Берём первый час и далее каждые 3 часа
		if i == 0 || i%3 == 0 {
			tempSecondary := ""
			if hf.FeelsLike != hf.Temperature {
				tempSecondary = fmt.Sprintf("ощущ. %.0f°", hf.FeelsLike)
			}
			card := ForecastCard{
				IsHourly:                 true,
				Label:                    hf.Time.Format("15:04"),
				Icon:                     hf.Icon,
				TempMain:                 fmt.Sprintf("%.0f°", hf.Temperature),
				TempSecondary:            tempSecondary,
				WeatherDescription:       hf.WeatherDescription,
				PrecipitationProbability: hf.PrecipitationProbability,
				HasPrecipitation:         hf.PrecipitationProbability > 0,
			}
			cards = append(cards, card)
			hourCount++
		}
	}

	// Добавляем дневные карточки (дополняем до 9 карточек)
	daysOfWeekShort := []string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
	totalCards := 9
	daysNeeded := totalCards - len(cards)

	for i, df := range filteredDaily {
		if i >= daysNeeded {
			break
		}
		card := ForecastCard{
			IsHourly:                 false,
			Label:                    daysOfWeekShort[df.Date.Weekday()],
			Icon:                     df.Icon,
			TempMain:                 fmt.Sprintf("%.0f/%.0f°", df.TemperatureMin, df.TemperatureMax),
			TempSecondary:            "",
			WeatherDescription:       df.WeatherDescription,
			PrecipitationProbability: df.PrecipitationProbability,
			HasPrecipitation:         df.PrecipitationProbability > 0,
		}
		cards = append(cards, card)
	}

	templateData := struct {
		Cards      []ForecastCard
		NoForecast bool
	}{
		Cards:      cards,
		NoForecast: len(cards) == 0,
	}

	tmpl, err := h.parsePartial("forecast.html")
	if err != nil {
		slog.Error("failed to parse forecast template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render forecast widget", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// NarodmonStatusWidget renders the narodmon status widget
func (h *Handler) NarodmonStatusWidget(w http.ResponseWriter, r *http.Request) {
	// Если сервис не настроен - не показываем виджет
	if h.narodmonService == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	status, err := h.narodmonService.GetStatus(r.Context())
	if err != nil {
		slog.Error("failed to get narodmon status", "error", err)
		http.Error(w, "Failed to load status", http.StatusInternalServerError)
		return
	}

	// Если нет отправок - не показываем виджет
	if status == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Форматируем время
	timeAgo := formatTimeAgo(status.LastSentAt)

	templateData := struct {
		Success      bool
		TimeAgo      string
		SensorsCount int
		ErrorMessage string
		DeviceURL    string
	}{
		Success:      status.Success,
		TimeAgo:      timeAgo,
		SensorsCount: status.SensorsCount,
		ErrorMessage: status.ErrorMessage,
		DeviceURL:    h.narodmonURL,
	}

	tmpl, err := h.parsePartial("narodmon_status.html")
	if err != nil {
		slog.Error("failed to parse narodmon status template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render narodmon status widget", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func formatTimeAgo(t *time.Time) string {
	if t == nil {
		return ""
	}

	duration := time.Since(*t)

	if duration < time.Minute {
		return "только что"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		return fmt.Sprintf("%d мин назад", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d ч назад", hours)
	} else {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d дн назад", days)
	}
}
