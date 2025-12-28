package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/guptarohit/asciigraph"
	"github.com/iRootPro/weather/internal/models"
)

var (
	chartBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			MarginTop(1).
			Height(30)

	chartTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textPrimary).
			MarginBottom(1)

	chartSubtitle = lipgloss.NewStyle().
			Foreground(textSecondary).
			MarginBottom(1)

	activeMetric = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	inactiveMetric = lipgloss.NewStyle().
			Foreground(textSecondary)

	secondaryTextStyle = lipgloss.NewStyle().
				Foreground(textSecondary)
)

// RenderCharts renders the charts view
func RenderCharts(data []models.WeatherData, period string, metric int, width int) string {
	if len(data) == 0 {
		return chartBox.Width(width).Render(
			chartTitle.Render("📈  ГРАФИКИ") + "\n\n" +
				secondaryTextStyle.Render("Нет данных для отображения"),
		)
	}

	var content string

	// Title with period selector
	periodText := map[string]string{
		"24h": "последние 24 часа",
		"7d":  "последние 7 дней",
		"30d": "последние 30 дней",
	}[period]

	content += chartTitle.Render("📈  ГРАФИКИ") + "\n"
	content += chartSubtitle.Render(fmt.Sprintf("Период: %s", periodText)) + "\n\n"

	// Metric selector
	metrics := []string{"Температура", "Давление", "Ветер", "Влажность"}
	var metricTabs []string
	for i, m := range metrics {
		style := inactiveMetric
		if i == metric {
			style = activeMetric
		}
		metricTabs = append(metricTabs, style.Render(m))
	}

	// Join with separator
	separator := "  "
	var tabsWithSeparators []string
	for i, tab := range metricTabs {
		tabsWithSeparators = append(tabsWithSeparators, tab)
		if i < len(metricTabs)-1 {
			tabsWithSeparators = append(tabsWithSeparators, separator)
		}
	}
	content += lipgloss.JoinHorizontal(lipgloss.Left, tabsWithSeparators...) + "\n\n"

	// Render selected metric chart
	switch metric {
	case 0:
		content += renderTemperatureChart(data, width)
	case 1:
		content += renderPressureChart(data, width)
	case 2:
		content += renderWindChart(data, width)
	case 3:
		content += renderHumidityChart(data, width)
	}

	return chartBox.Width(width).Render(content)
}

// renderTemperatureChart renders temperature ASCII chart
func renderTemperatureChart(data []models.WeatherData, width int) string {
	values := make([]float64, 0, len(data))
	for _, d := range data {
		if d.TempOutdoor != nil {
			values = append(values, float64(*d.TempOutdoor))
		} else {
			values = append(values, 0)
		}
	}

	if len(values) == 0 {
		return secondaryTextStyle.Render("Нет данных о температуре")
	}

	// Calculate stats
	min, max, avg := calculateStats(values)

	graph := asciigraph.Plot(values,
		asciigraph.Height(12),
		asciigraph.Width(width-20),
		asciigraph.Caption(fmt.Sprintf("Температура °C  |  Мин: %.1f  Макс: %.1f  Сред: %.1f", min, max, avg)),
	)

	// Add time labels
	timeLabels := generateTimeLabels(data, 5)

	return graph + "\n" + timeLabels
}

// renderPressureChart renders pressure ASCII chart
func renderPressureChart(data []models.WeatherData, width int) string {
	values := make([]float64, 0, len(data))
	for _, d := range data {
		if d.PressureRelative != nil {
			values = append(values, float64(*d.PressureRelative))
		} else {
			values = append(values, 0)
		}
	}

	if len(values) == 0 {
		return secondaryTextStyle.Render("Нет данных о давлении")
	}

	min, max, avg := calculateStats(values)

	graph := asciigraph.Plot(values,
		asciigraph.Height(12),
		asciigraph.Width(width-20),
		asciigraph.Caption(fmt.Sprintf("Давление мм рт.ст.  |  Мин: %.0f  Макс: %.0f  Сред: %.0f", min, max, avg)),
	)

	timeLabels := generateTimeLabels(data, 5)

	return graph + "\n" + timeLabels
}

