package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/iRootPro/weather/internal/models"
)

var (
	eventsBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			MarginTop(1).
			Height(30)

	eventsTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textPrimary).
			MarginBottom(1)

	eventItem = lipgloss.NewStyle().
			BorderLeft(true).
			BorderForeground(textSecondary).
			PaddingLeft(2).
			MarginBottom(1)

	eventIcon = lipgloss.NewStyle().
			Width(3)

	eventDescription = lipgloss.NewStyle().
			Bold(true).
			Foreground(textPrimary)

	eventDetails = lipgloss.NewStyle().
			Foreground(textSecondary)

	eventTime = lipgloss.NewStyle().
			Foreground(textSecondary).
			Italic(true)
)

// RenderEvents renders the events view
func RenderEvents(events []models.WeatherEvent, width int) string {
	var content string

	// Title
	content += eventsTitle.Render("🔔  ПОГОДНЫЕ СОБЫТИЯ (24 часа)") + "\n\n"

	if len(events) == 0 {
		content += secondaryTextStyle.Render("Значимых погодных событий не зафиксировано")
		return eventsBox.Width(width).Render(content)
	}

	// Render each event
	for i, event := range events {
		if i > 0 {
			content += "\n"
		}

		// Icon and description
		eventLine := eventIcon.Render(event.Icon) + " " +
			eventDescription.Render(event.Description)

		// Details if available
		if event.Details != "" {
			eventLine += "\n   " + eventDetails.Render(event.Details)
		}

		// Time
		months := []string{"", "января", "февраля", "марта", "апреля", "мая", "июня",
			"июля", "августа", "сентября", "октября", "ноября", "декабря"}
		timeStr := fmt.Sprintf("%s, %d %s", event.Time.Format("15:04"), event.Time.Day(), months[event.Time.Month()])
		eventLine += "\n   " + eventTime.Render(timeStr)

		// Style the event item
		styledEvent := eventItem.Render(eventLine)

		// Add color based on event type
		switch event.Type {
		case "temp_rise":
			styledEvent = lipgloss.NewStyle().
				BorderLeft(true).
				BorderForeground(warmColor).
				PaddingLeft(2).
				MarginBottom(1).
				Render(eventLine)
		case "temp_drop":
			styledEvent = lipgloss.NewStyle().
				BorderLeft(true).
				BorderForeground(coldColor).
				PaddingLeft(2).
				MarginBottom(1).
				Render(eventLine)
		case "rain_start", "rain_end":
			styledEvent = lipgloss.NewStyle().
				BorderLeft(true).
				BorderForeground(primaryColor).
				PaddingLeft(2).
				MarginBottom(1).
				Render(eventLine)
		case "wind_gust":
			styledEvent = lipgloss.NewStyle().
				BorderLeft(true).
				BorderForeground(warmColor).
				PaddingLeft(2).
				MarginBottom(1).
				Render(eventLine)
		case "pressure_rise", "pressure_drop":
			styledEvent = lipgloss.NewStyle().
				BorderLeft(true).
				BorderForeground(primaryColor).
				PaddingLeft(2).
				MarginBottom(1).
				Render(eventLine)
		}

		content += styledEvent
	}

	// Summary
	content += "\n" + secondaryTextStyle.Render(fmt.Sprintf("Всего событий: %d", len(events)))

	return eventsBox.Width(width).Render(content)
}
