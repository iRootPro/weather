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
}

type currentAttention struct {
	Temperature metricAttention
	Wind        metricAttention
	Rain        metricAttention
	Solar       metricAttention
	Pressure    metricAttention
	Notice      string
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
			result.Temperature = metricAttention{"heat", "Очень жарко", "Выбирайте тень, берите с собой воду", true}
		case t >= 30:
			result.Temperature = metricAttention{"heat", "Жарко", "", false}
		case t <= -10:
			result.Temperature = metricAttention{"cold", "Сильный мороз", "Одевайтесь теплее", true}
		case t <= 0:
			result.Temperature = metricAttention{"cold", "Холодно", "Температура на уровне нуля или ниже", false}
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
		result.Wind = metricAttention{"wind", "Очень сильный ветер", "Избегайте деревьев и непрочных конструкций", true}
	case wind >= 10:
		result.Wind = metricAttention{"wind", "Сильный ветер", "Уберите лёгкие предметы с улицы", true}
	case wind >= 5:
		result.Wind = metricAttention{"wind", "Ветрено", "", false}
	}
	if current.RainRate != nil {
		rate := *current.RainRate
		switch {
		case rate >= 7.5:
			result.Rain = metricAttention{"rain", "Ливень сейчас", "Лучше переждать сильные осадки", true}
		case rate >= 2.5:
			result.Rain = metricAttention{"rain", "Сильный дождь", "Проверьте, закрыты ли окна", true}
		case rate >= 0.1:
			result.Rain = metricAttention{"rain", "Идёт дождь", "Возьмите зонт", false}
		}
		if result.Rain.Title != "" {
			result.Rain.Detail = fmt.Sprintf("%.1f мм/ч · %s", rate, result.Rain.Detail)
		}
	}
	if current.UVIndex != nil {
		switch {
		case *current.UVIndex >= 8:
			result.Solar = metricAttention{"solar", "Очень высокий UV", "Избегайте прямого солнца, используйте солнцезащиту", true}
		case *current.UVIndex >= 6:
			result.Solar = metricAttention{"solar", "Высокий UV", "Выбирайте тень и используйте солнцезащиту", true}
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
				result.Pressure = metricAttention{"pressure", title, "", math.Abs(change) >= 3}
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
