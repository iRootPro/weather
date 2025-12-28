package telegram

import (
	"fmt"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
)

// FormatCurrentWeather форматирует текущую погоду с изменениями за час
func FormatCurrentWeather(current *models.WeatherData, hourAgo *models.WeatherData, dailyMinMax *repository.DailyMinMax) string {
	if current == nil {
		return "❌ Нет данных о погоде"
	}

	// Форматируем дату
	months := []string{"", "января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря"}
	day := current.Time.Day()
	month := months[current.Time.Month()]

	text := fmt.Sprintf("🌦️ *Текущая погода · %d %s*\n\n", day, month)

	// Температура
	if current.TempOutdoor != nil {
		text += fmt.Sprintf("🌡️ *Температура:* %.1f°C", *current.TempOutdoor)

		// Изменение за час
		if hourAgo != nil && hourAgo.TempOutdoor != nil {
			change := *current.TempOutdoor - *hourAgo.TempOutdoor
			if change > 0 {
				text += fmt.Sprintf(" (↗️ +%.1f°C/час)", change)
			} else if change < 0 {
				text += fmt.Sprintf(" (↘️ %.1f°C/час)", change)
			}
		}

		// Мин/Макс за день
		if dailyMinMax != nil && dailyMinMax.TempMin != nil && dailyMinMax.TempMax != nil {
			text += fmt.Sprintf(" · 📊 %.1f...%.1f°C", *dailyMinMax.TempMin, *dailyMinMax.TempMax)
		}
		text += "\n"
	}

	// Ощущается как
	if current.TempFeelsLike != nil {
		text += fmt.Sprintf("🤒 *Ощущается:* %.1f°C\n", *current.TempFeelsLike)
	}

	// Влажность
	if current.HumidityOutdoor != nil {
		text += fmt.Sprintf("💧 *Влажность:* %d%%", *current.HumidityOutdoor)
		if dailyMinMax != nil && dailyMinMax.HumidityMin != nil && dailyMinMax.HumidityMax != nil {
			text += fmt.Sprintf(" (%d...%d%%)", *dailyMinMax.HumidityMin, *dailyMinMax.HumidityMax)
		}
		text += "\n"
	}

	// Точка росы
	if current.DewPoint != nil {
		text += fmt.Sprintf("💦 *Точка росы:* %.1f°C\n", *current.DewPoint)
	}

	// Давление
	if current.PressureRelative != nil {
		text += fmt.Sprintf("🔽 *Давление:* %.0f мм рт.ст.", *current.PressureRelative)

		// Изменение за час
		if hourAgo != nil && hourAgo.PressureRelative != nil {
			change := *current.PressureRelative - *hourAgo.PressureRelative
			if change > 0.5 {
				text += fmt.Sprintf(" (↗️ +%.1f)", change)
			} else if change < -0.5 {
				text += fmt.Sprintf(" (↘️ %.1f)", change)
			}
		}
		text += "\n"
	}

	// Ветер
	if current.WindSpeed != nil || current.WindGust != nil {
		text += "💨 *Ветер:* "
		if current.WindSpeed != nil {
			text += fmt.Sprintf("%.1f м/с", *current.WindSpeed)
		}
		if current.WindGust != nil {
			text += fmt.Sprintf(", порывы до %.1f м/с", *current.WindGust)
		}
		if current.WindDirection != nil {
			direction := getWindDirection(*current.WindDirection)
			text += fmt.Sprintf(", %s", direction)
		}
		text += "\n"
	}

	// Осадки
	if current.RainRate != nil && *current.RainRate > 0 {
		text += fmt.Sprintf("🌧️ *Дождь:* %.1f мм/ч\n", *current.RainRate)
	}
	if current.RainDaily != nil && *current.RainDaily > 0 {
		text += fmt.Sprintf("☔ *За сутки:* %.1f мм\n", *current.RainDaily)
	}

	// UV индекс и солнечная радиация
	if current.UVIndex != nil {
		uvLevel := getUVLevel(*current.UVIndex)
		text += fmt.Sprintf("☀️ *UV индекс:* %.1f (%s)\n", *current.UVIndex, uvLevel)
	}
	if current.SolarRadiation != nil && *current.SolarRadiation > 0 {
		text += fmt.Sprintf("🌞 *Солнечная радиация:* %.0f Вт/м²\n", *current.SolarRadiation)
	}

	// Время обновления
	text += fmt.Sprintf("\n🕐 Обновлено: %s", current.Time.Format("15:04"))

	return text
}

