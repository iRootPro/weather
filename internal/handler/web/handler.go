package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
)

type Handler struct {
	templatesDir       string
	weatherService     *service.WeatherService
	sunService         *service.SunService
	moonService        *service.MoonService
	forecastService    *service.ForecastService
	photoRepo          repository.PhotoRepository
	narodmonService    *service.NarodmonService
	narodmonURL        string
	geomagneticService *service.GeomagneticService
	hydroService       *service.HydroService
	hydroCardCacheMu   sync.Mutex
	hydroCardCache     WaterLevelCardData
	hydroCardCachedAt  time.Time
}

func NewHandler(templatesDir string, weatherService *service.WeatherService, sunService *service.SunService, moonService *service.MoonService, forecastService *service.ForecastService, photoRepo repository.PhotoRepository, narodmonService *service.NarodmonService, narodmonURL string, geomagneticService *service.GeomagneticService, hydroService *service.HydroService) (*Handler, error) {
	return &Handler{
		templatesDir:       templatesDir,
		weatherService:     weatherService,
		sunService:         sunService,
		moonService:        moonService,
		forecastService:    forecastService,
		photoRepo:          photoRepo,
		narodmonService:    narodmonService,
		narodmonURL:        narodmonURL,
		geomagneticService: geomagneticService,
		hydroService:       hydroService,
	}, nil
}

var errInvalidInsightMonth = errors.New("invalid insight month")

var templateFuncs = template.FuncMap{
	"russianDate": func(t time.Time, format string) string {
		months := []string{"", "января", "февраля", "марта", "апреля", "мая", "июня",
			"июля", "августа", "сентября", "октября", "ноября", "декабря"}
		switch format {
		case "short":
			return fmt.Sprintf("%d %s", t.Day(), months[t.Month()])
		case "datetime":
			return fmt.Sprintf("%d %s %d, %s", t.Day(), months[t.Month()], t.Year(), t.Format("15:04"))
		default:
			return fmt.Sprintf("%d %s %d", t.Day(), months[t.Month()], t.Year())
		}
	},
	"mul": func(a, b float64) float64 {
		return a * b
	},
	"sub": func(a, b float64) float64 {
		return a - b
	},
	"deref": func(ptr interface{}) float64 {
		if ptr == nil {
			return 0
		}
		switch v := ptr.(type) {
		case *float32:
			if v == nil {
				return 0
			}
			return float64(*v)
		case *float64:
			if v == nil {
				return 0
			}
			return *v
		case *int16:
			if v == nil {
				return 0
			}
			return float64(*v)
		default:
			return 0
		}
	},
	"formatTime": func(t interface{}, format string) string {
		// Форматируем время в UTC, чтобы избежать смещения часового пояса
		// EXIF время уже в локальном формате, храним как UTC
		switch v := t.(type) {
		case time.Time:
			return v.UTC().Format(format)
		default:
			return ""
		}
	},
	"json": func(v interface{}) template.JS {
		data, err := json.Marshal(v)
		if err != nil {
			slog.Error("failed to marshal template json", "error", err)
			return template.JS("{}")
		}
		return template.JS(data)
	},
	"weatherIcon": weatherIcon,
}

// weatherIcon renders a small, predictable SVG from the forecast's WMO code.
// The markup is static for every branch, so it is safe to mark as template HTML.
func weatherIcon(code int16) template.HTML {
	const svgStart = `<svg class="ui-weather-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" aria-hidden="true" data-weather-icon="true">`
	const svgEnd = `</svg>`

	var path string
	switch {
	case code == 0:
		path = `<circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M4.93 4.93l1.41 1.41m11.32 11.32l1.41 1.41M2 12h2m16 0h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"/>`
	case code == 1 || code == 2:
		path = `<path d="M12 3v2m0 12v2m7-7h2M3 12h2m11.95-4.95l1.41-1.41M5.64 18.36l1.41-1.41"/><circle cx="12" cy="12" r="3.5"/><path d="M5 18h12a3 3 0 0 0 .3-5.98A5.5 5.5 0 0 0 7 10.5 3.75 3.75 0 0 0 5 18Z"/>`
	case code == 3:
		path = `<path d="M5 18h12a3 3 0 0 0 .3-5.98A5.5 5.5 0 0 0 7 10.5 3.75 3.75 0 0 0 5 18Z"/>`
	case code == 45 || code == 48:
		path = `<path d="M5 11h14M3 15h18M6 19h12"/>`
	case code >= 51 && code <= 57:
		path = `<path d="M5 14h12a3 3 0 0 0 .3-5.98A5.5 5.5 0 0 0 7 6.5 3.75 3.75 0 0 0 5 14Z"/><path d="M8 17l-1 2m5-2-1 2m5-2-1 2"/>`
	case code >= 61 && code <= 67 || code >= 80 && code <= 82:
		path = `<path d="M5 13h12a3 3 0 0 0 .3-5.98A5.5 5.5 0 0 0 7 5.5 3.75 3.75 0 0 0 5 13Z"/><path d="M8 16l-1 3m5-3-1 3m5-3-1 3"/>`
	case code >= 71 && code <= 77 || code >= 85 && code <= 86:
		path = `<path d="M5 13h12a3 3 0 0 0 .3-5.98A5.5 5.5 0 0 0 7 5.5 3.75 3.75 0 0 0 5 13Z"/><path d="M8 17h.01M12 17h.01M16 17h.01m-8 3h.01M12 20h.01M16 20h.01"/>`
	case code == 95 || code == 96 || code == 99:
		path = `<path d="M5 13h12a3 3 0 0 0 .3-5.98A5.5 5.5 0 0 0 7 5.5 3.75 3.75 0 0 0 5 13Z"/><path d="m12 15-2 4h3l-1 3 3-5h-3l1-2"/>`
	default:
		path = `<circle cx="12" cy="12" r="8"/><path d="M12 8v4m0 4h.01"/>`
	}
	return template.HTML(svgStart + path + svgEnd)
}

