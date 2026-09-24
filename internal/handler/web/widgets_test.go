package web

import (
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

func TestBuildForecastCardsKeepsCompactForecastDataAndAccessibleNames(t *testing.T) {
	now := time.Date(2026, time.September, 23, 10, 0, 0, 0, time.Local)
	hourly := make([]models.HourlyForecast, 13)
	for i := range hourly {
		hourly[i] = models.HourlyForecast{
			Time:               now.Add(time.Duration(i) * time.Hour),
			Temperature:        float32(20 + i),
			WeatherDescription: "Переменная облачность",
			Icon:               "⛅",
		}
	}
	hourly[3].PrecipitationProbability = 60

	daily := make([]models.DailyForecast, 8)
	for i := range daily {
		daily[i] = models.DailyForecast{
			Date:               now.AddDate(0, 0, i),
			TemperatureMin:     float32(10 + i),
			TemperatureMax:     float32(20 + i),
			WeatherDescription: "Облачно",
			Icon:               "☁️",
		}
	}
	daily[1].WeatherDescription = "Дождь"
	daily[1].Icon = "🌧️"
	daily[1].PrecipitationProbability = 80

	cards := buildForecastCards(now, hourly, daily)
	if len(cards) != 9 {
		t.Fatalf("buildForecastCards() returned %d cards, want 9", len(cards))
	}

	for index, wantLabel := range []string{"10:00", "13:00", "16:00", "Чт", "Пт"} {
		if cards[index].Label != wantLabel {
			t.Errorf("cards[%d].Label = %q, want %q", index, cards[index].Label, wantLabel)
		}
	}
	if cards[1].TempMain != "23°" || !cards[1].HasPrecipitation {
		t.Errorf("hourly card = %#v, want compact temperature and precipitation", cards[1])
	}
	if !cards[0].IsHourly || !cards[2].IsHourly || cards[3].IsHourly {
		t.Errorf("forecast card periods = %#v, want three hourly cards followed by daily cards", cards[:4])
	}
	if cards[3].TempMain != "11/21°" || !cards[3].HasPrecipitation {
		t.Errorf("daily card = %#v, want tomorrow's compact range and precipitation", cards[3])
	}
	if got, want := cards[1].AccessibleLabel, "13:00, Переменная облачность, 23°, вероятность осадков 60%"; got != want {
		t.Errorf("cards[1].AccessibleLabel = %q, want %q", got, want)
	}
	if got, want := cards[8].AccessibleLabel, "Вт, Облачно, 16/26°"; got != want {
		t.Errorf("cards[8].AccessibleLabel = %q, want %q", got, want)
	}
}
