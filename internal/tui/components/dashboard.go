package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

var (
	// Temperature colors
	coldColor = lipgloss.Color("#3b82f6")  // Blue
	coolColor = lipgloss.Color("#10b981")  // Green
	warmColor = lipgloss.Color("#f59e0b")  // Yellow
	hotColor  = lipgloss.Color("#ef4444")  // Red

	// Text colors
	textPrimary   = lipgloss.Color("#ffffff")
	textSecondary = lipgloss.Color("#9ca3af")
	primaryColor  = lipgloss.Color("#3b82f6")

	// Styles
	dashboardBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			MarginTop(1).
			Height(30)

	sectionTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textPrimary).
			MarginBottom(1)

	metricLabel = lipgloss.NewStyle().
			Foreground(textSecondary).
			Width(20)

	metricValue = lipgloss.NewStyle().
			Bold(true).
			Foreground(textPrimary)

	secondaryText = lipgloss.NewStyle().
			Foreground(textSecondary)
)

// RenderDashboard renders the dashboard view
func RenderDashboard(
	current *models.WeatherData,
	hourAgo *models.WeatherData,
	dailyMinMax *repository.DailyMinMax,
	sunrise, sunset time.Time,
	dayLength time.Duration,
	loading bool,
	spin spinner.Model,
	width int,
) string {
	if loading || current == nil {
		return dashboardBox.Render(
			sectionTitle.Render("⏳ Загрузка данных...") + "\n\n" +
				spin.View(),
		)
	}

	var content string

	// Title with data timestamp
	now := time.Now()
	dataAge := now.Sub(current.Time)
	timeStr := current.Time.Format("15:04:05")

	ageStr := ""
	if dataAge > time.Minute {
		minutes := int(dataAge.Minutes())
		if minutes >= 60 {
			hours := minutes / 60
			ageStr = fmt.Sprintf(" ⚠️  устарело на %dч %dм", hours, minutes%60)
		} else {
			ageStr = fmt.Sprintf(" ⚠️  устарело на %dм", minutes)
		}
	}

	content += sectionTitle.Render("🌡️  ТЕКУЩАЯ ПОГОДА") + " " +
		secondaryText.Render(fmt.Sprintf("(данные на %s%s)", timeStr, ageStr)) + "\n\n"

	// Temperature
	if current.TempOutdoor != nil {
		temp := float64(*current.TempOutdoor)
		tempStr := fmt.Sprintf("%.1f°C", temp)
		tempStyle := getTempStyle(temp)

		change := ""
		if hourAgo != nil && hourAgo.TempOutdoor != nil {
			diff := *current.TempOutdoor - *hourAgo.TempOutdoor
			if diff != 0 {
				changeStyle := getChangeStyle(float64(diff))
				icon := getChangeIcon(float64(diff))
				change = changeStyle.Render(fmt.Sprintf("  (%s %.1f°C за час)", icon, abs(diff)))
			}
		}

		// Daily min/max
		minMax := ""
		if dailyMinMax != nil && dailyMinMax.TempMin != nil && dailyMinMax.TempMax != nil {
			minMax = secondaryText.Render(fmt.Sprintf("  [мин %.1f°C, макс %.1f°C за сутки]",
				*dailyMinMax.TempMin, *dailyMinMax.TempMax))
		}

		content += fmt.Sprintf("%s%s%s%s\n",
			metricLabel.Render("🌡️  Температура"),
			tempStyle.Render(tempStr),
			change,
			minMax,
		)
	}

	// Humidity
	if current.HumidityOutdoor != nil {
		hum := int(*current.HumidityOutdoor)
		humStr := fmt.Sprintf("%d%%", hum)

		change := ""
		if hourAgo != nil && hourAgo.HumidityOutdoor != nil {
			diff := *current.HumidityOutdoor - *hourAgo.HumidityOutdoor
			if diff != 0 {
				changeStyle := getChangeStyle(float64(diff))
				icon := getChangeIcon(float64(diff))
				change = changeStyle.Render(fmt.Sprintf("  (%s %d%% за час)", icon, abs16(diff)))
			}
		}

		content += fmt.Sprintf("%s%s%s\n",
			metricLabel.Render("💧  Влажность"),
			metricValue.Render(humStr),
			change,
		)
	}

	// Pressure
	if current.PressureRelative != nil {
		press := float64(*current.PressureRelative)
		pressStr := fmt.Sprintf("%.0f мм", press)

		change := ""
		if hourAgo != nil && hourAgo.PressureRelative != nil {
			diff := *current.PressureRelative - *hourAgo.PressureRelative
			if abs(diff) >= 1 {
				changeStyle := getChangeStyle(float64(diff))
				icon := getChangeIcon(float64(diff))
				change = changeStyle.Render(fmt.Sprintf("  (%s %.0f мм за час)", icon, abs(diff)))
			}
		}

		content += fmt.Sprintf("%s%s%s\n",
			metricLabel.Render("📊  Давление"),
			metricValue.Render(pressStr),
			change,
		)
	}

	// Wind
	if current.WindSpeed != nil {
		wind := float64(*current.WindSpeed)
		windStr := fmt.Sprintf("%.1f м/с", wind)

		// Wind direction
		if current.WindDirection != nil {
			windStr += fmt.Sprintf(" (%s)", getWindDirection(*current.WindDirection))
		}

		// Wind gust
		gust := ""
		if current.WindGust != nil && *current.WindGust > *current.WindSpeed {
			gust = secondaryText.Render(fmt.Sprintf("  (порывы %.1f м/с)", *current.WindGust))
		}

		content += fmt.Sprintf("%s%s%s\n",
			metricLabel.Render("💨  Ветер"),
			metricValue.Render(windStr),
			gust,
		)
	}

	// UV Index
	if current.UVIndex != nil && *current.UVIndex > 0 {
		uv := float64(*current.UVIndex)
		uvStr := fmt.Sprintf("%.0f", uv)
		uvLevel := getUVLevel(uv)

		content += fmt.Sprintf("%s%s  %s\n",
			metricLabel.Render("☀️  UV индекс"),
			metricValue.Render(uvStr),
			secondaryText.Render(uvLevel),
		)
	}

	// Rain
	if current.RainRate != nil {
		rain := float64(*current.RainRate)
		rainStr := fmt.Sprintf("%.1f мм/ч", rain)

		daily := ""
		if current.RainDaily != nil {
			daily = secondaryText.Render(fmt.Sprintf("  (за сутки %.1f мм)", *current.RainDaily))
		}

		content += fmt.Sprintf("%s%s%s\n",
			metricLabel.Render("🌧️  Дождь"),
			metricValue.Render(rainStr),
			daily,
		)
	}

	// Sun times
	content += "\n" + renderSunTimes(sunrise, sunset, dayLength) + "\n"

	return dashboardBox.Width(width).Render(content)
}

