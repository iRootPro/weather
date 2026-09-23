package web

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

func TestInsightsTemplateRendersArchiveControls(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}
	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parseTemplate("insights.html")
	if err != nil {
		t.Fatalf("parseTemplate() error = %v", err)
	}

	tempMin, tempAvg, tempMax, rain := float32(12.3), float32(18.4), float32(25.6), float32(3.7)
	var output bytes.Buffer
	data := PageData{ActivePage: "insights", Data: &models.WeatherArchivePage{
		Period: "month", Metric: "all", PeriodLabel: "Август 2026", MonthParam: "2026-08", YearParam: 2026,
		Summary: models.WeatherArchiveSummary{DaysWithData: 1, DaysInPeriod: 1, HasTemp: true, HasRain: true},
		Events:  []models.WeatherArchiveEvent{{Icon: "🌡️", Title: "Самый жаркий день", Date: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC), Value: 25.6, Unit: "°C"}},
		Daily: []models.DailyWeatherInsight{{
			Date:    time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
			TempMin: &tempMin, TempAvg: &tempAvg, TempMax: &tempMax, RainTotal: &rain,
		}},
	}}
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !bytes.Contains(output.Bytes(), []byte(`id="insights-content"`)) {
		t.Fatal("archive content target is missing")
	}
	if !bytes.Contains(output.Bytes(), []byte(`hx-get="/insights"`)) {
		t.Fatal("archive period form is not HTMX-enabled")
	}
	for _, expected := range []string{
		`<section class="ui-surface px-4 py-5 sm:px-6">`,
		`class="mt-4 grid grid-cols-1 gap-5 md:flex md:flex-row md:flex-wrap md:items-end"`,
		`class="ui-field mt-2 block w-full"`,
		`class="ui-field-label ui-form-item min-w-0 w-full max-w-full md:w-auto md:min-w-48"`,
		`class="ui-metric-card basis-full p-4"`,
	} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered archive is missing %s", expected)
		}
	}
	if bytes.Contains(output.Bytes(), []byte("bg-gradient-to-br from-slate-900 via-blue-950")) {
		t.Fatal("archive must not use the legacy promotional gradient banner")
	}
	if bytes.Contains(output.Bytes(), []byte("</nav>\n        </div>\n\n        <form")) {
		t.Fatal("archive period navigation must remain inside the form-panel surface")
	}
	if !bytes.Contains(output.Bytes(), []byte(`name="search_field"`)) || !bytes.Contains(output.Bytes(), []byte("Найти дни по условию")) {
		t.Fatal("archive day search controls are missing")
	}
	for _, expected := range []string{
		`id="archive-table-scroll-hint"`,
		`Листайте таблицу влево`,
		`role="region"`,
	} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Errorf("rendered template is missing %s", expected)
		}
	}
	if !bytes.Contains(output.Bytes(), []byte("События периода")) || !bytes.Contains(output.Bytes(), []byte("Самый жаркий день")) {
		t.Fatal("archive period events are missing")
	}
	if !bytes.Contains(output.Bytes(), []byte("12.3°")) || !bytes.Contains(output.Bytes(), []byte("3.7 мм")) {
		t.Fatal("daily pointer values were not rendered as measurements")
	}

	for _, test := range []struct {
		name       string
		period     string
		controller string
	}{
		{name: "month", period: "month", controller: `name="month"`},
		{name: "season", period: "season", controller: `name="season"`},
		{name: "year", period: "year", controller: `name="year"`},
		{name: "range", period: "range", controller: `name="from"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			page := *data.Data.(*models.WeatherArchivePage)
			page.Period = test.period
			page.SeasonParam = "summer-2026"
			page.SeasonOptions = []models.WeatherInsightsPeriodOption{{Value: "summer-2026", Label: "Лето 2026"}}
			page.YearParam = 2026
			page.YearOptions = []int{2026}
			page.FromParam = "2026-08-01"
			page.ToParam = "2026-08-31"
			page.FirstDateParam = "2024-01-01"
			page.LastDateParam = "2026-08-31"

			var rendered bytes.Buffer
			if err := tmpl.Execute(&rendered, PageData{ActivePage: "insights", Data: &page}); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			for _, expected := range []string{`id="insights-content"`, `hx-target="#insights-content"`, `hx-swap="outerHTML"`, test.controller, `name="search_field"`, `ui-form-item min-w-0 w-full`, `grid grid-cols-1 gap-4 sm:flex sm:flex-row sm:items-end`} {
				if !bytes.Contains(rendered.Bytes(), []byte(expected)) {
					t.Errorf("rendered %s archive is missing %s", test.name, expected)
				}
			}
		})
	}
}
