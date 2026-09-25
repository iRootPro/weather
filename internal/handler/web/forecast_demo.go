package web

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

// NewForecastDemo renders the production forecast partial with local synthetic data.
// It is only exposed by cmd/weather-demo.
func NewForecastDemo(templatesDir string) (http.Handler, error) {
	h := &Handler{templatesDir: templatesDir}
	partial, err := h.parsePartial("forecast.html")
	if err != nil {
		return nil, err
	}
	base, err := os.ReadFile(filepath.Join(templatesDir, "base.html"))
	if err != nil {
		return nil, err
	}
	_, css, ok := strings.Cut(string(base), "<style>")
	if !ok {
		return nil, fmt.Errorf("base template has no inline styles")
	}
	css, _, ok = strings.Cut(css, "</style>")
	if !ok {
		return nil, fmt.Errorf("base template has unterminated styles")
	}
	page, err := template.New("forecast-demo").Parse(`<!doctype html><html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Демо прогноза</title><script src="/static/js/vendor/tailwind.min.js"></script><script>tailwind.config={darkMode:'class'}</script><style>{{.CSS}}</style></head><body class="bg-gray-50 text-gray-900 dark:bg-gray-950 dark:text-white"><main class="mx-auto max-w-5xl p-4 sm:p-8"><div class="mb-5 flex flex-wrap items-center gap-3"><h1 class="text-xl font-bold">Демо прогноза</h1><a class="underline" href="?scenario=fresh">Свежий</a><a class="underline" href="?scenario=stale">Устаревший</a><a class="underline" href="?scenario=unavailable">Недоступен</a><button class="underline" onclick="document.documentElement.classList.toggle('dark')">Тёмная тема</button></div>{{.Widget}}</main></body></html>`)
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		data := demoForecastData(now, r.URL.Query().Get("scenario"))
		var widget bytes.Buffer
		if err := partial.Execute(&widget, data); err != nil {
			http.Error(w, "Demo render failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, map[string]any{"CSS": template.CSS(css), "Widget": template.HTML(widget.String())}); err != nil {
			http.Error(w, "Demo render failed", http.StatusInternalServerError)
		}
	}), nil
}

func demoForecastData(now time.Time, scenario string) forecastWidgetData {
	if scenario == "unavailable" {
		return buildForecastWidgetData(now, nil, nil)
	}
	fetchedAt := now.Add(-time.Hour)
	if scenario == "stale" {
		fetchedAt = now.Add(-3 * time.Hour)
	}
	hourly := []models.HourlyForecast{
		{Time: now.Add(time.Hour), Temperature: 20, FeelsLike: 16, Precipitation: 0.4, PrecipitationProbability: 60, HasTemperature: true, HasFeelsLike: true, HasPrecipitation: true, HasPrecipitationProbability: true, FetchedAt: fetchedAt},
		{Time: now.Add(4 * time.Hour), Temperature: 18, WindSpeed: 11, WindGusts: 16, HasTemperature: true, HasWindSpeed: true, HasWindGusts: true, FetchedAt: fetchedAt},
	}
	daily := []models.DailyForecast{{Date: now.AddDate(0, 0, 1), TemperatureMin: 11, TemperatureMax: 19, PrecipitationSum: 1.2, PrecipitationProbability: 70, WindSpeedMax: 9, WindGustsMax: 14, UVIndexMax: 6, HasPrecipitationSum: true, HasPrecipitationProbability: true, HasWindSpeedMax: true, HasWindGustsMax: true, HasUVIndexMax: true, FetchedAt: fetchedAt}}
	return buildForecastWidgetData(now, hourly, daily)
}