// renderSunTimes renders sun rise/set information
func renderSunTimes(sunrise, sunset time.Time, dayLength time.Duration) string {
	if sunrise.IsZero() || sunset.IsZero() {
		return ""
	}

	sunriseStr := sunrise.Format("15:04")
	sunsetStr := sunset.Format("15:04")
	hours := int(dayLength.Hours())
	minutes := int(dayLength.Minutes()) % 60
	lengthStr := fmt.Sprintf("%dч %dмин", hours, minutes)

	return fmt.Sprintf("%s%s  |  %s%s  |  %s%s",
		secondaryText.Render("Восход: "),
		metricValue.Render(sunriseStr),
		secondaryText.Render("Закат: "),
		metricValue.Render(sunsetStr),
		secondaryText.Render("День: "),
		metricValue.Render(lengthStr),
	)
}

// Helper functions

func getTempStyle(temp float64) lipgloss.Style {
	var color lipgloss.Color
	switch {
	case temp < 0:
		color = coldColor
	case temp < 15:
		color = coolColor
	case temp < 25:
		color = warmColor
	default:
		color = hotColor
	}
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

func getChangeStyle(change float64) lipgloss.Style {
	var color lipgloss.Color
	switch {
	case change > 0:
		color = warmColor
	case change < 0:
		color = primaryColor
	default:
		color = textSecondary
	}
	return lipgloss.NewStyle().Foreground(color)
}

func getChangeIcon(change float64) string {
	switch {
	case change > 0:
		return "↑"
	case change < 0:
		return "↓"
	default:
		return "→"
	}
}

func getWindDirection(deg int16) string {
	directions := []string{"С", "СВ", "В", "ЮВ", "Ю", "ЮЗ", "З", "СЗ"}
	idx := int((float64(deg) + 22.5) / 45.0)
	if idx >= 8 {
		idx = 0
	}
	return directions[idx]
}

func getUVLevel(uv float64) string {
	switch {
	case uv < 3:
		return "низкий"
	case uv < 6:
		return "средний"
	case uv < 8:
		return "высокий"
	case uv < 11:
		return "очень высокий"
	default:
		return "экстремальный"
	}
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func abs16(x int16) int16 {
	if x < 0 {
		return -x
	}
	return x
}
