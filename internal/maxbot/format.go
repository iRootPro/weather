package maxbot

import "strings"

// adaptMarkdown converts the most common Telegram-style bold markers used by
// existing weather formatters to Max markdown. Max uses *text* for italics and
// **text** for bold.
func adaptMarkdown(text string) string {
	var b strings.Builder
	inBold := false
	for i := 0; i < len(text); i++ {
		if text[i] == '*' {
			if inBold {
				b.WriteString("**")
			} else {
				b.WriteString("**")
			}
			inBold = !inBold
			continue
		}
		b.WriteByte(text[i])
	}
	return b.String()
}

func textMessage(text string) NewMessageBody {
	return NewMessageBody{Text: adaptMarkdown(text), Format: "markdown"}
}

func GetEventTypeName(eventType string) string {
	names := map[string]string{
		EventAll:          "Все события",
		EventRain:         "Дождь",
		EventTemperature:  "Температура",
		EventWind:         "Ветер",
		EventPressure:     "Давление",
		EventDailySummary: "Утренняя сводка",
	}
	if name, ok := names[eventType]; ok {
		return name
	}
	return eventType
}
