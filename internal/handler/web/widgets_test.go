package web

import (
	"bytes"
	"path/filepath"
	"runtime"
	"slices"
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
	if got, want := cards[8].AccessibleLabel, "Вт, Облачно, 16/26°, осадки —, ветер —, порывы —, UV —"; got != want {
		t.Errorf("cards[8].AccessibleLabel = %q, want %q", got, want)
	}
}

func TestFormatDailyDetailsUsesReadableLocalizedNumbers(t *testing.T) {
	details := formatDailyDetails(models.DailyForecast{
		PrecipitationSum:            0,
		HasPrecipitationSum:         true,
		PrecipitationProbability:    40,
		HasPrecipitationProbability: true,
	})
	if got, want := details.Precipitation, "0 мм · 40%"; got != want {
		t.Errorf("daily precipitation = %q, want %q", got, want)
	}
}

func TestForecastCardsMarkOnlyExpectedPrecipitationForBlueTone(t *testing.T) {
	now := time.Date(2026, time.September, 25, 10, 0, 0, 0, time.UTC)
	cards := buildForecastCards(now,
		[]models.HourlyForecast{{
			Time: now.Add(time.Hour), HasPrecipitation: true, HasPrecipitationProbability: true,
		}},
		[]models.DailyForecast{{
			Date: now.AddDate(0, 0, 1), HasPrecipitationSum: true, HasPrecipitationProbability: true, PrecipitationProbability: 20,
		}},
	)
	if cards[0].HasExpectedPrecipitation {
		t.Fatal("known 0 mm and 0% hourly conditions must use the neutral precipitation tone")
	}
	if !cards[1].HasExpectedPrecipitation {
		t.Fatal("daily precipitation probability must retain the semantic precipitation tone even with a 0 mm total")
	}
}

func TestForecastWidgetDataAndTemplateExposeFreshnessAndPartialStates(t *testing.T) {
	now := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	fresh := []models.HourlyForecast{{
		Time:      now.Add(time.Hour),
		FetchedAt: now.Add(-time.Hour),
	}}
	daily := []models.DailyForecast{{
		Date:      now.AddDate(0, 0, 1),
		FetchedAt: now.Add(-time.Hour),
	}}

	tests := []struct {
		name       string
		data       forecastWidgetData
		contains   []string
		notContain []string
	}{
		{
			name:       "fresh complete forecast",
			data:       buildForecastWidgetData(now, fresh, daily),
			contains:   []string{"самое раннее обновление показанных данных 25 сентября 2026, 11:00", "обновлено в 11:00"},
			notContain: []string{"устарели", "Время обновления части прогноза неизвестно.", "Почасовой прогноз пока недоступен", "Прогноз на ближайшие дни пока недоступен"},
		},
		{
			name:     "stale forecast",
			data:     buildForecastWidgetData(now, []models.HourlyForecast{{Time: now.Add(time.Hour), FetchedAt: now.Add(-3 * time.Hour)}}, nil),
			contains: []string{"Данные прогноза устарели: самое раннее обновление показанных данных 25 сентября 2026, 09:00.", "Прогноз на ближайшие дни пока недоступен"},
		},
		{
			name:     "stale hourly with fresh daily forecast",
			data:     buildForecastWidgetData(now, []models.HourlyForecast{{Time: now.Add(time.Hour), FetchedAt: now.Add(-3 * time.Hour)}}, daily),
			contains: []string{"Данные прогноза устарели: самое раннее обновление показанных данных 25 сентября 2026, 09:00."},
		},
		{
			name:     "daily only with unknown timestamp",
			data:     buildForecastWidgetData(now, nil, []models.DailyForecast{{Date: now.AddDate(0, 0, 1)}}),
			contains: []string{"Время обновления части прогноза неизвестно.", "Почасовой прогноз пока недоступен"},
		},
		{
			name:     "unknown hourly timestamp with fresh daily forecast",
			data:     buildForecastWidgetData(now, []models.HourlyForecast{{Time: now.Add(time.Hour)}}, daily),
			contains: []string{"Время обновления части прогноза неизвестно.", "самое раннее обновление показанных данных 25 сентября 2026, 11:00"},
		},
		{
			name: "filtered daily timestamp does not make visible forecast stale",
			data: buildForecastWidgetData(now, fresh, []models.DailyForecast{{
				Date:      now,
				FetchedAt: now.Add(-3 * time.Hour),
			}}),
			notContain: []string{"устарели", "Время обновления части прогноза неизвестно."},
		},
		{
			name:     "empty forecast",
			data:     buildForecastWidgetData(now, nil, nil),
			contains: []string{"Прогноз пока недоступен", "Время обновления части прогноза неизвестно."},
		},
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}
	h := &Handler{templatesDir: filepath.Join(filepath.Dir(filename), "..", "..", "web", "templates")}
	tmpl, err := h.parsePartial("forecast.html")
	if err != nil {
		t.Fatalf("parsePartial() error = %v", err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := tmpl.Execute(&output, test.data); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			for _, want := range test.contains {
				if !strings.Contains(output.String(), want) {
					t.Errorf("rendered forecast missing %q", want)
				}
			}
			for _, unwanted := range test.notContain {
				if strings.Contains(output.String(), unwanted) {
					t.Errorf("rendered forecast unexpectedly contains %q", unwanted)
				}
			}
		})
	}
}

