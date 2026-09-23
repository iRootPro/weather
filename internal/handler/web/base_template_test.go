package web

import (
	"bytes"
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
)

type widgetWeatherRepository struct {
	repository.WeatherRepository
	current *models.WeatherData
	hourAgo *models.WeatherData
	daily   *repository.DailyMinMax
}

func (r widgetWeatherRepository) GetLatest(context.Context) (*models.WeatherData, error) {
	return r.current, nil
}

func (r widgetWeatherRepository) GetDataNearTime(context.Context, time.Time) (*models.WeatherData, error) {
	return r.hourAgo, nil
}

func (r widgetWeatherRepository) GetDailyMinMax(context.Context) (*repository.DailyMinMax, error) {
	return r.daily, nil
}

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
		`family=Golos+Text`,
		`family=IBM+Plex+Mono`,
		`display=swap`,
		`--ui-font-sans`,
		`--ui-font-mono`,
		`--ui-surface`,
		`--ui-text-muted`,
		`--ui-on-action`,
		`.ui-button-primary`,
		`.ui-button-secondary`,
		`.ui-field`,
		`.ui-field[type="date"]`,
		`.ui-field[type="month"]`,
		`-webkit-appearance: none`,
		`.ui-field-label`,
		`.ui-form-item`,
		`box-sizing: border-box`,
		`inline-size: 100%`,
		`max-inline-size: 100%`,
		`@media (max-width: 767px)`,
		`min-inline-size: 0`,
		`padding: 0.5rem 0.75rem`,
		`color: var(--ui-on-action)`,
		`class="min-h-screen transition-colors duration-200"`,
		`.ui-page-header`,
		`.ui-section-heading`,
		`.ui-card-heading`,
		`.ui-kicker`,
		`.ui-tabular`,
		`.ui-body`,
		`.ui-text-muted`,
		`.ui-caption`,
		`main .text-gray-400`,
		`.dark main .dark\:text-gray-500`,
		`.ui-mobile-nav`,
		`--ui-mobile-nav-height`,
		`.ui-mobile-nav-clearance`,
		`padding-bottom: env(safe-area-inset-bottom)`,
		`grid-cols-5`,
		`min-h-16`,
		`aria-label="Ещё"`,
		`summary:focus-visible`,
		`prefers-reduced-motion: reduce`,
		`.ui-chart-panel`,
		`.ui-chart-plot`,
		`.ui-chart-plot-with-legend`,
		`.ui-chart-data`,
		`aria-label="Главная"`,
		`aria-label="История"`,
		`aria-label="Рекорды"`,
		`aria-label="Архив"`,
		`aria-label="Галерея"`,
		`aria-label="Справка"`,
		`aria-label="Переключить цветовую тему"`,
		`id="telegram-bot-promo"`,
		`Метеостанция Армавир`,
		`aria-current="page"`,
		`a:focus-visible`,
		`.dark a:focus-visible`,
		`outline-color: var(--ui-focus)`,
	} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered template is missing %s", expected)
		}
	}
}

func TestGalleryModalRendersAboveMobileNavigation(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	contents, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates", "gallery.html"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	for _, expected := range []string{
		`id="photoModal" class="ui-overlay`,
		`role="dialog"`,
		`aria-modal="true"`,
		`event.key === 'Escape'`,
		`photoModalTrigger.focus()`,
		`event.key !== 'Tab'`,
		`focusable.at(-1)`,
	} {
		if !bytes.Contains(contents, []byte(expected)) {
			t.Errorf("gallery modal is missing %s", expected)
		}
	}
}

func TestDataHeavyPagesUseSharedDesignSystemRoles(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	templatesDir := filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")
	for _, name := range []string{
		"records.html",
		"detail/temperature.html", "detail/humidity.html", "detail/pressure.html", "detail/wind.html",
		"detail/rain.html", "detail/solar.html", "detail/geomagnetic.html", "detail/water_level.html",
	} {
		t.Run(name, func(t *testing.T) {
			contents, err := os.ReadFile(filepath.Join(templatesDir, name))
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if !bytes.Contains(contents, []byte("ui-surface")) {
				t.Error("page must use the shared surface role")
			}
			if !bytes.Contains(contents, []byte("ui-metric-card")) {
				t.Error("page must use the shared metric card role")
			}
		})
	}

	for _, name := range []string{"detail/temperature.html", "detail/humidity.html", "detail/pressure.html", "detail/wind.html", "detail/rain.html", "detail/solar.html"} {
		contents, err := os.ReadFile(filepath.Join(templatesDir, name))
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", name, err)
		}
		for _, expected := range []string{"ui-button-primary", "ui-button-secondary", `aria-pressed="true"`, `aria-label="Период графика"`} {
			if !bytes.Contains(contents, []byte(expected)) {
				t.Errorf("%s is missing shared chart control %s", name, expected)
			}
		}
	}

	for _, name := range []string{"detail/geomagnetic.html", "detail/water_level.html"} {
		contents, err := os.ReadFile(filepath.Join(templatesDir, name))
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", name, err)
		}
		if !bytes.Contains(contents, []byte("ui-status-surface")) {
			t.Errorf("%s must use the shared status surface role", name)
		}
	}
}