func (h *Handler) parseTemplate(name string) (*template.Template, error) {
	basePath := filepath.Join(h.templatesDir, "base.html")
	pagePath := filepath.Join(h.templatesDir, name)
	partialsPattern := filepath.Join(h.templatesDir, "partials", "*.html")

	// Parse base template first with functions, then the specific page
	tmpl, err := template.New(filepath.Base(basePath)).Funcs(templateFuncs).ParseFiles(basePath, pagePath)
	if err != nil {
		return nil, err
	}

	// Add partials
	tmpl, err = tmpl.ParseGlob(partialsPattern)
	if err != nil {
		// If no partials, that's ok
		slog.Debug("no partials found", "error", err)
	}

	return tmpl, nil
}

func (h *Handler) parseStandaloneTemplate(name string) (*template.Template, error) {
	pagePath := filepath.Join(h.templatesDir, name)
	tmpl, err := template.New(filepath.Base(pagePath)).Funcs(templateFuncs).ParseFiles(pagePath)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

type PageData struct {
	ActivePage string
	HasCharts  bool
	Data       interface{}
	Current    *models.WeatherData
}

// Dashboard renders the main dashboard page
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	// Exact match for root path only
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := h.parseTemplate("dashboard.html")
	if err != nil {
		slog.Error("failed to parse dashboard template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		ActivePage: "dashboard",
		HasCharts:  true,
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("failed to render dashboard", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// History renders the history page
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	tmpl, err := h.parseTemplate("history.html")
	if err != nil {
		slog.Error("failed to parse history template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		ActivePage: "history",
		HasCharts:  true,
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("failed to render history", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Records renders the records page
func (h *Handler) Records(w http.ResponseWriter, r *http.Request) {
	records, err := h.weatherService.GetRecords(r.Context())
	if err != nil {
		slog.Error("failed to get records", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err := h.parseTemplate("records.html")
	if err != nil {
		slog.Error("failed to parse records template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		ActivePage: "records",
		Data:       records,
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("failed to render records", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Insights renders the interactive station-observation archive.
func (h *Handler) Insights(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	archive, err := h.weatherService.GetArchive(
		r.Context(),
		query.Get("period"), query.Get("metric"), query.Get("month"), query.Get("season"),
		query.Get("year"), query.Get("from"), query.Get("to"),
		query.Get("search_field"), query.Get("search_comparison"), query.Get("search_threshold"),
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidArchivePeriod) || errors.Is(err, service.ErrInvalidArchiveRange) || errors.Is(err, service.ErrInvalidArchiveSearch) {
			http.Error(w, "Некорректные параметры архива", http.StatusBadRequest)
			return
		}
		slog.Error("failed to get weather archive", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err := h.parseTemplate("insights.html")
	if err != nil {
		slog.Error("failed to parse archive template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, PageData{ActivePage: "insights", Data: archive}); err != nil {
		slog.Error("failed to render weather archive", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// InsightsReport renders a public share-friendly weather report.
func (h *Handler) InsightsReport(w http.ResponseWriter, r *http.Request) {
	insights, err := h.getInsightsFromRequest(r)
	if err != nil {
		h.handleInsightsError(w, err, "failed to get weather insights report")
		return
	}

	tmpl, err := h.parseTemplate("insights_report.html")
	if err != nil {
		slog.Error("failed to parse insights report template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		ActivePage: "insights",
		Data:       insights,
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("failed to render insights report", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// InsightsStory renders a vertical 9:16 card suitable for screenshotting into stories.
func (h *Handler) InsightsStory(w http.ResponseWriter, r *http.Request) {
	insights, err := h.getInsightsFromRequest(r)
	if err != nil {
		h.handleInsightsError(w, err, "failed to get weather insights story")
		return
	}

	tmpl, err := h.parseStandaloneTemplate("insights_story.html")
	if err != nil {
		slog.Error("failed to parse insights story template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, PageData{ActivePage: "insights", Data: insights}); err != nil {
		slog.Error("failed to render insights story", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// InsightsText renders a ready-to-copy Telegram/social post text.
func (h *Handler) InsightsText(w http.ResponseWriter, r *http.Request) {
	insights, err := h.getInsightsFromRequest(r)
	if err != nil {
		h.handleInsightsError(w, err, "failed to get weather insights text")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(buildInsightsShareText(insights)))
}

func (h *Handler) handleInsightsError(w http.ResponseWriter, err error, message string) {
	if errors.Is(err, service.ErrInvalidInsightSeason) {
		http.Error(w, "Bad season format, expected YYYY-season", http.StatusBadRequest)
		return
	}
	if errors.Is(err, errInvalidInsightMonth) {
		http.Error(w, "Bad month format, expected YYYY-MM", http.StatusBadRequest)
		return
	}
	slog.Error(message, "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func buildInsightsShareText(insights *models.WeatherInsightsPage) string {
	period := insights.SelectedMonthLabel
	if insights.IsSeason {
		period = insights.SelectedSeasonLabel
	}
	reportURL := "https://meteo.armavir.ru/insights/report?month=" + insights.SelectedMonthParam
	if insights.IsSeason {
		reportURL = "https://meteo.armavir.ru/insights/report?period=season&season=" + insights.SelectedSeasonParam
	}

	lines := []string{
		fmt.Sprintf("%s в Армавире %s", period, insights.Season.Icon),
		"",
		insights.MainInsight.Text,
		"",
		fmt.Sprintf("🌧 Осадки: %.1f мм", insights.CurrentMonth.RainTotal),
		fmt.Sprintf("🌡 Средняя температура: %.1f°C", insights.CurrentMonth.AvgTemp),
		fmt.Sprintf("☔ Дождливых дней: %d", insights.CurrentMonth.RainDays),
		fmt.Sprintf("🚶 Комфортных дней: %d из %d", insights.CurrentMonth.ComfortableDays, insights.CurrentMonth.DaysWithData),
		fmt.Sprintf("🧬 Характер: %s %s", insights.DominantDayType.Icon, insights.DominantDayType.Label),
	}
	if insights.CurrentMonth.MaxRainDay != nil {
		lines = append(lines, fmt.Sprintf("🏆 Главный дождь: %s — %.1f мм", insights.CurrentMonth.MaxRainDay.Date.Format("02.01"), insights.CurrentMonth.MaxRainDay.Value))
	}
	if insights.SameMonthBenchmark.Available {
		lines = append(lines, fmt.Sprintf("📚 Архив: %s", insights.SameMonthBenchmark.Verdict))
	}
	lines = append(lines, "", "Полный отчёт:", reportURL)
	return strings.Join(lines, "\n")
}

func (h *Handler) getInsightsFromRequest(r *http.Request) (*models.WeatherInsightsPage, error) {
	if r.URL.Query().Get("period") == "season" {
		insights, err := h.weatherService.GetInsightsForSeason(r.Context(), r.URL.Query().Get("season"))
		if errors.Is(err, service.ErrInvalidInsightSeason) {
			return nil, err
		}
		return insights, err
	}

	var selectedMonth time.Time
	if monthParam := r.URL.Query().Get("month"); monthParam != "" {
		parsed, err := time.Parse("2006-01", monthParam)
		if err != nil {
			return nil, errInvalidInsightMonth
		}
		selectedMonth = parsed
	}
	return h.weatherService.GetInsightsForMonth(r.Context(), selectedMonth)
}

// Help renders the help/reference page
func (h *Handler) Help(w http.ResponseWriter, r *http.Request) {
	tmpl, err := h.parseTemplate("help.html")
	if err != nil {
		slog.Error("failed to parse help template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get current weather for power status
	current, err := h.weatherService.GetCurrent(r.Context())
	if err != nil {
		slog.Warn("failed to get current weather for help page", "error", err)
		current = nil
	}

	data := PageData{
		ActivePage: "help",
		Current:    current,
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("failed to render help", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Gallery renders the photo gallery page
func (h *Handler) Gallery(w http.ResponseWriter, r *http.Request) {
	// Получаем видимые фотографии (лимит 50)
	photos, err := h.photoRepo.GetVisible(r.Context(), 50, 0)
	if err != nil {
		slog.Error("failed to get photos", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err := h.parseTemplate("gallery.html")
	if err != nil {
		slog.Error("failed to parse gallery template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := PageData{
		ActivePage: "gallery",
		Data:       photos,
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("failed to render gallery", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