func TestForecastCardsExposeOnlyAvailableDetailsAndFreshSummary(t *testing.T) {
	now := time.Date(2026, time.September, 25, 10, 0, 0, 0, time.UTC)
	hourly := []models.HourlyForecast{
		{Time: now.Add(time.Hour), Temperature: 20, FeelsLike: 16, HasTemperature: true, HasFeelsLike: true, Precipitation: 0.4, PrecipitationProbability: 60, HasPrecipitation: true, HasPrecipitationProbability: true},
		{Time: now.Add(2 * time.Hour), Temperature: 19, HasTemperature: true},
		{Time: now.Add(3 * time.Hour), Temperature: 19, HasTemperature: true},
		{Time: now.Add(4 * time.Hour), Temperature: 19, WindGusts: 16, HasTemperature: true, HasWindGusts: true},
	}
	daily := []models.DailyForecast{{Date: now.AddDate(0, 0, 1), PrecipitationSum: 1.2, WindSpeedMax: 9, WindGustsMax: 14, UVIndexMax: 6, HasPrecipitationSum: true, HasWindSpeedMax: true, HasWindGustsMax: true, HasUVIndexMax: true}}
	cards := buildForecastCards(now, hourly, daily)
	if got, want := cards[0].Precipitation, "0.4 мм · 60%"; got != want {
		t.Errorf("hourly precipitation = %q, want %q", got, want)
	}
	if got, want := cards[0].FeelsLike, "ощущается 16°"; got != want {
		t.Errorf("feels like = %q, want %q", got, want)
	}
	if cards[0].Wind != "" || cards[1].Wind != "порывы 16" {
		t.Errorf("notable wind = %q, %q", cards[0].Wind, cards[1].Wind)
	}
	if got, want := []string{cards[2].DailyPrecipitation, cards[2].DailyWind, cards[2].DailyGusts, cards[2].DailyUV}, []string{"1,2 мм", "9 м/с", "14", "6"}; !slices.Equal(got, want) {
		t.Errorf("daily details = %q, want %q", got, want)
	}

	hourly[0].FetchedAt = now.Add(-time.Hour)
	hourly[1].FetchedAt = now.Add(-time.Hour)
	hourly[2].FetchedAt = now.Add(-time.Hour)
	hourly[3].FetchedAt = now.Add(-time.Hour)
	daily[0].FetchedAt = now.Add(-time.Hour)
	fresh := buildForecastWidgetData(now, hourly, daily)
	if got, want := fresh.Summary, "Осадки вероятны с 11:00"; got != want {
		t.Errorf("fresh summary = %q, want %q", got, want)
	}
	staleHourly := append([]models.HourlyForecast(nil), hourly...)
	staleHourly[0].FetchedAt = now.Add(-3 * time.Hour)
	if got := buildForecastWidgetData(now, staleHourly, daily).Summary; got != "" {
		t.Errorf("stale summary = %q, want empty", got)
	}
}

func TestForecastCardsDoNotRenderMissingDailyTemperaturesAsZero(t *testing.T) {
	now := time.Date(2026, time.September, 25, 10, 0, 0, 0, time.UTC)
	cards := buildForecastCards(now, nil, []models.DailyForecast{{Date: now.AddDate(0, 0, 1)}})
	if len(cards) != 1 || cards[0].TempMain != "—/—°" {
		t.Errorf("daily temperature = %#v, want —/—°", cards)
	}
}

func TestForecastSummaryUsesOnlyVisibleFreshHourlyRows(t *testing.T) {
	now := time.Date(2026, time.September, 25, 10, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Hour)
	stale := now.Add(-3 * time.Hour)
	forecast := []models.HourlyForecast{
		{Time: now.Add(time.Hour), FetchedAt: fresh, Temperature: 20, HasTemperature: true},
		{Time: now.Add(2 * time.Hour), FetchedAt: fresh, Temperature: 20, HasTemperature: true},
		{Time: now.Add(3 * time.Hour), FetchedAt: fresh, Temperature: 20, HasTemperature: true},
		{Time: now.Add(4 * time.Hour), FetchedAt: fresh, Temperature: 20, HasTemperature: true},
		{Time: now.Add(5 * time.Hour), FetchedAt: stale, Precipitation: 1, PrecipitationProbability: 80, HasPrecipitation: true, HasPrecipitationProbability: true},
	}
	if got := buildForecastWidgetData(now, forecast, nil).Summary; got != "" {
		t.Errorf("summary = %q, want empty because notable row is not displayed", got)
	}
}