func TestArchiveReportAndHelpUseSharedRoles(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	templatesDir := filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")
	assertContains := func(name string, expected []string) {
		t.Helper()
		contents, err := os.ReadFile(filepath.Join(templatesDir, name))
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", name, err)
		}
		for _, value := range expected {
			if !bytes.Contains(contents, []byte(value)) {
				t.Errorf("%s is missing %s", name, value)
			}
		}
	}

	assertContains("insights.html", []string{"ui-page", "ui-field", "ui-button-primary", "ui-button-secondary"})
	assertContains("insights_report.html", []string{"ui-page", "ui-surface", "ui-button-primary", "ui-button-secondary"})
	assertContains("help.html", []string{`aria-label="Оглавление справки"`, `href="#calculations"`, "<details", "Формулы и пороги расчётных показателей", `id="station"`})

	report, err := os.ReadFile(filepath.Join(templatesDir, "insights_report.html"))
	if err != nil {
		t.Fatalf("ReadFile(insights_report.html) error = %v", err)
	}
	for _, emoji := range []string{"🌧️", "🌡️", "🚶", "🧬", ".MainInsight.Icon", ".DominantDayType.Icon"} {
		if bytes.Contains(report, []byte(emoji)) {
			t.Errorf("public report must not use decorative KPI icon %s", emoji)
		}
	}
}

func TestMainTemplatesHaveExactlyOnePageHeading(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	templatesDir := filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")
	templates := []string{
		"dashboard.html",
		"history.html",
		"records.html",
		"gallery.html",
		"help.html",
		"insights.html",
		"insights_report.html",
		"insights_story.html",
		"detail/temperature.html",
		"detail/humidity.html",
		"detail/pressure.html",
		"detail/wind.html",
		"detail/rain.html",
		"detail/solar.html",
		"detail/geomagnetic.html",
		"detail/water_level.html",
	}

	baseContents, err := os.ReadFile(filepath.Join(templatesDir, "base.html"))
	if err != nil {
		t.Fatalf("ReadFile(base.html) error = %v", err)
	}
	if bytes.Contains(baseContents, []byte("<h1")) {
		t.Fatal("global header must not contain a page heading")
	}

	headingPattern := regexp.MustCompile(`<h([1-6])(?:\s|>)`)

	for _, name := range templates {
		t.Run(name, func(t *testing.T) {
			contents, err := os.ReadFile(filepath.Join(templatesDir, name))
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if count := strings.Count(string(contents), "<h1"); count != 1 {
				t.Errorf("page heading count = %d, want 1", count)
			}

			previousLevel := 0
			for _, match := range headingPattern.FindAllSubmatch(contents, -1) {
				level := int(match[1][0] - '0')
				if previousLevel != 0 && level > previousLevel+1 {
					t.Errorf("heading level jumps from h%d to h%d", previousLevel, level)
				}
				previousLevel = level
			}
		})
	}
}

func TestConditionalDetailTemplatesKeepPageHeading(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	for _, name := range []string{"detail/geomagnetic.html", "detail/water_level.html"} {
		t.Run(name, func(t *testing.T) {
			tmpl, err := h.parseTemplate(name)
			if err != nil {
				t.Fatalf("parseTemplate() error = %v", err)
			}

			for _, hasData := range []bool{false, true} {
				t.Run(map[bool]string{false: "empty", true: "data"}[hasData], func(t *testing.T) {
					var output bytes.Buffer
					data := map[string]any{
						"Card":      map[string]any{"HasData": hasData},
						"ChartJSON": "[]",
					}
					if err := tmpl.Execute(&output, PageData{Data: data}); err != nil {
						t.Fatalf("Execute() error = %v", err)
					}
					if count := bytes.Count(output.Bytes(), []byte("<h1")); count != 1 {
						t.Errorf("rendered page heading count = %d, want 1", count)
					}
				})
			}
		})
	}
}

