package maxbot

const (
	CmdStart       = "start"
	CmdHelp        = "help"
	CmdWeather     = "weather"
	CmdCurrent     = "current"
	CmdSubscribe   = "subscribe"
	CmdUnsubscribe = "unsubscribe"
)

const (
	EventAll          = "all"
	EventRain         = "rain"
	EventTemperature  = "temperature"
	EventWind         = "wind"
	EventPressure     = "pressure"
	EventDailySummary = "daily_summary"
)

func subscriptionTypeForWeatherEvent(eventType string) string {
	switch eventType {
	case "rain_start", "rain_end":
		return EventRain
	case "temp_rise", "temp_drop":
		return EventTemperature
	case "wind_gust":
		return EventWind
	case "pressure_rise", "pressure_drop":
		return EventPressure
	default:
		return ""
	}
}
