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
)

// NewDashboardDemo renders both production dashboard widgets with synthetic data.
// It is loopback-only through cmd/weather-demo and exists for visual regression checks.
func NewDashboardDemo(templatesDir string) (http.Handler, error) {
	h := &Handler{templatesDir: templatesDir}
	currentPartial, err := h.parsePartial("current_weather.html")
	if err != nil {
		return nil, err
	}
	forecastPartial, err := h.parsePartial("forecast.html")
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
	page, err := template.New("dashboard-demo").Parse(dashboardDemoHTML)
	if err != nil {
		return nil, err
	}

	render := func(now time.Time, name, currentScenario, forecastScenario string) ([]byte, error) {
		var output bytes.Buffer
		var renderErr error
		if name == "current" {
			selected := attentionDemoScenarios[len(attentionDemoScenarios)-1]
			for _, scenario := range attentionDemoScenarios {
				if scenario.ID == currentScenario {
					selected = scenario
					break
				}
			}
			renderErr = currentPartial.Execute(&output, demoCurrentWeatherData(now, selected))
		} else {
			renderErr = forecastPartial.Execute(&output, demoForecastData(now, forecastScenario))
		}
		return output.Bytes(), renderErr
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		widget := r.URL.Query().Get("widget")
		currentScenario := r.URL.Query().Get("current")
		forecastScenario := r.URL.Query().Get("forecast")
		if forecastScenario == "" {
			forecastScenario = "stale"
		}
		if widget == "current" || widget == "forecast" {
			output, renderErr := render(time.Now(), widget, currentScenario, forecastScenario)
			if renderErr != nil {
				http.Error(w, "Demo render failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(output)
			return
		}
		current, renderErr := render(time.Now(), "current", currentScenario, forecastScenario)
		if renderErr != nil {
			http.Error(w, "Demo render failed", http.StatusInternalServerError)
			return
		}
		forecast, renderErr := render(time.Now(), "forecast", currentScenario, forecastScenario)
		if renderErr != nil {
			http.Error(w, "Demo render failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, map[string]any{
			"CSS":      template.CSS(css),
			"Current":  template.HTML(current),
			"Forecast": template.HTML(forecast),
			"Theme":    r.URL.Query().Get("theme"),
		}); err != nil {
			http.Error(w, "Demo render failed", http.StatusInternalServerError)
		}
	}), nil
}

const dashboardDemoHTML = `<!doctype html><html lang="ru" {{if eq .Theme "dark"}}class="dark"{{end}}><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Демо дашборда</title><script src="/static/js/vendor/tailwind.min.js"></script><script>tailwind.config={darkMode:'class'}</script><script src="/static/js/vendor/htmx.min.js"></script><style>{{.CSS}}</style></head><body class="bg-gray-50 text-gray-900 dark:bg-gray-950 dark:text-white"><main class="mx-auto max-w-7xl p-4 sm:p-8"><div class="mb-5 flex flex-wrap items-center gap-3"><h1 class="text-xl font-bold">Дашборд · синтетические данные</h1><a class="underline" href="?current=normal&amp;forecast=fresh">Обычная погода</a><a class="underline" href="?current=six&amp;forecast=stale">Все шесть сигналов</a><a class="underline" href="?theme=dark">Тёмная тема</a><a class="underline" href="?theme=light">Светлая тема</a><button class="underline" type="button" hx-get="?widget=current" hx-target="#current-weather" hx-swap="innerHTML">Обновить «Сейчас»</button><button class="underline" type="button" hx-get="?widget=forecast" hx-target="#forecast" hx-swap="innerHTML">Обновить прогноз</button></div><div class="ui-page flex flex-col gap-6 lg:grid lg:grid-cols-12 lg:items-stretch lg:gap-6"><div id="current-weather" class="order-1 lg:col-span-8 lg:flex">{{.Current}}</div><aside class="contents order-2 lg:col-span-4 lg:flex lg:h-full lg:flex-col lg:gap-6" aria-label="Прогноз"><div id="forecast" class="order-3 lg:order-1 lg:flex lg:h-full">{{.Forecast}}</div></aside></div></main></body></html>`
