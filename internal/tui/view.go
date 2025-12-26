package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/iRootPro/weather/internal/tui/components"
)

// View renders the TUI
func (m Model) View() string {
	if !m.ready {
		return "\n  Инициализация..."
	}

	var content string

	// Header with tabs
	header := m.renderHeader()

	// Content based on active tab
	width := m.width - 4
	switch m.activeTab {
	case TabDashboard:
		content = components.RenderDashboard(m.currentWeather, m.hourAgoWeather, m.dailyMinMax, m.sunrise, m.sunset, m.dayLength, m.loading, m.spinner, width)
	case TabCharts:
		content = components.RenderCharts(m.chartData, m.chartPeriod, m.chartMetric, width)
	case TabEvents:
		content = components.RenderEvents(m.events, width)
	case TabHelp:
		content = components.RenderHelp(m.currentWeather, width)
	}

	// Error message if present
	if m.err != nil {
		content = errorStyle.Render(fmt.Sprintf("❌ Ошибка: %v", m.err))
	}

	// Footer with help
	footer := m.renderFooter()

	// Combine all parts
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

// renderHeader renders the header with tabs
func (m Model) renderHeader() string {
	tabs := []string{"Дашборд", "Графики", "События", "Справка"}
	var renderedTabs []string

	for i, tab := range tabs {
		var style lipgloss.Style
		if i == m.activeTab {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		renderedTabs = append(renderedTabs, style.Render(tab))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	gap := tabGapStyle.Render(strings.Repeat(" ", max(0, m.width-lipgloss.Width(row))))

	return lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
}

// renderFooter renders the footer with help text
func (m Model) renderFooter() string {
	var help string

	switch m.activeTab {
	case TabDashboard:
		help = "Tab: переключить вкладку | 1-4: прямой переход | r: обновить | q: выход"
	case TabCharts:
		help = "←/→ или h/l: период | ↑/↓ или k/j: метрика | Tab: вкладки | r: обновить | q: выход"
	case TabEvents:
		help = "Tab: переключить вкладку | 1-4: прямой переход | r: обновить | q: выход"
	case TabHelp:
		help = "Tab: переключить вкладку | 1-4: прямой переход | r: обновить | q: выход"
	}

	// Add last update time
	if !m.lastUpdate.IsZero() {
		updateTime := m.lastUpdate.Format("15:04:05")
		help += fmt.Sprintf(" | Обновлено: %s", updateTime)
	}

	return helpStyle.Render(help)
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
