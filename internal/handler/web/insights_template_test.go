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
	if !bytes.Contains(output.Bytes(), []byte("12.3°")) || !bytes.Contains(output.Bytes(), []byte("3.7 мм")) {
		t.Fatal("daily pointer values were not rendered as measurements")
	}
}
