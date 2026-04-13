package xras

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Реальный формат xras.ru: все числа — строки, "null" — литерал, kp_step — "3h",
// tzone — "Krasnodar (UTC+03)".
const sampleJSON = `{
  "version": "1.0",
  "type": "kp",
  "error": "",
  "tzone": "Krasnodar (UTC+03)",
  "stime": "2026-04-11T00:00:00",
  "etime": "2026-04-13T23:59:59",
  "kp_type": "m",
  "kp_step": "3h",
  "data": [
    {
      "time": "2026-04-13",
      "f10": "null",
      "sn": "60",
      "ap": "6",
      "max_kp": "2.33",
      "h00": "2.33",
      "h03": "0.67",
      "h06": "1",
      "h09": "1",
      "h12": "-0.33",
      "h15": "null",
      "h18": "null",
      "h21": "null"
    },
    {
      "time": "2026-04-12",
      "f10": "98",
      "sn": "43",
      "ap": "8",
      "max_kp": "2.33",
      "h00": "1.33",
      "h03": "2.33",
      "h06": "2.33",
      "h09": "2.33",
      "h12": "2.33",
      "h15": "2.33",
      "h18": "1.33",
      "h21": "1.67"
    }
  ]
}`

func TestGetKpData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleJSON))
	}))
	defer srv.Close()

	c, err := NewClient(5*time.Second, srv.URL, "")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := c.GetKpData(context.Background())
	if err != nil {
		t.Fatalf("GetKpData: %v", err)
	}

	if resp.Tzone != "Krasnodar (UTC+03)" {
		t.Errorf("Tzone = %q, want Krasnodar (UTC+03)", resp.Tzone)
	}
	if resp.KpStep != "3h" {
		t.Errorf("KpStep = %q, want 3h", resp.KpStep)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(resp.Data))
	}

	// Первая запись: f10 = "null" → nil, h12 = -0.33 (отрицательное), h15..h21 = null
	d0 := resp.Data[0]
	if d0.F10Float() != nil {
		t.Errorf("d0.F10Float() = %v, want nil", *d0.F10Float())
	}
	if got := d0.SnFloat(); got == nil || *got != 60 {
		t.Errorf("d0.Sn = %v, want 60", got)
	}
	slots0 := d0.Slots()
	if got := slots0[0]; got == nil || *got != 2.33 {
		t.Errorf("d0 slot 0 (h00) = %v, want 2.33", got)
	}
	if got := slots0[4]; got == nil || *got != -0.33 {
		t.Errorf("d0 slot 4 (h12) = %v, want -0.33", got)
	}
	if slots0[5] != nil || slots0[6] != nil || slots0[7] != nil {
		t.Errorf("d0 slots 5-7 should be nil, got %v %v %v", slots0[5], slots0[6], slots0[7])
	}

	// Вторая запись: все слоты заполнены, max_kp = 2.33
	d1 := resp.Data[1]
	if got := d1.F10Float(); got == nil || *got != 98 {
		t.Errorf("d1.F10 = %v, want 98", got)
	}
	if got := d1.MaxKpFloat(); got == nil || *got != 2.33 {
		t.Errorf("d1.MaxKp = %v, want 2.33", got)
	}
	slots1 := d1.Slots()
	for i, s := range slots1 {
		if s == nil {
			t.Errorf("d1 slot %d is nil, expected value", i)
		}
	}
}

func TestGetKpData_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c, _ := NewClient(5*time.Second, srv.URL, "")
	_, err := c.GetKpData(context.Background())
	if err == nil {
		t.Fatal("expected error on 503, got nil")
	}
}

func TestParseNullableFloat(t *testing.T) {
	cases := []struct {
		in   string
		want *float64
	}{
		{"", nil},
		{"null", nil},
		{"  null  ", nil},
		{"abc", nil},
	}
	for _, c := range cases {
		got := ParseNullableFloat(c.in)
		if got != nil {
			t.Errorf("ParseNullableFloat(%q) = %v, want nil", c.in, *got)
		}
	}
	// Числовые
	for _, in := range []string{"0", "2.33", "-0.33", "98"} {
		got := ParseNullableFloat(in)
		if got == nil {
			t.Errorf("ParseNullableFloat(%q) = nil, want non-nil", in)
		}
	}
}

func TestParseTzone(t *testing.T) {
	cases := []struct {
		in     string
		offset int // в секундах
	}{
		{"UTC+3", 3 * 3600},
		{"UTC-2", -2 * 3600},
		{"UTC", 0},
		{"utc+5", 5 * 3600},
		{"Krasnodar (UTC+03)", 3 * 3600},
		{"Moscow (UTC+3)", 3 * 3600},
		{"Honolulu (UTC-10)", -10 * 3600},
	}
	for _, c := range cases {
		loc, err := ParseTzone(c.in)
		if err != nil {
			t.Errorf("ParseTzone(%q) error: %v", c.in, err)
			continue
		}
		_, off := time.Date(2026, 1, 1, 0, 0, 0, 0, loc).Zone()
		if off != c.offset {
			t.Errorf("ParseTzone(%q) offset = %d, want %d", c.in, off, c.offset)
		}
	}
}

func TestParseTzone_FallbackOnEmpty(t *testing.T) {
	loc, err := ParseTzone("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, off := time.Date(2026, 1, 1, 0, 0, 0, 0, loc).Zone()
	if off != 3*3600 {
		t.Errorf("empty tzone should fallback to UTC+3, got offset %d", off)
	}
}
