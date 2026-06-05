package service

import (
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

func TestDashboardSplitAndSortCards(t *testing.T) {
	cards, quiet := splitAndSortCards([]models.AttentionCard{
		{ID: "water", Domain: "hydro", Title: "Вода в норме", Priority: 10},
		{ID: "storm", Domain: "geomagnetic", Title: "Магнитная буря", Priority: 90},
		{ID: "wind", Domain: "wind", Title: "Сильный ветер", Priority: 70},
	})

	if len(cards) != 2 {
		t.Fatalf("ожидалось 2 важные карточки, получено %d", len(cards))
	}
	if cards[0].ID != "storm" || cards[1].ID != "wind" {
		t.Fatalf("карточки отсортированы неверно: %#v", cards)
	}
	if len(quiet) != 1 || quiet[0] != "вода" {
		t.Fatalf("ожидалась спокойная вода, получено %#v", quiet)
	}
}

func TestBuildStationStatusMarksStaleData(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	current := &models.WeatherData{Time: now.Add(-65 * time.Minute)}

	status := buildStationStatus(current, now)
	if status.OK {
		t.Fatal("данные старше часа должны считаться неактуальными")
	}
	if status.Severity != string(models.DashboardSeverityDanger) {
		t.Fatalf("ожидался danger, получено %s", status.Severity)
	}
}

func TestBuildWindCardRaisesPriorityForStrongGusts(t *testing.T) {
	windSpeed := float32(4)
	windGust := float32(16)
	card := buildWindCard(&models.WeatherData{WindSpeed: &windSpeed, WindGust: &windGust})

	if card.Priority < 80 {
		t.Fatalf("сильные порывы должны быть высокоприоритетными, priority=%d", card.Priority)
	}
	if card.Severity != string(models.DashboardSeverityWarning) {
		t.Fatalf("ожидался warning, получено %s", card.Severity)
	}
}

func TestBuildRainCardKeepsNoRainQuiet(t *testing.T) {
	rainRate := float32(0)
	card := buildRainCard(&models.WeatherData{RainRate: &rainRate}, nil)

	if card.Priority > quietPriorityThreshold {
		t.Fatalf("отсутствие дождя должно быть спокойным, priority=%d", card.Priority)
	}
	if card.Severity != string(models.DashboardSeverityCalm) {
		t.Fatalf("ожидался calm, получено %s", card.Severity)
	}
}
