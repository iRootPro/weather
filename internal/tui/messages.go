package tui

import (
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

// tickMsg is sent on every tick for auto-refresh
type tickMsg time.Time

// weatherDataMsg is sent when weather data is fetched
type weatherDataMsg struct {
	current     *models.WeatherData
	hourAgo     *models.WeatherData
	dailyMinMax *repository.DailyMinMax
}

// eventsDataMsg is sent when weather events are fetched
type eventsDataMsg struct {
	events []models.WeatherEvent
}

// chartDataMsg is sent when chart data is fetched
type chartDataMsg struct {
	data []models.WeatherData
}

// sunDataMsg is sent when sun times are fetched
type sunDataMsg struct {
	sunrise    time.Time
	sunset     time.Time
	dayLength  time.Duration
	civilDawn  time.Time
	civilDusk  time.Time
	nautical   bool
}

// errMsg is sent when an error occurs
type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}
