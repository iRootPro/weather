package web

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBaseTemplateRendersAccessibleNavigation(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parseTemplate("dashboard.html")
	if err != nil {
		t.Fatalf("parseTemplate() error = %v", err)
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, PageData{ActivePage: "dashboard"}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{
		`aria-label="Главная"`,
		`aria-label="История"`,
		`aria-label="Рекорды"`,
		`aria-label="Архив"`,
		`aria-label="Галерея"`,
		`aria-label="Справка"`,
		`aria-label="Переключить цветовую тему"`,
		`aria-label="Скрыть предложение Telegram-бота"`,
		`aria-current="page"`,
		`a:focus-visible`,
		`.dark a:focus-visible`,
		`outline-color: #93c5fd`,
	} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered template is missing %s", expected)
		}
	}
}

func TestChartTemplatesRenderAccessibleDataControls(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	pages := map[string][]string{
		"dashboard.html": {"tempChart", "humidityChart", "pressureChart", "windChart", "solarChart", "rainChart"},
		"history.html":   {"historyTempChart", "historyHumidityChart", "historyPressureChart", "historyWindChart", "historyRainChart", "historySolarChart"},
	}
	for page, chartIDs := range pages {
		t.Run(page, func(t *testing.T) {
			tmpl, err := h.parseTemplate(page)
			if err != nil {
				t.Fatalf("parseTemplate() error = %v", err)
			}

			var output bytes.Buffer
			if err := tmpl.Execute(&output, PageData{}); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			for _, chartID := range chartIDs {
				for _, suffix := range []string{"summary", "details", "table"} {
					expected := `id="` + chartID + `-` + suffix + `"`
					if !bytes.Contains(output.Bytes(), []byte(expected)) {
						t.Errorf("rendered template is missing %s", expected)
					}
				}
				if !bytes.Contains(output.Bytes(), []byte(`aria-describedby="`+chartID+`-summary"`)) {
					t.Errorf("rendered template is missing an accessible description for %s", chartID)
				}
				if !bytes.Contains(output.Bytes(), []byte(`id="`+chartID+`" role="img" aria-label=`)) {
					t.Errorf("rendered template is missing an accessible name for %s", chartID)
				}
			}
		})
	}
}

func TestChartTemplatesUseCurrentChartsScriptVersion(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	for _, page := range []string{"dashboard.html", "history.html"} {
		t.Run(page, func(t *testing.T) {
			tmpl, err := h.parseTemplate(page)
			if err != nil {
				t.Fatalf("parseTemplate() error = %v", err)
			}

			var output bytes.Buffer
			if err := tmpl.Execute(&output, PageData{}); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			const currentChartsScript = `<script src="/static/js/charts.js?v=8"></script>`
			if bytes.Count(output.Bytes(), []byte(currentChartsScript)) != 1 {
				t.Fatalf("rendered template must contain exactly one current charts script reference: %s", currentChartsScript)
			}
			if bytes.Contains(output.Bytes(), []byte(`/static/js/charts.js?v=7`)) {
				t.Fatal("rendered template still references the previous charts script version")
			}
		})
	}
}

func TestDashboardTemplatePrioritizesWeatherBeforeTelegramPromotion(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parseTemplate("dashboard.html")
	if err != nil {
		t.Fatalf("parseTemplate() error = %v", err)
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, PageData{ActivePage: "dashboard"}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{
		`href="#charts-24h"`,
		`id="weather-events"`,
		`id="charts-24h"`,
		`id="telegram-bot-card" class="order-9`,
	} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered dashboard is missing %s", expected)
		}
	}

	waterIndex := bytes.Index(output.Bytes(), []byte(`id="water-level"`))
	telegramIndex := bytes.Index(output.Bytes(), []byte(`id="telegram-bot-card"`))
	sunIndex := bytes.Index(output.Bytes(), []byte(`id="sun-times"`))
	eventsIndex := bytes.Index(output.Bytes(), []byte(`id="weather-events"`))
	if waterIndex >= telegramIndex || telegramIndex >= sunIndex || sunIndex >= eventsIndex {
		t.Fatal("desktop dashboard order changed")
	}
}

func TestCurrentWeatherTemplateShowsMeasurementAndRefreshTimes(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parsePartial("current_weather.html")
	if err != nil {
		t.Fatalf("parsePartial() error = %v", err)
	}

	var output bytes.Buffer
	data := map[string]interface{}{
		"ObservationTime": "12:00",
		"UpdatedAt":       "12:01",
		"Geomagnetic":     map[string]bool{"HasData": false},
	}
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !bytes.Contains(output.Bytes(), []byte("Измерено 12:00 · обновлено 12:01")) {
		t.Fatal("current weather timestamp is missing or incomplete")
	}
}

func TestForecastTemplateUsesMobileScrollSnap(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parsePartial("forecast.html")
	if err != nil {
		t.Fatalf("parsePartial() error = %v", err)
	}

	type forecastCard struct {
		Label                    string
		Icon                     string
		TempMain                 string
		TempSecondary            string
		WeatherDescription       string
		PrecipitationProbability int16
		HasPrecipitation         bool
	}
	data := struct {
		Cards      []forecastCard
		NoForecast bool
	}{Cards: []forecastCard{{Label: "12:00", Icon: "☀️", TempMain: "20°"}}}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{"snap-x snap-mandatory", `aria-label="Периоды прогноза"`, "w-40 shrink-0 snap-start", "Проведите влево"} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered forecast is missing %s", expected)
		}
	}
}

func TestWaterLevelTemplateUsesMobileSummaryAndDetails(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parsePartial("water_level.html")
	if err != nil {
		t.Fatalf("parsePartial() error = %v", err)
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, WaterLevelCardData{Upstream: []WaterLevelMiniData{{}}}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{"class=\"sm:hidden\"", "Подробный график и пороги", "Сравнить посты выше Армавира", "hidden sm:block"} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered water level widget is missing %s", expected)
		}
	}
}

func TestHistoryTemplateHasQuickPeriodsAndChartAccordions(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parseTemplate("history.html")
	if err != nil {
		t.Fatalf("parseTemplate() error = %v", err)
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, PageData{ActivePage: "history"}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{
		`data-history-period="24h"`, `data-history-period="7d"`, `data-history-period="30d"`, `data-history-period="month"`,
		`id="history-status"`, `id="history-retry"`,
		`data-history-chart="temp"`, `data-history-chart="humidity"`, `data-history-chart="pressure"`,
		`data-history-chart="wind"`, `data-history-chart="rain"`, `data-history-chart="solar"`,
	} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered history is missing %s", expected)
		}
	}
}