func TestWaterLevelDetailDoesNotRenderMissingThresholdsAsZero(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parseTemplate("detail/water_level.html")
	if err != nil {
		t.Fatalf("parseTemplate() error = %v", err)
	}

	var output bytes.Buffer
	data := PageData{Data: map[string]any{
		"Card":      WaterLevelCardData{HasData: true},
		"Gauge":     &models.HydroGauge{},
		"ChartJSON": "[]",
	}}
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if bytes.Contains(output.Bytes(), []byte("0.000 м")) {
		t.Fatal("missing water-level thresholds must not be rendered as 0.000 м")
	}
	if count := bytes.Count(output.Bytes(), []byte("Порог не указан источником")); count != 2 {
		t.Fatalf("missing threshold message count = %d, want 2", count)
	}
}

func TestWaterLevelDetailRendersProvidedThresholds(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	prevention := float32(162.5)
	danger := float32(163.75)
	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parseTemplate("detail/water_level.html")
	if err != nil {
		t.Fatalf("parseTemplate() error = %v", err)
	}

	var output bytes.Buffer
	data := PageData{Data: map[string]any{
		"Card": WaterLevelCardData{HasData: true},
		"Gauge": &models.HydroGauge{
			FloodingPreventionBM: &prevention,
			FloodingDangerBSM:    &danger,
		},
		"ChartJSON": "[]",
	}}
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{"162.500 м", "163.750 м"} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("provided water-level threshold is missing %s", expected)
		}
	}
	if bytes.Contains(output.Bytes(), []byte("Порог не указан источником")) {
		t.Fatal("provided water-level thresholds must not render the missing-source message")
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
			const currentChartsScript = `<script src="/static/js/charts.js?v=9"></script>`
			if bytes.Count(output.Bytes(), []byte(currentChartsScript)) != 1 {
				t.Fatalf("rendered template must contain exactly one current charts script reference: %s", currentChartsScript)
			}
			if bytes.Contains(output.Bytes(), []byte(`/static/js/charts.js?v=8`)) {
				t.Fatal("rendered template still references the previous charts script version")
			}
		})
	}
}

func TestChartsScriptRendersMobileDataCards(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	script, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "..", "web", "static", "js", "charts.js"))
	if err != nil {
		t.Fatalf("ReadFile(charts.js) error = %v", err)
	}

	for _, expected := range []string{
		"function renderChartDataCards",
		"window.matchMedia('(max-width: 639px)')",
		"grid-cols-[minmax(0,1fr)_auto]",
		"details[id$=\"-details\"][open]",
	} {
		if !bytes.Contains(script, []byte(expected)) {
			t.Errorf("charts.js is missing %s", expected)
		}
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
		`ui-chart-panel`,
		`aria-pressed="true"`,
		`aria-label="Интервал графиков"`,
		`id="weather-events"`,
		`id="charts-24h"`,
		`ui-chart-plot`, `ui-chart-plot-with-legend`, `ui-chart-data`,
		`id="sun-times"`,
		`id="telegram-bot-promo"`,
		`syncChartVisibility`,
		`<h1 class="sr-only">Погода в Армавире</h1>`,
		`<h2 class="ui-section-heading sr-only sm:not-sr-only px-4 pt-4 sm:px-6 sm:pt-6">Графики за 24 часа</h2>`,
	} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered dashboard is missing %s", expected)
		}
	}

	eventsIndex := bytes.Index(output.Bytes(), []byte(`id="weather-events"`))
	forecastIndex := bytes.Index(output.Bytes(), []byte(`id="forecast"`))
	waterIndex := bytes.Index(output.Bytes(), []byte(`id="water-level"`))
	sunIndex := bytes.Index(output.Bytes(), []byte(`id="sun-times"`))
	chartsIndex := bytes.Index(output.Bytes(), []byte(`id="charts-24h"`))
	telegramIndex := bytes.Index(output.Bytes(), []byte(`id="telegram-bot-promo"`))
	if eventsIndex >= forecastIndex || forecastIndex >= waterIndex || waterIndex >= sunIndex || sunIndex >= chartsIndex || chartsIndex >= telegramIndex {
		t.Fatal("dashboard content order changed")
	}
	if bytes.Contains(output.Bytes(), []byte(`id="daily-stats"`)) || bytes.Contains(output.Bytes(), []byte(`href="#charts-24h"`)) {
		t.Fatal("dashboard must not contain duplicated stats or a chart jump link")
	}
	if bytes.Contains(output.Bytes(), []byte("Погода сейчас")) || bytes.Contains(output.Bytes(), []byte(`class="ui-kicker">Метеостанция Армавир`)) {
		t.Fatal("dashboard must not repeat the application name or current-weather label in its page header")
	}
	if bytes.Contains(output.Bytes(), []byte(`<h1 class="ui-page-header">`)) {
		t.Fatal("dashboard h1 must stay visually hidden because the application header already provides the location context")
	}
	chartHeadingIndex := bytes.Index(output.Bytes(), []byte(`<h2 class="ui-section-heading sr-only sm:not-sr-only px-4 pt-4 sm:px-6 sm:pt-6">Графики за 24 часа</h2>`))
	chartDetailsIndex := bytes.Index(output.Bytes(), []byte(`<details id="charts-24h">`))
	if chartHeadingIndex < 0 || chartHeadingIndex > chartDetailsIndex {
		t.Fatal("chart heading must stay outside the collapsed mobile disclosure")
	}
}

