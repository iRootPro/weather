package models

import "testing"

func TestClassifyKp(t *testing.T) {
	cases := []struct {
		kp   float32
		want KpStatus
	}{
		{0, KpCalm},
		{2.5, KpCalm},
		{3.99, KpCalm},
		{4, KpUnsettled},
		{4.9, KpUnsettled},
		{5, KpStorm},
		{6, KpStorm},
		{6.99, KpStorm},
		{7, KpSevereStorm},
		{9, KpSevereStorm},
	}
	for _, c := range cases {
		got := ClassifyKp(c.kp)
		if got != c.want {
			t.Errorf("ClassifyKp(%v) = %v, want %v", c.kp, got, c.want)
		}
	}
}

func TestKpStatusHelpers(t *testing.T) {
	// Каждый статус возвращает непустую строку для всех вспомогательных методов
	for _, s := range []KpStatus{KpCalm, KpUnsettled, KpStorm, KpSevereStorm} {
		if s.Label() == "" {
			t.Errorf("Label() empty for %v", s)
		}
		if s.Color() == "" {
			t.Errorf("Color() empty for %v", s)
		}
		if s.Emoji() == "" {
			t.Errorf("Emoji() empty for %v", s)
		}
		if s.TailwindGradient() == "" {
			t.Errorf("TailwindGradient() empty for %v", s)
		}
		if s.TextColor() == "" {
			t.Errorf("TextColor() empty for %v", s)
		}
	}
}
