package web

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"

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

	var output bytes.Buffer
	data := PageData{ActivePage: "insights", Data: &models.WeatherInsightsPage{}}
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !bytes.Contains(output.Bytes(), []byte(`id="insights-content"`)) {
		t.Fatal("archive content target is missing")
	}
	if !bytes.Contains(output.Bytes(), []byte(`hx-get="/insights"`)) {
		t.Fatal("archive period form is not HTMX-enabled")
	}
}
