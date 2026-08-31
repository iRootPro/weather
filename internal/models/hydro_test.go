package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHydroLevelReadingExposesHDIIHRAsSourceIndex(t *testing.T) {
	value := float32(1.407)
	payload, err := json.Marshal(HydroLevelReading{SourceHDIIHR: &value})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	jsonText := string(payload)
	if !strings.Contains(jsonText, `"source_hdi_ihr":1.407`) {
		t.Fatalf("payload = %s, want source_hdi_ihr", jsonText)
	}
	if strings.Contains(jsonText, `"change_cm_per_hour"`) {
		t.Fatalf("payload = %s, source index must not claim cm/hour units", jsonText)
	}
}
