package web

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

func TestMetricAttentionTemplateShowsContextAndMissingValues(t *testing.T) {
	h := &Handler{templatesDir: "../../web/templates"}
	tmpl, err := h.parsePartial("current_weather.html")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	temp := float32(36)
	data := map[string]any{
		"Attention":   buildCurrentAttention(&models.WeatherData{Time: now, TempOutdoor: &temp}, nil, now),
		"TempOutdoor": temp,
		"Missing":     map[string]bool{"wind": true, "rain": true, "uv": true, "solar": true},
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"ui-attention-heat", "Очень жарко", "Важно", "UV —"} {
		if !strings.Contains(output.String(), text) {
			t.Errorf("missing %q", text)
		}
	}
	// The shared annotation renders nothing for normal conditions.
	annotation, err := template.New("annotation").Parse(`{{define "annotation"}}{{template "metric-attention" .}}{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = annotation.AddParseTree("metric-attention", tmpl.Lookup("metric-attention").Tree)
	if err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := annotation.Execute(&output, metricAttention{}); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatal("normal conditions should have no annotation")
	}
}

func TestCurrentAttentionTemperatureBoundaries(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		value     float32
		title     string
		important bool
	}{
		{29.9, "", false}, {30, "Жарко", false}, {34.9, "Жарко", false}, {35, "Очень жарко", true},
		{0.1, "", false}, {0, "Холодно", false}, {-9.9, "Холодно", false}, {-10, "Сильный мороз", true},
	} {
		got := buildCurrentAttention(&models.WeatherData{Time: now, TempOutdoor: &tc.value}, nil, now).Temperature
		if got.Title != tc.title || got.Important != tc.important {
			t.Errorf("%v: got %+v", tc.value, got)
		}
	}
}

func TestCurrentAttentionFreshnessAndMultipleSignals(t *testing.T) {
	now := time.Now()
	temp, gust, rain, uv := float32(35), float32(17), float32(7.5), float32(8)
	current := &models.WeatherData{Time: now, TempOutdoor: &temp, WindGust: &gust, RainRate: &rain, UVIndex: &uv}
	got := buildCurrentAttention(current, nil, now)
	if got.Notice == "" || !got.Wind.Important || !got.Rain.Important || !got.Solar.Important {
		t.Fatalf("missing signals: %+v", got)
	}
	current.Time = now.Add(-10 * time.Minute)
	got = buildCurrentAttention(current, nil, now)
	if got.Notice == "" || got.Temperature.Title != "" || got.Wind.Title != "" || got.Rain.Title != "" || got.Solar.Title != "" {
		t.Fatalf("stale data: %+v", got)
	}
	got = buildCurrentAttention(&models.WeatherData{Time: now}, nil, now)
	if got != (currentAttention{}) {
		t.Fatalf("missing values should not trigger: %+v", got)
	}
}

func TestCurrentAttentionPressureRequiresRealHourlyInterval(t *testing.T) {
	now := time.Now()
	p, old := float32(750), float32(753)
	current := &models.WeatherData{Time: now, PressureRelative: &p}
	previous := &models.WeatherData{Time: now.Add(-time.Hour), PressureRelative: &old}
	if !buildCurrentAttention(current, previous, now).Pressure.Important {
		t.Fatal("missing pressure change")
	}
	previous.Time = now.Add(-3 * time.Hour)
	if buildCurrentAttention(current, previous, now).Pressure.Title != "" {
		t.Fatal("three hours must not be called an hourly change")
	}
}
