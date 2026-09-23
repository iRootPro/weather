package web

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
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
		`aria-label="Скрыть предложение Telegram-бота"`,
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
		`href="#charts-24h"`,
		`id="weather-events"`,
		`id="charts-24h"`,
		`ui-chart-plot`, `ui-chart-plot-with-legend`, `ui-chart-data`,
		`id="telegram-bot-card" class="ui-surface order-9`,
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
	if !bytes.Contains(output.Bytes(), []byte(`class="ui-tabular text-sm`)) {
		t.Fatal("current weather timestamp must use the tabular typography role")
	}
	if bytes.Contains(output.Bytes(), []byte("hover:scale-105")) {
		t.Fatal("metric cards must not use scale as their primary hover feedback")
	}
	if !bytes.Contains(output.Bytes(), []byte("ui-metric-card")) {
		t.Fatal("current weather values must use the shared metric card role")
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
