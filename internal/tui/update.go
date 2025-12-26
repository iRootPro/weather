package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/iRootPro/weather/internal/service"
)

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tick(),
		fetchWeatherData(m.weatherService),
		fetchEventsData(m.weatherService),
		fetchChartData(m.weatherService, m.chartPeriod),
		fetchSunData(m.sunService),
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			m.activeTab = (m.activeTab + 1) % 3
			return m, nil

		case "1":
			m.activeTab = TabDashboard
			return m, nil

		case "2":
			m.activeTab = TabCharts
			return m, nil

		case "3":
			m.activeTab = TabEvents
			return m, nil

		case "r":
			// Manual refresh
			m.loading = true
			return m, tea.Batch(
				fetchWeatherData(m.weatherService),
				fetchEventsData(m.weatherService),
				fetchChartData(m.weatherService, m.chartPeriod),
				fetchSunData(m.sunService),
			)

		case "left", "h":
			if m.activeTab == TabCharts {
				// Cycle chart period backwards
				switch m.chartPeriod {
				case "24h":
					m.chartPeriod = "30d"
				case "7d":
					m.chartPeriod = "24h"
				case "30d":
					m.chartPeriod = "7d"
				}
				return m, fetchChartData(m.weatherService, m.chartPeriod)
			}

		case "right", "l":
			if m.activeTab == TabCharts {
				// Cycle chart period forwards
				switch m.chartPeriod {
				case "24h":
					m.chartPeriod = "7d"
				case "7d":
					m.chartPeriod = "30d"
				case "30d":
					m.chartPeriod = "24h"
				}
				return m, fetchChartData(m.weatherService, m.chartPeriod)
			}

		case "up", "k":
			if m.activeTab == TabCharts {
				// Cycle metric backwards
				m.chartMetric = (m.chartMetric - 1 + 4) % 4
			}

		case "down", "j":
			if m.activeTab == TabCharts {
				// Cycle metric forwards
				m.chartMetric = (m.chartMetric + 1) % 4
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.ready = true
		}
		return m, nil

	case tickMsg:
		// Auto-refresh every 30 seconds
		return m, tea.Batch(
			fetchWeatherData(m.weatherService),
			fetchEventsData(m.weatherService),
			fetchChartData(m.weatherService, m.chartPeriod),
			tick(),
		)

	case weatherDataMsg:
		m.currentWeather = msg.current
		m.hourAgoWeather = msg.hourAgo
		m.dailyMinMax = msg.dailyMinMax
		m.loading = false
		m.lastUpdate = time.Now()
		m.err = nil
		return m, nil

	case eventsDataMsg:
		m.events = msg.events
		return m, nil

	case chartDataMsg:
		m.chartData = msg.data
		return m, nil

	case sunDataMsg:
		m.sunrise = msg.sunrise
		m.sunset = msg.sunset
		m.dayLength = msg.dayLength
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// tick returns a command that sends a tick message every 30 seconds
func tick() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// fetchWeatherData returns a command that fetches current weather data
func fetchWeatherData(svc WeatherServiceInterface) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		current, hourAgo, dailyMinMax, err := svc.GetCurrentWithHourlyChange(ctx)
		if err != nil {
			return errMsg{err: err}
		}

		return weatherDataMsg{
			current:     current,
			hourAgo:     hourAgo,
			dailyMinMax: dailyMinMax,
		}
	}
}

// fetchEventsData returns a command that fetches weather events
func fetchEventsData(svc WeatherServiceInterface) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		events, err := svc.GetRecentEvents(ctx, 24)
		if err != nil {
			return errMsg{err: err}
		}

		return eventsDataMsg{events: events}
	}
}

// fetchChartData returns a command that fetches chart data
func fetchChartData(svc WeatherServiceInterface, period string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		now := time.Now()
		var from time.Time
		var interval string

		switch period {
		case "24h":
			from = now.Add(-24 * time.Hour)
			interval = "5m"
		case "7d":
			from = now.Add(-7 * 24 * time.Hour)
			interval = "1h"
		case "30d":
			from = now.Add(-30 * 24 * time.Hour)
			interval = "6h"
		default:
			from = now.Add(-24 * time.Hour)
			interval = "5m"
		}

		data, err := svc.GetHistory(ctx, from, now, interval)
		if err != nil {
			return errMsg{err: err}
		}

		return chartDataMsg{data: data}
	}
}

// fetchSunData returns a command that fetches sun times
func fetchSunData(svc *service.SunService) tea.Cmd {
	return func() tea.Msg {
		now := time.Now()
		sunTimes := svc.GetSunTimes(now)

		return sunDataMsg{
			sunrise:   sunTimes.Sunrise,
			sunset:    sunTimes.Sunset,
			dayLength: sunTimes.DayLength,
		}
	}
}
