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
		`.ui-chart-visual`,
		`.ui-chart-legend-slot`,
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
		"dashboard.html": {"tempChart", "humidityChart", "pressureChart", "windChart", "solarChart"},
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
			if page == "dashboard.html" {
				for _, expected := range []string{
					`id="rainDailyChart" role="img" aria-label=`,
					`id="rainRateChart" role="img" aria-label=`,
					`id="rainChart-summary"`, `id="rainChart-details"`, `id="rainChart-table"`,
					`aria-describedby="rainDailyChart-summary rainChart-summary"`,
					`aria-describedby="rainRateChart-summary rainChart-summary"`,
				} {
					if !bytes.Contains(output.Bytes(), []byte(expected)) {
						t.Errorf("rendered dashboard is missing combined rain accessibility control %s", expected)
					}
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
			const currentChartsScript = `<script src="/static/js/charts.js?v=10"></script>`
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
		"function renderCombinedRainTable",
		"Осадки за день, мм",
		"Интенсивность, мм/ч",
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
		`aria-label="Шаг данных"`,
		`id="weather-events"`,
		`id="charts-24h"`,
		`ui-chart-plot`, `ui-chart-visual`, `ui-chart-legend-slot`, `ui-chart-data`,
		`aria-label="Таблица данных: ветер"`,
		`aria-label="Таблица данных: осадки за день и интенсивность"`,
		`aria-label="Таблица данных: освещённость"`,
		`Погода в Telegram`, `Уведомления от метеостанции`, `Перейти в бот`,
		`rel="noopener noreferrer"`, `message-circle`, `external-link`,
		`syncDashboardLowerGrid`,
		`id="sun-times"`,
		`id="telegram-bot-promo"`,
		`syncChartVisibility`,
		`desktopEventJournal`,
		`eventJournalOpen`,
		`<h1 class="sr-only">Погода в Армавире</h1>`,
		`<h2 class="ui-section-heading sr-only sm:not-sr-only px-4 pt-4 sm:px-6 sm:pt-6">Динамика погоды</h2>`,
		`lg:grid-cols-12`,
		`id="current-weather" class="order-1 lg:col-span-8"`,
		`aria-label="Прогноз"`,
		`contents order-2 lg:col-span-4 lg:flex lg:flex-col lg:gap-6`,
		`id="weather-events" class="order-2 empty:hidden lg:order-7 lg:col-span-8"`,
		`id="forecast" class="order-3 lg:order-1"`,
		`id="water-level" class="order-4 lg:hidden"`,
		`id="sun-times" class="order-5 lg:hidden"`,
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
	if forecastIndex >= waterIndex || waterIndex >= sunIndex || sunIndex >= chartsIndex || chartsIndex >= eventsIndex || eventsIndex >= telegramIndex {
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
	chartHeadingIndex := bytes.Index(output.Bytes(), []byte(`<h2 class="ui-section-heading sr-only sm:not-sr-only px-4 pt-4 sm:px-6 sm:pt-6">Динамика погоды</h2>`))
	chartDetailsIndex := bytes.Index(output.Bytes(), []byte(`<details id="charts-24h">`))
	if chartHeadingIndex < 0 || chartHeadingIndex > chartDetailsIndex {
		t.Fatal("chart heading must stay outside the collapsed mobile disclosure")
	}
	if !bytes.Contains(output.Bytes(), []byte(`hx-swap="innerHTML"></div>`)) {
		t.Fatal("weather-events HTMX target must be truly empty before its first response")
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
		"forecast.html":        `<h2 class="ui-section-heading">Прогноз</h2>`,
		"sun_times.html":       `<h2 class="ui-section-heading">Солнце и Луна</h2>`,
		"water_level.html":     `<h2 class="ui-section-heading">Уровень Кубани</h2>`,
		"weather_events.html":  `<h2 class="ui-card-heading min-w-0 flex-1 text-gray-900 dark:text-white">`,
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
		HasData:       true,
		DayChangeText: "-5 см",
		Upstream: []WaterLevelMiniData{
			{ObjectName: "Уруп р.", DayChangeText: "-2 см"},
		},
	})
	if !water.HasData || water.KubanChange != "-5 см" || water.UrupChange != "-2 см" {
		t.Fatalf("water summary = %+v, want Kuban and Urup daily changes", water)
	}

	withoutDailyChange := buildCurrentWeatherWaterData(WaterLevelCardData{HasData: true, LevelM: 161.681, RelativeLevelCm: "-732 см над нулём поста"})
	if withoutDailyChange.HasData {
		t.Fatal("water summary must not fall back to unexplained water-level marks without a daily change")
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
			HasData:     true,
			KubanChange: "-5 см",
			UrupChange:  "-2 см",
		},
	}
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{`href="/detail/water-level"`, `sm:hidden`, `aria-label="Изменение уровня рек за сутки"`, "Кубань", "Уруп", "-5 см", "-2 см", "за сутки", "whitespace-nowrap"} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("mobile water summary is missing %s", expected)
		}
	}
	if bytes.Contains(output.Bytes(), []byte("над нулём поста")) {
		t.Fatal("mobile water summary must not render an unexplained absolute level")
	}
}

func TestCurrentWeatherTemplateMarksUnavailableDesktopUrupDailyChange(t *testing.T) {
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
			HasData:     true,
			KubanChange: "-5 см",
		},
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := output.String()
	if !strings.Contains(body, "Уруп") || !strings.Contains(body, "нет данных") {
		t.Fatal("desktop river column must mark an unavailable daily change explicitly")
	}
}