func TestDashboardWidgetTemplatesUseConsistentHeadingRoles(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	partialsDir := filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates", "partials")
	for name, expected := range map[string]string{
		"current_weather.html": `<h2 class="ui-section-heading">Сейчас</h2>`,
		"forecast.html":        `<h2 class="ui-section-heading mb-4">Прогноз погоды</h2>`,
		"sun_times.html":       `<h2 class="ui-section-heading">Солнце и Луна</h2>`,
		"water_level.html":     `<h2 class="ui-section-heading">Уровень Кубани</h2>`,
		"weather_events.html":  `<h2 class="ui-card-heading">Погодные события (24 часа)</h2>`,
	} {
		t.Run(name, func(t *testing.T) {
			contents, err := os.ReadFile(filepath.Join(partialsDir, name))
			if err != nil {
				t.Fatalf("ReadFile(%s) error = %v", name, err)
			}
			if !bytes.Contains(contents, []byte(expected)) {
				t.Errorf("widget heading must use its shared role: %s", expected)
			}
		})
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
		"ObservationTime":       "12:00",
		"UpdatedAt":             "12:01",
		"HasTempHourlyData":     true,
		"HasHumidityHourlyData": true,
		"HasPressureHourlyData": true,
		"HasTempDailyData":      true,
		"HasHumidityDailyData":  true,
		"HasPressureDailyData":  true,
		"HasWindMax":            true,
		"HasWindGustMax":        true,
		"HasDewPoint":           true,
		"HasWindGust":           true,
		"HasRainMonthly":        true,
		"TempChange":            float32(0),
		"HumidityChange":        int16(0),
		"PressureChange":        float32(0),
		"Geomagnetic":           map[string]bool{"HasData": false},
	}
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !bytes.Contains(output.Bytes(), []byte("Измерено 12:00 · обновлено 12:01")) {
		t.Fatal("current weather timestamp is missing or incomplete")
	}
	if !bytes.Contains(output.Bytes(), []byte(`class="ui-tabular text-xs`)) {
		t.Fatal("current weather timestamp must use the tabular typography role")
	}
	if bytes.Contains(output.Bytes(), []byte("hover:scale-105")) {
		t.Fatal("metric cards must not use scale as their primary hover feedback")
	}
	if !bytes.Contains(output.Bytes(), []byte("ui-metric-card")) {
		t.Fatal("current weather values must use the shared metric card role")
	}
	for _, expected := range []string{`p-3 text-center group sm:p-4`, `sm:text-3xl`, `sm:hidden">UV`, `mt-1 text-xs text-gray-400 dark:text-gray-500`} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("current weather is missing detailed mobile treatment %s", expected)
		}
	}
	if !bytes.Contains(output.Bytes(), []byte("0.0 мм рт. ст./ч")) {
		t.Fatal("pressure change must include its unit")
	}
}

