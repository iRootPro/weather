package telegram

// Команды бота
const (
	CmdStart       = "start"
	CmdHelp        = "help"
	CmdWeather     = "weather"
	CmdCurrent     = "current"
	CmdStats       = "stats"
	CmdRecords     = "records"
	CmdHistory     = "history"
	CmdSun         = "sun"
	CmdMoon        = "moon"
	CmdChart       = "chart"
	CmdSubscribe   = "subscribe"
	CmdUnsubscribe = "unsubscribe"
	CmdSettings    = "settings"
	CmdUsers       = "users"        // Админская команда
	CmdMyID        = "myid"         // Показать свой chat_id
	CmdTestSummary = "test_summary" // Админская команда - тест утренней сводки
)

// Типы событий для подписок
const (
	EventAll          = "all"
	EventRain         = "rain"
	EventTemperature  = "temperature"
	EventWind         = "wind"
	EventPressure     = "pressure"
	EventDailySummary = "daily_summary" // Ежедневная утренняя сводка
)