// FormatStats форматирует статистику за период
func FormatStats(stats *models.WeatherStats) string {
	if stats == nil {
		return "❌ Нет данных статистики"
	}

	periodName := map[string]string{
		"day":   "сутки",
		"week":  "неделю",
		"month": "месяц",
		"year":  "год",
	}

	text := fmt.Sprintf("📈 *Статистика за %s*\n\n", periodName[stats.Period])

	// Температура
	if stats.TempOutdoorMin != nil && stats.TempOutdoorMax != nil {
		text += fmt.Sprintf("🌡️ *Температура:*\n")
		text += fmt.Sprintf("   Мин: %.1f°C\n", *stats.TempOutdoorMin)
		text += fmt.Sprintf("   Макс: %.1f°C\n", *stats.TempOutdoorMax)
		if stats.TempOutdoorAvg != nil {
			text += fmt.Sprintf("   Средняя: %.1f°C\n", *stats.TempOutdoorAvg)
		}
		text += "\n"
	}

	// Влажность
	if stats.HumidityOutdoorMin != nil && stats.HumidityOutdoorMax != nil {
		text += fmt.Sprintf("💧 *Влажность:*\n")
		text += fmt.Sprintf("   Мин: %d%%\n", *stats.HumidityOutdoorMin)
		text += fmt.Sprintf("   Макс: %d%%\n", *stats.HumidityOutdoorMax)
		if stats.HumidityOutdoorAvg != nil {
			text += fmt.Sprintf("   Средняя: %d%%\n", *stats.HumidityOutdoorAvg)
		}
		text += "\n"
	}

	// Давление
	if stats.PressureRelativeMin != nil && stats.PressureRelativeMax != nil {
		text += fmt.Sprintf("🔽 *Давление:*\n")
		text += fmt.Sprintf("   Мин: %.0f мм рт.ст.\n", *stats.PressureRelativeMin)
		text += fmt.Sprintf("   Макс: %.0f мм рт.ст.\n", *stats.PressureRelativeMax)
		if stats.PressureRelativeAvg != nil {
			text += fmt.Sprintf("   Среднее: %.0f мм рт.ст.\n", *stats.PressureRelativeAvg)
		}
		text += "\n"
	}

	// Ветер
	if stats.WindSpeedMax != nil || stats.WindGustMax != nil {
		text += "💨 *Ветер:*\n"
		if stats.WindSpeedMax != nil {
			text += fmt.Sprintf("   Макс скорость: %.1f м/с\n", *stats.WindSpeedMax)
		}
		if stats.WindGustMax != nil {
			text += fmt.Sprintf("   Макс порыв: %.1f м/с\n", *stats.WindGustMax)
		}
		text += "\n"
	}

	// Осадки
	if stats.RainTotal != nil && *stats.RainTotal > 0 {
		text += fmt.Sprintf("☔ *Осадки:* %.1f мм\n\n", *stats.RainTotal)
	}

	text += fmt.Sprintf("📅 %s — %s",
		stats.StartTime.Format("02.01 15:04"),
		stats.EndTime.Format("02.01 15:04"))

	return text
}

