package mqtt

import (
	"context"
	"log/slog"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/iRootPro/weather/internal/repository"
)

// Handler обрабатывает входящие MQTT сообщения
type Handler struct {
	parser     *Parser
	weatherRepo repository.WeatherRepository
	logger     *slog.Logger
}

func NewHandler(weatherRepo repository.WeatherRepository, logger *slog.Logger) *Handler {
	return &Handler{
		parser:      NewParser(),
		weatherRepo: weatherRepo,
		logger:      logger,
	}
}

// HandleMessage возвращает обработчик для MQTT сообщений
func (h *Handler) HandleMessage() mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		h.logger.Debug("received message",
			"topic", msg.Topic(),
			"payload_size", len(msg.Payload()),
		)

		weather, err := h.parser.Parse(msg.Payload())
		if err != nil {
			h.logger.Error("failed to parse message",
				"topic", msg.Topic(),
				"error", err,
			)
			return
		}

		ctx := context.Background()
		if err := h.weatherRepo.Save(ctx, weather); err != nil {
			h.logger.Error("failed to save weather data",
				"error", err,
			)
			return
		}

		// Форматируем значения для логов (разыменовываем указатели)
		logAttrs := []any{"time", weather.Time}
		if weather.TempOutdoor != nil {
			logAttrs = append(logAttrs, "temp_outdoor", *weather.TempOutdoor)
		}
		if weather.HumidityOutdoor != nil {
			logAttrs = append(logAttrs, "humidity_outdoor", *weather.HumidityOutdoor)
		}
		if weather.PressureRelative != nil {
			logAttrs = append(logAttrs, "pressure", *weather.PressureRelative)
		}
		if weather.WindSpeed != nil {
			logAttrs = append(logAttrs, "wind_speed", *weather.WindSpeed)
		}
		if weather.RainRate != nil && *weather.RainRate > 0 {
			logAttrs = append(logAttrs, "rain_rate", *weather.RainRate)
		}

		h.logger.Info("weather data saved", logAttrs...)
	}
}
