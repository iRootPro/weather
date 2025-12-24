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
		// Temporary INFO level to see raw payload
		h.logger.Info("received MQTT message",
			"topic", msg.Topic(),
			"payload", string(msg.Payload()),
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

		h.logger.Info("weather data saved",
			"time", weather.Time,
			"temp_outdoor", weather.TempOutdoor,
			"humidity_outdoor", weather.HumidityOutdoor,
			"pressure", weather.PressureRelative,
		)
	}
}