func TestCurrentWeatherTemplateRendersDesktopNaturalConditionsColumns(t *testing.T) {
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
		"Water":       currentWeatherWaterData{HasData: true, KubanChange: "+3 см", UrupChange: "-1 см"},
		"Geomagnetic": GeomagneticCardData{HasData: true, Kp: 2.3, StatusHeading: "Спокойно", StatusText: "text-green-700", Sparkline: []SparkBar{{HeightPct: 28, Color: "#22c55e", Title: "Kp 2.3"}}},
		"Sun":         SunTimesData{HasData: true, Date: "24 сентября", Sunrise: "06:12", Sunset: "18:24", DayLength: "12ч 12мин", Dawn: "05:40", Dusk: "18:56", LightLength: "13ч 16мин", DayChangeDay: "+3мин", DayChangeWeek: "+21мин", DayChangeMonth: "+1ч 12мин", LightChangeDay: "+4мин", LightChangeWeek: "+28мин", LightChangeMonth: "+1ч 24мин", HasMoonData: true, MoonPhase: "Растущая луна", MoonIllumination: 35, Moonrise: "09:00", Moonset: "21:00", MoonAge: 5.2, NextPhaseName: "До полнолуния", DaysToNextPhase: 9},
	}
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{"lg:grid-cols-3", "Реки", "Геомагнитная обстановка", "Солнце и луна", "3 см", "1 см", "Kp 2.3", "Все астроданные", `id="current-astronomy-details"`, "за сутки", "24 сентября", "День за неделю", "День за месяц", "Светлое время вчера", "Светлое время за неделю", "Светлое время за месяц", "21мин", "1ч 12мин", "4мин", "28мин", "1ч 24мин"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("desktop natural conditions is missing %q", expected)
		}
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
	if output.Len() != 0 {
		t.Fatalf("empty weather events must clear the HTMX target, got %q", output.String())
	}
}

func TestWeatherEventsTemplateRendersAccessibleJournalSummary(t *testing.T) {
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
	events := []models.WeatherEvent{{Type: "wind_gust", Description: "Порыв ветра", Time: time.Date(2026, time.September, 25, 12, 30, 0, 0, time.Local)}}
	if err := tmpl.Execute(&output, map[string]any{"Events": events}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{"book-open-text", "Погодные события", "За последние 24 часа", "Всего: 1", "chevron-down", "ui-event-timeline", "12:30", "Порыв ветра"} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("event journal is missing %q", expected)
		}
	}
	if bytes.Contains(output.Bytes(), []byte(`data-icon="circle-alert"`)) {
		t.Fatal("event journal summary must not use a warning icon")
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

	data := forecastWidgetData{
		Cards: []forecastCard{{
			Label: "12:00", Icon: "☀️", TempMain: "20°", AccessibleLabel: "12:00, Ясно, 20°", PrecipitationProbability: 40, HasPrecipitation: true, IsHourly: true,
		}},
		HourlyCards: []forecastCard{{
			Label: "12:00", Icon: "☀️", TempMain: "20°", AccessibleLabel: "12:00, Ясно, 20°", PrecipitationProbability: 40, HasPrecipitation: true, IsHourly: true,
		}},
		DailyCards: []forecastCard{
			{Label: "Пт", Icon: "☀️", TempMain: "-12/-3°", AccessibleLabel: "Пт, Ясно, -12/-3°", DailyPrecipitation: "—", DailyWind: "—", DailyGusts: "—", DailyUV: "—"},
			{Label: "Сб", Icon: "🌧️", TempMain: "-10/2°", AccessibleLabel: "Сб, Дождь, -10/2°, вероятность осадков 100%", PrecipitationProbability: 100, HasPrecipitation: true, DailyPrecipitation: "3,2 мм · 100%", DailyWind: "8 м/с", DailyGusts: "12", DailyUV: "5"},
		},
		FetchedAtKnown: true,
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, expected := range []string{"grid grid-cols-3", "lg:hidden", `aria-label="Периоды прогноза"`, `aria-label="Прогноз на ближайшие часы"`, `aria-label="Прогноз на ближайшие дни"`, "Ближайшие дни", `class="sr-only">12:00, Ясно, 20°`, `aria-hidden="true"`, "40%", `data-icon="droplet"`, `data-icon="sun"`, "ui-forecast-hourly", "flex-col items-center", "px-2 py-2"} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered forecast is missing %s", expected)
		}
	}
	for _, unexpected := range []string{"snap-x", "overflow-x-auto", "w-40", "Проведите влево", "ощущ. 18°", ">Ясно<"} {
		if bytes.Contains(output.Bytes(), []byte(unexpected)) {
			t.Errorf("rendered forecast must not contain %s", unexpected)
		}
	}
	for _, expected := range []string{"ui-forecast-daily", "ui-forecast-daily-row", "ui-forecast-daily-top", "-12/-3°", "Осадки", "порывы 12", "3,2 мм · 100%", ">—<", "Open-Meteo"} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered daily forecast is missing %s", expected)
		}
	}
	if got := bytes.Count(output.Bytes(), []byte("ui-forecast-daily-row")); got != 2 {
		t.Errorf("rendered daily forecast has %d shared-grid rows, want 2", got)
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