// FormatRecords форматирует рекорды за всё время
func FormatRecords(records *models.WeatherRecords) string {
	if records == nil {
		return "❌ Нет данных о рекордах"
	}

	text := "🏆 *Рекорды за всё время*\n\n"

	// Температура
	text += "🌡️ *Температура:*\n"
	text += fmt.Sprintf("   ❄️ Мин: %.1f°C (%s)\n",
		records.TempOutdoorMin.Value,
		records.TempOutdoorMin.Time.Format("02.01.2006"))
	text += fmt.Sprintf("   🔥 Макс: %.1f°C (%s)\n\n",
		records.TempOutdoorMax.Value,
		records.TempOutdoorMax.Time.Format("02.01.2006"))

	// Влажность
	text += "💧 *Влажность:*\n"
	text += fmt.Sprintf("   Мин: %.0f%% (%s)\n",
		records.HumidityOutdoorMin.Value,
		records.HumidityOutdoorMin.Time.Format("02.01.2006"))
	text += fmt.Sprintf("   Макс: %.0f%% (%s)\n\n",
		records.HumidityOutdoorMax.Value,
		records.HumidityOutdoorMax.Time.Format("02.01.2006"))

	// Давление
	text += "🔽 *Давление:*\n"
	text += fmt.Sprintf("   Мин: %.0f мм (%s)\n",
		records.PressureMin.Value,
		records.PressureMin.Time.Format("02.01.2006"))
	text += fmt.Sprintf("   Макс: %.0f мм (%s)\n\n",
		records.PressureMax.Value,
		records.PressureMax.Time.Format("02.01.2006"))

	// Ветер
	text += "💨 *Ветер:*\n"
	if records.WindSpeedMax.Value > 0 {
		text += fmt.Sprintf("   Скорость: %.1f м/с (%s)\n",
			records.WindSpeedMax.Value,
			records.WindSpeedMax.Time.Format("02.01.2006"))
	}
	text += fmt.Sprintf("   Порыв: %.1f м/с (%s)\n\n",
		records.WindGustMax.Value,
		records.WindGustMax.Time.Format("02.01.2006"))

	// Осадки
	if records.RainDailyMax.Value > 0 {
		text += fmt.Sprintf("☔ *Макс осадки за день:* %.1f мм (%s)\n\n",
			records.RainDailyMax.Value,
			records.RainDailyMax.Time.Format("02.01.2006"))
	}

	// Солнце
	if records.SolarRadiationMax.Value > 0 {
		text += fmt.Sprintf("🌞 *Макс солнечная радиация:* %.0f Вт/м² (%s)\n",
			records.SolarRadiationMax.Value,
			records.SolarRadiationMax.Time.Format("02.01.2006"))
	}
	if records.UVIndexMax.Value > 0 {
		text += fmt.Sprintf("☀️ *Макс UV индекс:* %.1f (%s)\n\n",
			records.UVIndexMax.Value,
			records.UVIndexMax.Time.Format("02.01.2006"))
	}

	text += fmt.Sprintf("📊 Данные с %s (%d дней)",
		records.FirstRecord.Format("02.01.2006"),
		records.TotalDays)

	return text
}

// FormatSunData форматирует данные о солнце
func FormatSunData(sunData *service.SunTimesWithComparison) string {
	if sunData == nil {
		return "❌ Нет данных о солнце"
	}

	text := "☀️ *Солнце*\n\n"

	text += fmt.Sprintf("🌅 *Восход:* %s\n", sunData.Sunrise.Format("15:04"))
	text += fmt.Sprintf("🌇 *Закат:* %s\n\n", sunData.Sunset.Format("15:04"))

	// Продолжительность дня
	dayHours := int(sunData.DayLength.Hours())
	dayMinutes := int(sunData.DayLength.Minutes()) % 60
	text += fmt.Sprintf("⏱️ *Световой день:* %dч %dм\n", dayHours, dayMinutes)

	// Изменения по сравнению с вчера
	if sunData.DayChangeDay != 0 {
		change := formatDurationChange(sunData.DayChangeDay)
		if sunData.DayChangeDay > 0 {
			text += fmt.Sprintf("   По сравнению с вчера: ↗️ +%s\n", change)
		} else {
			text += fmt.Sprintf("   По сравнению с вчера: ↘️ %s\n", change)
		}
	}

	// Изменения за неделю
	if sunData.DayChangeWeek != 0 {
		change := formatDurationChange(sunData.DayChangeWeek)
		if sunData.DayChangeWeek > 0 {
			text += fmt.Sprintf("   За неделю: ↗️ +%s\n", change)
		} else {
			text += fmt.Sprintf("   За неделю: ↘️ %s\n", change)
		}
	}

	// Сумерки
	text += fmt.Sprintf("\n🌄 *Рассвет:* %s\n", sunData.Dawn.Format("15:04"))
	text += fmt.Sprintf("🌆 *Сумерки:* %s\n", sunData.Dusk.Format("15:04"))

	return text
}