// renderWindChart renders wind speed ASCII chart
func renderWindChart(data []models.WeatherData, width int) string {
	values := make([]float64, 0, len(data))
	gusts := make([]float64, 0, len(data))

	for _, d := range data {
		if d.WindSpeed != nil {
			values = append(values, float64(*d.WindSpeed))
		} else {
			values = append(values, 0)
		}

		if d.WindGust != nil {
			gusts = append(gusts, float64(*d.WindGust))
		} else {
			gusts = append(gusts, 0)
		}
	}

	if len(values) == 0 {
		return secondaryTextStyle.Render("Нет данных о ветре")
	}

	min, max, avg := calculateStats(values)
	gustMax := max
	if len(gusts) > 0 {
		_, gustMax, _ = calculateStats(gusts)
	}

	graph := asciigraph.Plot(values,
		asciigraph.Height(12),
		asciigraph.Width(width-20),
		asciigraph.Caption(fmt.Sprintf("Скорость ветра м/с  |  Мин: %.1f  Макс: %.1f  Сред: %.1f  Порывы до: %.1f", min, max, avg, gustMax)),
	)

	timeLabels := generateTimeLabels(data, 5)

	return graph + "\n" + timeLabels
}

// renderHumidityChart renders humidity ASCII chart
func renderHumidityChart(data []models.WeatherData, width int) string {
	values := make([]float64, 0, len(data))
	for _, d := range data {
		if d.HumidityOutdoor != nil {
			values = append(values, float64(*d.HumidityOutdoor))
		} else {
			values = append(values, 0)
		}
	}

	if len(values) == 0 {
		return secondaryTextStyle.Render("Нет данных о влажности")
	}

	min, max, avg := calculateStats(values)

	graph := asciigraph.Plot(values,
		asciigraph.Height(12),
		asciigraph.Width(width-20),
		asciigraph.Caption(fmt.Sprintf("Влажность %%  |  Мин: %.0f  Макс: %.0f  Сред: %.0f", min, max, avg)),
	)

	timeLabels := generateTimeLabels(data, 5)

	return graph + "\n" + timeLabels
}

// calculateStats calculates min, max, and average of values
func calculateStats(values []float64) (min, max, avg float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}

	min = values[0]
	max = values[0]
	sum := 0.0

	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}

	avg = sum / float64(len(values))
	return min, max, avg
}

// generateTimeLabels generates time labels for the chart
func generateTimeLabels(data []models.WeatherData, count int) string {
	if len(data) == 0 {
		return ""
	}

	// Protect against invalid count values
	if count <= 1 {
		count = 2
	}

	// Generate evenly spaced labels
	step := len(data) / (count - 1)
	if step == 0 {
		step = 1
	}

	var labels []string
	for i := 0; i < count && i*step < len(data); i++ {
		idx := i * step
		if idx >= len(data) {
			idx = len(data) - 1
		}
		t := data[idx].Time
		var format string
		if t.Hour() == 0 && t.Minute() == 0 {
			format = "02.01"
		} else {
			format = "15:04"
		}
		labels = append(labels, t.Format(format))
	}

	// Ensure we have exactly count labels by padding if necessary
	for len(labels) < count {
		labels = append(labels, "")
	}

	// Pad labels to align with chart width
	spacing := "          "
	return secondaryTextStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left, labels[0], spacing+labels[1], spacing+labels[2], spacing+labels[3], spacing+labels[4]))
}

// formatTime formats time based on the period
func formatTime(t time.Time, period string) string {
	switch period {
	case "24h":
		return t.Format("15:04")
	case "7d", "30d":
		return t.Format("02.01")
	default:
		return t.Format("15:04")
	}
}