func TestCurrentWeatherTemplateOmitsSecondaryMetricsWithoutSourceData(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parsePartial("current_weather.html")
	if err != nil {
		t.Fatalf("parsePartial() error = %v", err)
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, map[string]interface{}{
		"ObservationTime": "12:00",
		"UpdatedAt":       "12:01",
		"Geomagnetic":     map[string]bool{"HasData": false},
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, unexpected := range []string{"→\n                0.0°/ч", "0.0° / 0.0°", "→\n                0%/ч", "0% / 0%", "0 мм рт. ст./ч", "макс: 0.0 м/с", "макс порыв: 0.0 м/с"} {
		if bytes.Contains(output.Bytes(), []byte(unexpected)) {
			t.Errorf("secondary metric without source data must be omitted, got %q", unexpected)
		}
	}
}

func TestCurrentWeatherWidgetOmitsPartialSecondaryMetrics(t *testing.T) {
	temp, previousTemp := float32(20), float32(19)
	humidity := int16(60)
	pressure, windSpeed, rainDaily := float32(740), float32(3), float32(0)
	tempMin, windMax := float32(12), float32(6)

	h := &Handler{
		templatesDir: filepath.Join("..", "..", "web", "templates"),
		weatherService: service.NewWeatherService(widgetWeatherRepository{
			current: &models.WeatherData{
				Time:             time.Now(),
				TempOutdoor:      &temp,
				HumidityOutdoor:  &humidity,
				PressureRelative: &pressure,
				WindSpeed:        &windSpeed,
				RainDaily:        &rainDaily,
			},
			hourAgo: &models.WeatherData{TempOutdoor: &previousTemp},
			daily:   &repository.DailyMinMax{TempMin: &tempMin, WindMax: &windMax},
		}),
	}

	recorder := httptest.NewRecorder()
	h.CurrentWeatherWidget(recorder, httptest.NewRequest("GET", "/", nil))
	if recorder.Code != 200 {
		t.Fatalf("CurrentWeatherWidget() status = %d, want 200", recorder.Code)
	}

	body := recorder.Body.String()
	for _, unexpected := range []string{"точка росы:", "порывы:", "месяц:", "0.0° / 0.0°", "0% / 0%", "0 / 0", "макс порыв: 0.0 м/с"} {
		if strings.Contains(body, unexpected) {
			t.Errorf("secondary metric with incomplete source data must be omitted, got %q", unexpected)
		}
	}
	for _, expected := range []string{"1.0°/ч", "макс: 6.0 м/с"} {
		if !strings.Contains(body, expected) {
			t.Errorf("secondary metric with complete source data is missing %q", expected)
		}
	}
}

func TestBuildCurrentWeatherWaterData(t *testing.T) {
	water := buildCurrentWeatherWaterData(WaterLevelCardData{
		HasData:         true,
		RelativeLevelCm: "168 см над нулём поста",
		DayChangeText:   "+5 см",
	})
	if !water.HasData || water.RelativeLevel != "168 см над нулём поста" || water.DayChange != "+5 см" {
		t.Fatalf("water summary = %+v, want relative level and daily change", water)
	}

	withoutReference := buildCurrentWeatherWaterData(WaterLevelCardData{HasData: true, LevelM: 161.681})
	if withoutReference.HasData {
		t.Fatal("water summary must not fall back to the unexplained absolute water-level mark")
	}
}

func TestCurrentWeatherTemplateRendersMobileWaterSummary(t *testing.T) {
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
	data := map[string]any{
		"Geomagnetic": map[string]bool{"HasData": false},
		"Water": currentWeatherWaterData{
			HasData:       true,
			RelativeLevel: "168 см над нулём поста",
			DayChange:     "+5 см",
		},
	}
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{`href="/detail/water-level"`, `sm:hidden`, "Кубань", "168 см над нулём поста", "&#43;5 см", "за 24 ч"} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("mobile water summary is missing %s", expected)
		}
	}
}

