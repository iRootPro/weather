package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
)

// WeatherServiceInterface defines the interface for weather service
type WeatherServiceInterface interface {
	GetCurrent(ctx context.Context) (*models.WeatherData, error)
	GetCurrentWithHourlyChange(ctx context.Context) (*models.WeatherData, *models.WeatherData, *repository.DailyMinMax, error)
	GetHistory(ctx context.Context, from, to time.Time, interval string) ([]models.WeatherData, error)
	GetRecentEvents(ctx context.Context, hours int) ([]models.WeatherEvent, error)
}

// Tab constants
const (
	TabDashboard = iota
	TabCharts
	TabEvents
	TabHelp
)

// Model represents the main TUI application state
type Model struct {
	// Services
	weatherService WeatherServiceInterface
	sunService     *service.SunService

	// UI state
	activeTab int
	width     int
	height    int

	// Data
	currentWeather *models.WeatherData
	hourAgoWeather *models.WeatherData
	dailyMinMax    *repository.DailyMinMax
	events         []models.WeatherEvent
	chartData      []models.WeatherData
	sunrise        time.Time
	sunset         time.Time
	dayLength      time.Duration

	// Chart state
	chartPeriod   string // "24h", "7d", "30d"
	chartMetric   int    // 0=temp, 1=pressure, 2=wind, 3=humidity

	// Components
	spinner spinner.Model

	// State flags
	loading    bool
	err        error
	lastUpdate time.Time
	ready      bool
}

// NewModel creates a new TUI model
func NewModel(weatherService WeatherServiceInterface, sunService *service.SunService) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return Model{
		weatherService: weatherService,
		sunService:     sunService,
		activeTab:      TabDashboard,
		chartPeriod:    "24h",
		chartMetric:    0,
		spinner:        s,
		loading:        true,
	}
}
