package telegram

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/wcharczuk/go-chart/v2"
	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/service"
)

// ChartType определяет тип графика
type ChartType string

const (
	ChartTemperature ChartType = "temperature"
	ChartPressure    ChartType = "pressure"
	ChartHumidity    ChartType = "humidity"
)

// ChartPeriod определяет период для графика
type ChartPeriod string

const (
	Chart24Hours ChartPeriod = "24h"
	Chart7Days   ChartPeriod = "7d"
)

// GenerateChart генерирует график погоды и возвращает PNG изображение
func GenerateChart(ctx context.Context, weatherSvc *service.WeatherService, chartType ChartType, period ChartPeriod) ([]byte, error) {
	// Определяем временной диапазон
	now := time.Now()
	var from time.Time
	var interval string

	switch period {
	case Chart24Hours:
		from = now.Add(-24 * time.Hour)
		interval = "5m" // 5-минутные данные
	case Chart7Days:
		from = now.Add(-7 * 24 * time.Hour)
		interval = "1h" // часовые данные
	default:
		from = now.Add(-24 * time.Hour)
		interval = "5m"
	}

	// Получаем данные
	data, err := weatherSvc.GetHistory(ctx, from, now, interval)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no data available")
	}

	// Генерируем график в зависимости от типа
	switch chartType {
	case ChartTemperature:
		return generateTemperatureChart(data, period)
	case ChartPressure:
		return generatePressureChart(data, period)
	case ChartHumidity:
		return generateHumidityChart(data, period)
	default:
		return nil, fmt.Errorf("unknown chart type: %s", chartType)
	}
}

// generateTemperatureChart создаёт график температуры
func generateTemperatureChart(data []models.WeatherData, period ChartPeriod) ([]byte, error) {
	var xValues []time.Time
	var yValues []float64

	// Извлекаем данные температуры
	for _, d := range data {
		if d.TempOutdoor != nil {
			xValues = append(xValues, d.Time)
			yValues = append(yValues, float64(*d.TempOutdoor))
		}
	}

	if len(xValues) == 0 {
		return nil, fmt.Errorf("no temperature data")
	}

	// Создаём график
	graph := chart.Chart{
		Title:      "Температура",
		TitleStyle: chart.Style{FontSize: 14},
		Width:      800,
		Height:     400,
		XAxis: chart.XAxis{
			Style: chart.Style{FontSize: 8},
			ValueFormatter: func(v interface{}) string {
				if t, ok := v.(time.Time); ok {
					if period == Chart24Hours {
						return t.Format("15:04")
					}
					return t.Format("02.01 15:04")
				}
				return ""
			},
		},
		YAxis: chart.YAxis{
			Name:      "°C",
			NameStyle: chart.Style{FontSize: 10},
			Style:     chart.Style{FontSize: 10},
		},
		Series: []chart.Series{
			chart.TimeSeries{
				XValues: xValues,
				YValues: yValues,
				Style: chart.Style{
					StrokeColor: chart.ColorRed,
					StrokeWidth: 2,
				},
			},
		},
	}

	// Рендерим в PNG
	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart: %w", err)
	}

	return buffer.Bytes(), nil
}

// generatePressureChart создаёт график давления
func generatePressureChart(data []models.WeatherData, period ChartPeriod) ([]byte, error) {
	var xValues []time.Time
	var yValues []float64

	// Извлекаем данные давления
	for _, d := range data {
		if d.PressureRelative != nil {
			xValues = append(xValues, d.Time)
			yValues = append(yValues, float64(*d.PressureRelative))
		}
	}

	if len(xValues) == 0 {
		return nil, fmt.Errorf("no pressure data")
	}

	// Создаём график
	graph := chart.Chart{
		Title:      "Давление",
		TitleStyle: chart.Style{FontSize: 14},
		Width:      800,
		Height:     400,
		XAxis: chart.XAxis{
			Style: chart.Style{FontSize: 8},
			ValueFormatter: func(v interface{}) string {
				if t, ok := v.(time.Time); ok {
					if period == Chart24Hours {
						return t.Format("15:04")
					}
					return t.Format("02.01 15:04")
				}
				return ""
			},
		},
		YAxis: chart.YAxis{
			Name:      "мм рт.ст.",
			NameStyle: chart.Style{FontSize: 10},
			Style:     chart.Style{FontSize: 10},
		},
		Series: []chart.Series{
			chart.TimeSeries{
				XValues: xValues,
				YValues: yValues,
				Style: chart.Style{
					StrokeColor: chart.ColorBlue,
					StrokeWidth: 2,
				},
			},
		},
	}

	// Рендерим в PNG
	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart: %w", err)
	}

	return buffer.Bytes(), nil
}

// generateHumidityChart создаёт график влажности
func generateHumidityChart(data []models.WeatherData, period ChartPeriod) ([]byte, error) {
	var xValues []time.Time
	var yValues []float64

	// Извлекаем данные влажности
	for _, d := range data {
		if d.HumidityOutdoor != nil {
			xValues = append(xValues, d.Time)
			yValues = append(yValues, float64(*d.HumidityOutdoor))
		}
	}

	if len(xValues) == 0 {
		return nil, fmt.Errorf("no humidity data")
	}

	// Создаём график
	graph := chart.Chart{
		Title:      "Влажность",
		TitleStyle: chart.Style{FontSize: 14},
		Width:      800,
		Height:     400,
		XAxis: chart.XAxis{
			Style: chart.Style{FontSize: 8},
			ValueFormatter: func(v interface{}) string {
				if t, ok := v.(time.Time); ok {
					if period == Chart24Hours {
						return t.Format("15:04")
					}
					return t.Format("02.01 15:04")
				}
				return ""
			},
		},
		YAxis: chart.YAxis{
			Name:      "%",
			NameStyle: chart.Style{FontSize: 10},
			Style:     chart.Style{FontSize: 10},
		},
		Series: []chart.Series{
			chart.TimeSeries{
				XValues: xValues,
				YValues: yValues,
				Style: chart.Style{
					StrokeColor: chart.ColorGreen,
					StrokeWidth: 2,
				},
			},
		},
	}

	// Рендерим в PNG
	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart: %w", err)
	}

	return buffer.Bytes(), nil
}
