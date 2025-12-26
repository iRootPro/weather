package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/iRootPro/weather/internal/models"
)

var (
	helpBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2).
		MarginTop(1).
		Height(30)

	helpTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textPrimary).
			MarginBottom(1)

	helpSection = lipgloss.NewStyle().
			Foreground(textSecondary).
			MarginTop(1).
			MarginBottom(1)

	batteryOK = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#10b981")) // Green

	batteryWarning = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#f59e0b")) // Yellow

	batteryLow = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ef4444")) // Red
)

// RenderHelp renders the help/info view with system information and battery status
func RenderHelp(current *models.WeatherData, width int) string {
	var content string

	content += helpTitle.Render("ℹ️  СПРАВКА И ИНФОРМАЦИЯ О СИСТЕМЕ") + "\n\n"

	// Station info
	content += sectionTitle.Render("О станции") + "\n"
	content += fmt.Sprintf("%s%s\n", metricLabel.Render("Датчик"), metricValue.Render("Ecowitt WS90 (7-в-1)"))
	content += fmt.Sprintf("%s%s\n", metricLabel.Render("Шлюз"), metricValue.Render("Ecowitt GW3000"))
	content += fmt.Sprintf("%s%s\n", metricLabel.Render("Местоположение"), metricValue.Render("Армавир, Краснодарский край"))
	content += fmt.Sprintf("%s%s\n", metricLabel.Render("Координаты"), metricValue.Render("44.9956° с.ш., 41.1284° в.д."))
	content += fmt.Sprintf("%s%s\n", metricLabel.Render("Часовой пояс"), metricValue.Render("Europe/Moscow (UTC+3)"))
	content += fmt.Sprintf("%s%s\n", metricLabel.Render("Питание датчика"), metricValue.Render("Солнечная панель + 2×AA"))

	// Data collection
	content += "\n" + sectionTitle.Render("Сбор данных") + "\n"
	content += helpSection.Render("• Данные поступают каждые 60 секунд по протоколу MQTT") + "\n"
	content += helpSection.Render("• Хранятся в базе данных TimescaleDB") + "\n"
	content += helpSection.Render("• На графиках данные агрегируются с интервалами 5 мин, 15 мин или 1 час") + "\n"
	content += helpSection.Render("• Текущие показания обновляются автоматически каждую минуту") + "\n"

	// Battery/Power status
	content += "\n" + sectionTitle.Render("Питание датчика") + "\n"

	if current == nil {
		content += secondaryText.Render("Нет данных о питании") + "\n"
	} else {
		// WS90 Capacitor (Solar Panel)
		if current.WS90CapVolt != nil {
			voltage := float64(*current.WS90CapVolt)
			var voltStatus string
			var voltStyle lipgloss.Style

			switch {
			case voltage < 3.0:
				voltStatus = fmt.Sprintf("%.2fV - НИЗКИЙ", voltage)
				voltStyle = batteryLow
			case voltage >= 3.0 && voltage < 4.0:
				voltStatus = fmt.Sprintf("%.2fV - Нормальный", voltage)
				voltStyle = batteryOK
			case voltage >= 4.0:
				voltStatus = fmt.Sprintf("%.2fV - Отличный", voltage)
				voltStyle = batteryOK
			default:
				voltStatus = fmt.Sprintf("%.2fV", voltage)
				voltStyle = batteryWarning
			}

			content += fmt.Sprintf("%s%s\n",
				metricLabel.Render("☀️  Аккумулятор"),
				voltStyle.Render(voltStatus),
			)
			content += secondaryText.Render("   (заряжается от солнечной панели)") + "\n"
		} else {
			content += fmt.Sprintf("%s%s\n",
				metricLabel.Render("☀️  Аккумулятор"),
				secondaryText.Render("Нет данных"),
			)
		}

		// WH65 - AA Batteries
		if current.WH65Batt != nil {
			voltage := float64(*current.WH65Batt)
			var battStatus string
			var battStyle lipgloss.Style

			switch {
			case voltage < 2.4:
				battStatus = fmt.Sprintf("%.2fV - НИЗКИЙ! Требуется замена", voltage)
				battStyle = batteryLow
			case voltage >= 2.4 && voltage < 2.7:
				battStatus = fmt.Sprintf("%.2fV - Средний", voltage)
				battStyle = batteryWarning
			case voltage >= 2.7:
				battStatus = fmt.Sprintf("%.2fV - Хороший", voltage)
				battStyle = batteryOK
			default:
				battStatus = fmt.Sprintf("%.2fV", voltage)
				battStyle = batteryWarning
			}

			content += fmt.Sprintf("%s%s\n",
				metricLabel.Render("🔋  Батарейки 2×AA"),
				battStyle.Render(battStatus),
			)

			if voltage < 2.4 {
				content += secondaryText.Render("   Рекомендуется заменить батарейки для надежной работы ночью") + "\n"
			}
		} else {
			content += fmt.Sprintf("%s%s\n",
				metricLabel.Render("🔋  Батарейки 2×AA"),
				secondaryText.Render("Нет данных"),
			)
		}
	}

	// Keyboard shortcuts
	content += "\n" + sectionTitle.Render("Горячие клавиши") + "\n"
	content += helpSection.Render("Tab          - Переключение между вкладками") + "\n"
	content += helpSection.Render("1, 2, 3, 4   - Прямой переход на вкладку") + "\n"
	content += helpSection.Render("r            - Обновить данные вручную") + "\n"
	content += helpSection.Render("q, Ctrl+C    - Выход из приложения") + "\n"
	content += "\n"
	content += helpSection.Render("На вкладке 'Графики':") + "\n"
	content += helpSection.Render("  ← →  или  h l  - Изменить период (24ч / 7д / 30д)") + "\n"
	content += helpSection.Render("  ↑ ↓  или  k j  - Выбрать метрику") + "\n"

	return helpBox.Width(width).Render(content)
}
