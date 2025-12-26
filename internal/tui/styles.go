package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	// Temperature colors
	coldColor   = lipgloss.Color("#3b82f6")  // Blue
	coolColor   = lipgloss.Color("#10b981")  // Green
	warmColor   = lipgloss.Color("#f59e0b")  // Yellow
	hotColor    = lipgloss.Color("#ef4444")  // Red

	// General colors
	primaryColor   = lipgloss.Color("#3b82f6")  // Blue
	secondaryColor = lipgloss.Color("#6b7280")  // Gray
	accentColor    = lipgloss.Color("#8b5cf6")  // Purple
	successColor   = lipgloss.Color("#10b981")  // Green
	warningColor   = lipgloss.Color("#f59e0b")  // Orange
	errorColor     = lipgloss.Color("#ef4444")  // Red

	// Text colors
	textPrimary   = lipgloss.Color("#ffffff")  // White
	textSecondary = lipgloss.Color("#9ca3af")  // Light gray
	textMuted     = lipgloss.Color("#6b7280")  // Gray

	// Background colors
	bgPrimary   = lipgloss.Color("#1f2937")  // Dark gray
	bgSecondary = lipgloss.Color("#111827")  // Darker gray
	bgAccent    = lipgloss.Color("#374151")  // Medium gray
)

// Component styles
var (
	// Header style
	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(textPrimary).
		Background(primaryColor).
		Padding(0, 2).
		MarginBottom(1)

	// Tab styles
	activeTabStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(textPrimary).
		Background(primaryColor).
		Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
		Foreground(textSecondary).
		Background(bgAccent).
		Padding(0, 2)

	tabGapStyle = lipgloss.NewStyle().
		Background(bgSecondary)

	// Box styles
	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2).
		MarginBottom(1)

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(textPrimary).
		MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
		Foreground(textSecondary)

	// Value styles
	valueStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(textPrimary)

	labelStyle = lipgloss.NewStyle().
		Foreground(textSecondary)

	// Status styles
	goodStyle = lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true)

	neutralStyle = lipgloss.NewStyle().
		Foreground(textSecondary)

	badStyle = lipgloss.NewStyle().
		Foreground(warningColor).
		Bold(true)

	// Footer style
	footerStyle = lipgloss.NewStyle().
		Foreground(textMuted).
		MarginTop(1)

	// Help style
	helpStyle = lipgloss.NewStyle().
		Foreground(textSecondary).
		MarginTop(1)

	// Error style
	errorStyle = lipgloss.NewStyle().
		Foreground(errorColor).
		Bold(true)

	// Spinner style
	spinnerStyle = lipgloss.NewStyle().
		Foreground(primaryColor)
)

// GetTempColor returns color based on temperature
func GetTempColor(temp float64) lipgloss.Color {
	switch {
	case temp < 0:
		return coldColor
	case temp < 15:
		return coolColor
	case temp < 25:
		return warmColor
	default:
		return hotColor
	}
}

// GetTempStyle returns style based on temperature
func GetTempStyle(temp float64) lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(GetTempColor(temp))
}

// GetChangeStyle returns style based on value change
func GetChangeStyle(change float64) lipgloss.Style {
	switch {
	case change > 0:
		return lipgloss.NewStyle().Foreground(warningColor)
	case change < 0:
		return lipgloss.NewStyle().Foreground(primaryColor)
	default:
		return lipgloss.NewStyle().Foreground(textSecondary)
	}
}

// GetChangeIcon returns icon based on value change
func GetChangeIcon(change float64) string {
	switch {
	case change > 0:
		return "↑"
	case change < 0:
		return "↓"
	default:
		return "→"
	}
}