func TestCurrentWeatherTemplateOmitsWaterDailyChangeWhenUnavailable(t *testing.T) {
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
	if err := tmpl.Execute(&output, map[string]any{
		"Geomagnetic": map[string]bool{"HasData": false},
		"Water": currentWeatherWaterData{
			HasData:       true,
			RelativeLevel: "168 см над нулём поста",
		},
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if bytes.Contains(output.Bytes(), []byte("за 24 ч")) {
		t.Fatal("mobile water summary must omit the daily-change label when no daily change is available")
	}
}

func TestWeatherEventsTemplateOmitsEmptyState(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parsePartial("weather_events.html")
	if err != nil {
		t.Fatalf("parsePartial() error = %v", err)
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, map[string]any{"Events": []models.WeatherEvent(nil)}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if bytes.Contains(output.Bytes(), []byte("ui-surface")) {
		t.Fatal("empty weather events must clear the HTMX target without rendering a card")
	}
}

func TestSunTimesTemplateUsesCompactSummaryAndDisclosure(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}

	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parsePartial("sun_times.html")
	if err != nil {
		t.Fatalf("parsePartial() error = %v", err)
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, map[string]any{"Sunrise": "06:00", "Sunset": "18:00", "DayLength": "12ч", "HasMoonData": false}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{"Восход", "Закат", "День", "Солнце и Луна подробнее", "<details"} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("compact sun summary is missing %s", expected)
		}
	}
}

func TestIsGeomagneticAttention(t *testing.T) {
	for _, test := range []struct {
		name     string
		status   models.KpStatus
		peakLine string
		want     bool
	}{
		{name: "calm", status: models.KpCalm, want: false},
		{name: "forecast storm", status: models.KpCalm, peakLine: "Прогноз: буря G1", want: true},
		{name: "unsettled", status: models.KpUnsettled, want: true},
		{name: "storm", status: models.KpStorm, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isGeomagneticAttention(test.status, test.peakLine); got != test.want {
				t.Errorf("isGeomagneticAttention(%v, %q) = %t, want %t", test.status, test.peakLine, got, test.want)
			}
		})
	}
}

func TestForecastTemplateUsesCompactGridAndAccessibleLabels(t *testing.T) {
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
		AccessibleLabel          string
		PrecipitationProbability int16
		HasPrecipitation         bool
	}
	data := struct {
		Cards      []forecastCard
		NoForecast bool
	}{Cards: []forecastCard{{
		Label: "12:00", Icon: "☀️", TempMain: "20°", TempSecondary: "ощущ. 18°", WeatherDescription: "Ясно",
		AccessibleLabel: "12:00, Ясно, 20°", PrecipitationProbability: 40, HasPrecipitation: true,
	}}}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{"grid grid-cols-3", "lg:grid-cols-9", `aria-label="Периоды прогноза"`, `class="sr-only">12:00, Ясно, 20°`, `aria-hidden="true"`, "💧", "40%"} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered forecast is missing %s", expected)
		}
	}
	for _, unexpected := range []string{"snap-x", "overflow-x-auto", "w-40", "Проведите влево", "ощущ. 18°", ">Ясно<"} {
		if bytes.Contains(output.Bytes(), []byte(unexpected)) {
			t.Errorf("rendered forecast must not contain %s", unexpected)
		}
	}
}

func TestWaterLevelTemplateUsesCompactDashboardSummary(t *testing.T) {
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
	if err := tmpl.Execute(&output, WaterLevelCardData{LevelM: 220.28, StatusNote: "Уровень ниже порога, резких изменений нет."}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{"отметка уровня", "220.280 м", "за 24 часа", "статус", "Уровень ниже порога, резких изменений нет.", "График и пороги →", `min-h-11`} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered water level widget is missing %s", expected)
		}
	}
	for _, unexpected := range []string{"Сравнить посты выше Армавира", "hydroFill", "Шкала заполнена", "Балтийская система высот"} {
		if bytes.Contains(output.Bytes(), []byte(unexpected)) {
			t.Errorf("rendered water level widget must not contain %s", unexpected)
		}
	}
}

func TestWaterLevelTemplateShowsThresholdDetailsOnlyForWarnings(t *testing.T) {
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
	data := WaterLevelCardData{
		ShowThreshold: true,
		RiskHeadline:  "12 см",
		RiskCaption:   "до неблагоприятного уровня",
		StatusNote:    "Порог почти рядом.",
		ToDanger:      "48 см",
	}
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{"12 см до неблагоприятного уровня", "Порог почти рядом.", "До опасного: 48 см"} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered warning summary is missing %s", expected)
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
		`ui-status-surface`,
		`ui-field`, `ui-field-label`, `ui-form-item`, `ui-button-primary`, `ui-button-secondary`,
		`grid grid-cols-1 gap-5 sm:flex sm:flex-wrap sm:items-end`,
		`data-history-period="24h"`, `data-history-period="7d"`, `data-history-period="30d"`, `data-history-period="month"`,
		`id="history-status"`, `id="history-retry"`,
		`data-history-chart="temp"`, `data-history-chart="humidity"`, `data-history-chart="pressure"`,
		`data-history-chart="wind"`, `data-history-chart="rain"`, `data-history-chart="solar"`,
		`ui-chart-data`,
	} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered history is missing %s", expected)
		}
	}
}
