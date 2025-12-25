package web

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/iRootPro/weather/internal/models"
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
	return template.ParseFiles(partialPath)
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
		WindMax        float32
		HasDailyData   bool
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

// SunTimesWidget renders the sun times widget
func (h *Handler) SunTimesWidget(w http.ResponseWriter, r *http.Request) {
	if h.sunService == nil {
		http.Error(w, "Sun service not configured", http.StatusInternalServerError)
		return
	}

	sunTimes := h.sunService.GetTodaySunTimes()

	templateData := struct {
		Date        string
		Dawn        string
		Sunrise     string
		Sunset      string
		Dusk        string
		DayLength   string
		LightLength string
	}{
		Date:        time.Now().Format("2 января"),
		Dawn:        sunTimes.Dawn.Format("15:04"),
		Sunrise:     sunTimes.Sunrise.Format("15:04"),
		Sunset:      sunTimes.Sunset.Format("15:04"),
		Dusk:        sunTimes.Dusk.Format("15:04"),
		DayLength:   formatDuration(sunTimes.DayLength),
		LightLength: formatDuration(sunTimes.LightLength),
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