func TestWeatherIconMapsKnownWMOCodesAndUsesUnknownFallback(t *testing.T) {
	tests := []struct {
		name     string
		code     int16
		iconName string
	}{
		{name: "clear", code: 0, iconName: "sun"},
		{name: "partly cloudy", code: 1, iconName: "cloud-sun"},
		{name: "cloudy", code: 3, iconName: "cloud"},
		{name: "fog", code: 45, iconName: "cloud-fog"},
		{name: "drizzle", code: 51, iconName: "cloud-drizzle"},
		{name: "rain", code: 63, iconName: "cloud-rain"},
		{name: "snow", code: 73, iconName: "cloud-snow"},
		{name: "thunder", code: 95, iconName: "cloud-lightning"},
		{name: "thunder with light hail", code: 96, iconName: "cloud-lightning"},
		{name: "thunder with hail", code: 99, iconName: "cloud-lightning"},
		{name: "unknown positive", code: 100, iconName: "circle-alert"},
		{name: "unknown negative", code: -1, iconName: "circle-alert"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			icon := string(weatherIcon(test.code))
			if !strings.Contains(icon, `<svg`) || !strings.Contains(icon, `ui-weather-icon`) {
				t.Fatalf("weatherIcon(%d) = %q, want static SVG", test.code, icon)
			}
			if !strings.Contains(icon, `data-icon="`+test.iconName+`"`) {
				t.Errorf("weatherIcon(%d) = %q, want Lucide %q", test.code, icon, test.iconName)
			}
		})
	}
}

func TestMoonPhaseIconMapsKnownPhasesAndUsesSafeFallback(t *testing.T) {
	seen := make(map[string]string)
	for _, phase := range []string{"Новолуние", "Растущая луна", "Первая четверть", "Прибывающая луна", "Полнолуние", "Убывающая луна", "Последняя четверть", "Стареющая луна"} {
		got := string(moonPhaseIcon(phase))
		if !strings.Contains(got, `data-moon-phase="true"`) {
			t.Errorf("moonPhaseIcon(%q) = %q, want distinct static phase SVG", phase, got)
		}
		if previous, duplicate := seen[got]; duplicate {
			t.Errorf("moonPhaseIcon(%q) duplicates phase geometry for %q", phase, previous)
		}
		seen[got] = phase
	}
	for phase, arc := range map[string]string{
		"Растущая луна":    `A4 9 0 0 0 12 3`,
		"Прибывающая луна": `A4 9 0 0 1 12 3`,
		"Убывающая луна":   `A4 9 0 0 0 12 3`,
		"Стареющая луна":   `A4 9 0 0 1 12 3`,
	} {
		if got := string(moonPhaseIcon(phase)); !strings.Contains(got, arc) {
			t.Errorf("moonPhaseIcon(%q) = %q, want non-normalized inner arc %q", phase, got, arc)
		}
	}
	if got := string(moonPhaseIcon("неизвестная фаза")); !strings.Contains(got, `data-icon="circle-alert"`) {
		t.Errorf("moonPhaseIcon unknown = %q, want neutral safe fallback", got)
	}
}

func TestEventIconUsesEventTypeRatherThanBackendEmoji(t *testing.T) {
	for eventType, want := range map[string]string{
		"rain_start":    "cloud-rain",
		"rain_end":      "cloud-rain",
		"temp_drop":     "thermometer",
		"temp_rise":     "thermometer",
		"wind_gust":     "wind",
		"pressure_drop": "gauge",
		"pressure_rise": "gauge",
	} {
		if got := string(eventIcon(eventType)); !strings.Contains(got, `data-icon="`+want+`"`) {
			t.Errorf("eventIcon(%q) = %q, want %q", eventType, got, want)
		}
	}
	if got := string(eventIcon("unknown")); !strings.Contains(got, `data-icon="circle-alert"`) {
		t.Errorf("eventIcon unknown = %q, want safe fallback", got)
	}
}

func TestIconRejectsUnknownName(t *testing.T) {
	if got := string(icon(`<path d="untrusted"/>`)); !strings.Contains(got, `data-icon="circle-alert"`) || strings.Contains(got, "untrusted") {
		t.Errorf("icon() must not render untrusted SVG, got %q", got)
	}
}

func TestBuildSunTimesDataWithoutServiceIsUnavailable(t *testing.T) {
	h := &Handler{}
	if data := h.buildSunTimesData(time.Now()); data.HasData {
		t.Fatal("sun data must be unavailable when the sun service is not configured")
	}
}
