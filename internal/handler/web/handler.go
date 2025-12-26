package web

import (
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/service"
)

type Handler struct {
	templatesDir   string
	weatherService *service.WeatherService
	sunService     *service.SunService
	moonService    *service.MoonService
}

func NewHandler(templatesDir string, weatherService *service.WeatherService, sunService *service.SunService, moonService *service.MoonService) (*Handler, error) {
	return &Handler{
		templatesDir:   templatesDir,
		weatherService: weatherService,
		sunService:     sunService,
		moonService:    moonService,
	}, nil
}

var templateFuncs = template.FuncMap{
	"mul": func(a, b float64) float64 {
		return a * b
	},
	"sub": func(a, b float64) float64 {
		return a - b
	},
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

type PageData struct {
	ActivePage string
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
