package web

import (
	"fmt"
	"math"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

type metricAttention struct {
	Tone      string
	Title     string
	Detail    string
	Important bool
	Severity  string
}

func (a metricAttention) Icon() string {
	switch a.Tone {
	case "heat", "cold":
		return "thermometer"
	case "wind":
		return "wind"
	case "rain":
		return "cloud-rain"
	case "solar":
		return "sun"
	default:
		return "gauge"
	}
}

func (a metricAttention) LevelLabel() string {
	if a.Severity == "danger" {
		return "Высокий уровень"
	}
	return "Обратите внимание"
}

type currentAttention struct {
	Temperature metricAttention
	Wind        metricAttention
	Rain        metricAttention
	Solar       metricAttention
	Pressure    metricAttention
	Notice      string
}

// Signals keeps header icons in the same stable order as the weather domains.
func (a currentAttention) Signals() []metricAttention {
	var signals []metricAttention
	for _, signal := range []metricAttention{a.Temperature, a.Wind, a.Rain, a.Solar, a.Pressure} {
		if signal.Title != "" {
			signals = append(signals, signal)
		}
	}
	return signals
}

// Attention describes observations only; missing or stale measurements never
// become current weather signals. Thresholds apply to unrounded values.
func buildCurrentAttention(current, previous *models.WeatherData, now time.Time) currentAttention {
	var result currentAttention
	if current == nil || current.Time.IsZero() || current.Time.After(now) {
		result.Notice = "Нет актуальных измерений станции"
		return result
	}
	age := now.Sub(current.Time)
	if age >= 10*time.Minute {
		result.Notice = fmt.Sprintf("Последние измерения %d мин назад. Текущие условия могут отличаться.", int(age.Minutes()))
		return result
	}
	if current.TempOutdoor != nil {
		t := *current.TempOutdoor
		switch {
		case t >= 35:
			result.Temperature = metricAttention{"heat", "Очень жарко", "Выбирайте тень, берите с собой воду", true, "danger"}
		case t >= 30:
			result.Temperature = metricAttention{"heat", "Жарко", "", false, "warning"}
		case t <= -10:
			result.Temperature = metricAttention{"cold", "Сильный мороз", "Одевайтесь теплее", true, "danger"}
		case t <= 0:
			result.Temperature = metricAttention{"cold", "Холодно", "Температура на уровне нуля или ниже", false, "warning"}
		}
	}
	wind := float32(0)
	if current.WindSpeed != nil {
		wind = *current.WindSpeed
	}
	if current.WindGust != nil && *current.WindGust > wind {
		wind = *current.WindGust
	}
	switch {
	case wind >= 17:
		result.Wind = metricAttention{"wind", "Очень сильный ветер", "Избегайте деревьев и непрочных конструкций", true, "danger"}
	case wind >= 10:
		result.Wind = metricAttention{"wind", "Сильный ветер", "Уберите лёгкие предметы с улицы", true, "warning"}
	case wind >= 5:
		result.Wind = metricAttention{"wind", "Ветрено", "", false, "warning"}
	}
	if current.RainRate != nil {
		rate := *current.RainRate
		switch {
		case rate >= 7.5:
			result.Rain = metricAttention{"rain", "Ливень сейчас", "Лучше переждать сильные осадки", true, "danger"}
		case rate >= 2.5:
			result.Rain = metricAttention{"rain", "Сильный дождь", "Проверьте, закрыты ли окна", true, "warning"}
		case rate >= 0.1:
			result.Rain = metricAttention{"rain", "Идёт дождь", "Возьмите зонт", false, "warning"}
		}
		if result.Rain.Title != "" {
			result.Rain.Detail = fmt.Sprintf("%.1f мм/ч · %s", rate, result.Rain.Detail)
		}
	}
	if current.UVIndex != nil {
		switch {
		case *current.UVIndex >= 8:
			result.Solar = metricAttention{"solar", "Очень высокий UV", "Избегайте прямого солнца, используйте солнцезащиту", true, "danger"}
		case *current.UVIndex >= 6:
			result.Solar = metricAttention{"solar", "Высокий UV", "Выбирайте тень и используйте солнцезащиту", true, "warning"}
		}
	}
	if previous != nil && current.PressureRelative != nil && previous.PressureRelative != nil {
		interval := current.Time.Sub(previous.Time)
		if interval >= 50*time.Minute && interval <= 70*time.Minute {
			change := float64(*current.PressureRelative-*previous.PressureRelative) / interval.Hours()
			if math.Abs(change) >= 1.5 {
				title := "Давление быстро растёт"
				if change < 0 {
					title = "Давление быстро падает"
				}
				severity := "warning"
				if math.Abs(change) >= 3 {
					severity = "danger"
				}
				result.Pressure = metricAttention{"pressure", title, "", math.Abs(change) >= 3, severity}
			}
		}
	}
	count := 0
	for _, signal := range []metricAttention{result.Temperature, result.Wind, result.Rain, result.Solar, result.Pressure} {
		if signal.Important {
			count++
		}
	}
	if count > 1 {
		result.Notice = fmt.Sprintf("Важных сигналов: %d — пояснения в блоках показателей.", count)
	}
	return result
}
