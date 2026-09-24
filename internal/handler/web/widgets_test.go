package web

import (
	"strings"
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

func TestWeatherIconMapsKnownWMOCodesAndUsesUnknownFallback(t *testing.T) {
	tests := []struct {
		name     string
		code     int16
		contains string
	}{
		{name: "clear", code: 0, contains: `r="4"`},
		{name: "partly cloudy", code: 1, contains: `r="3.5"`},
		{name: "cloudy", code: 3, contains: `M5 18h12`},
		{name: "fog", code: 45, contains: `M5 11h14`},
		{name: "drizzle", code: 51, contains: `M8 17l-1 2`},
		{name: "rain", code: 63, contains: `M8 16l-1 3`},
		{name: "snow", code: 73, contains: `M8 17h.01`},
		{name: "thunder", code: 95, contains: `m12 15-2 4`},
		{name: "thunder with light hail", code: 96, contains: `m12 15-2 4`},
		{name: "thunder with hail", code: 99, contains: `m12 15-2 4`},
		{name: "unknown positive", code: 100, contains: `r="8"`},
		{name: "unknown negative", code: -1, contains: `r="8"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			icon := string(weatherIcon(test.code))
			if !strings.Contains(icon, `<svg`) || !strings.Contains(icon, `data-weather-icon="true"`) {
				t.Fatalf("weatherIcon(%d) = %q, want static SVG", test.code, icon)
			}
			if !strings.Contains(icon, test.contains) {
				t.Errorf("weatherIcon(%d) = %q, want %q", test.code, icon, test.contains)
			}
		})
	}
}