// FormatMoonData форматирует данные о луне
func FormatMoonData(moonData *service.MoonData) string {
	if moonData == nil {
		return "❌ Нет данных о луне"
	}

	text := fmt.Sprintf("🌙 *Луна*\n\n")

	text += fmt.Sprintf("%s *%s*\n", moonData.PhaseIcon, moonData.PhaseName)
	text += fmt.Sprintf("💡 *Освещённость:* %.0f%%\n", moonData.Illumination)
	text += fmt.Sprintf("📅 *Возраст луны:* %.1f дней\n\n", moonData.Age)

	text += fmt.Sprintf("🌔 *Восход луны:* %s\n", moonData.Moonrise.Format("15:04"))
	text += fmt.Sprintf("🌖 *Заход луны:* %s\n\n", moonData.Moonset.Format("15:04"))

	if moonData.IsAboveHorizon {
		text += "✅ Луна над горизонтом"
	} else {
		text += "❌ Луна под горизонтом"
	}

	return text
}

// FormatEventNotification форматирует уведомление о погодном событии
func FormatEventNotification(event models.WeatherEvent) string {
	text := fmt.Sprintf("%s *%s*\n", event.Icon, event.Description)

	// Детали события
	if event.Details != "" {
		text += fmt.Sprintf("%s\n", event.Details)
	}

	// Текущее значение
	if event.Value != 0 && event.Type == "wind_gust" {
		text += fmt.Sprintf("Скорость: %.1f м/с\n", event.Value)
	}

	// Время события
	text += fmt.Sprintf("\n🕐 %s", event.Time.Format("15:04"))

	return text
}

// GetEventTypeName возвращает название типа события на русском
func GetEventTypeName(eventType string) string {
	names := map[string]string{
		"all":         "Все события",
		"rain":        "Дождь",
		"temperature": "Изменения температуры",
		"wind":        "Сильный ветер",
		"pressure":    "Изменения давления",
	}
	if name, ok := names[eventType]; ok {
		return name
	}
	return eventType
}

// getWindDirection возвращает направление ветра по градусам
func getWindDirection(degrees int16) string {
	directions := []string{"Север", "Северо-Восток", "Восток", "Юго-Восток",
		"Юг", "Юго-Запад", "Запад", "Северо-Запад"}
	index := int((float64(degrees) + 22.5) / 45.0)
	return directions[index%8]
}

// getUVLevel возвращает уровень UV индекса
func getUVLevel(uv float32) string {
	switch {
	case uv < 3:
		return "низкий"
	case uv < 6:
		return "умеренный"
	case uv < 8:
		return "высокий"
	case uv < 11:
		return "очень высокий"
	default:
		return "экстремальный"
	}
}

// formatDurationChange форматирует изменение длительности
func formatDurationChange(d time.Duration) string {
	totalMinutes := int(d.Minutes())
	if totalMinutes < 0 {
		totalMinutes = -totalMinutes
	}

	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dч %dм", hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dч", hours)
	}
	return fmt.Sprintf("%dм", minutes)
}
