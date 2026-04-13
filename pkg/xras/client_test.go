package xras

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const sampleJSON = `{
  "version": "1.0",
  "type": "kp",
  "tzone": "UTC+3",
  "stime": "2026-04-11",
  "etime": "2026-04-13",
  "kp_type": "m",
  "kp_step": 3,
  "data": [
    {
      "time": "2026-04-11",
      "f10": 145.2,
      "sn": 78,
      "ap": 12,
      "max_kp": 4,
      "h00": 2, "h03": 3, "h06": 4, "h09": 3,
      "h12": 2, "h15": 2, "h18": 1, "h21": 2
    },
    {
      "time": "2026-04-12",
      "f10": null,
      "sn": null,
      "ap": null,
      "max_kp": 5,
      "h00": 3, "h03": 4, "h06": 5, "h09": null,
      "h12": null, "h15": null, "h18": null, "h21": null
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

	if resp.Tzone != "UTC+3" {
		t.Errorf("Tzone = %q, want UTC+3", resp.Tzone)
	}
	if resp.KpStep != 3 {
		t.Errorf("KpStep = %d, want 3", resp.KpStep)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(resp.Data))
	}

	// Первая запись — все слоты заполнены
	d0 := resp.Data[0]
	if d0.F10 == nil || *d0.F10 != 145.2 {
		t.Errorf("d0.F10 = %v, want 145.2", d0.F10)
	}
	slots0 := d0.Slots()
	if slots0[0] == nil || *slots0[0] != 2 {
		t.Errorf("d0 slot 0 = %v, want 2", slots0[0])
	}
	if slots0[2] == nil || *slots0[2] != 4 {
		t.Errorf("d0 slot 2 (h06) = %v, want 4", slots0[2])
	}

	// Вторая запись — null значения должны быть nil
	d1 := resp.Data[1]
	if d1.F10 != nil {
		t.Errorf("d1.F10 = %v, want nil", *d1.F10)
	}
	slots1 := d1.Slots()
	if slots1[3] != nil {
		t.Errorf("d1 slot 3 (h09) = %v, want nil", *slots1[3])
	}
	if slots1[2] == nil || *slots1[2] != 5 {
		t.Errorf("d1 slot 2 (h06) = %v, want 5", slots1[2])
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

func TestParseTzone(t *testing.T) {
	cases := []struct {
		in     string
		offset int // в секундах
	}{
		{"UTC+3", 3 * 3600},
		{"UTC-2", -2 * 3600},
		{"UTC", 0},
		{"utc+5", 5 * 3600},
	}
	for _, c := range cases {
		loc, err := ParseTzone(c.in)
		if err != nil {
			t.Errorf("ParseTzone(%q) error: %v", c.in, err)
			continue
		}
		// Получаем смещение через локальную дату
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
