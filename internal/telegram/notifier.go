package telegram

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
)

type Notifier struct {
	bot        *tgbotapi.BotAPI
	weatherSvc *service.WeatherService
	subRepo    repository.TelegramSubscriptionRepository
	notifRepo  repository.TelegramNotificationRepository
	userRepo   repository.TelegramUserRepository
	interval   time.Duration
	logger     *slog.Logger
}

func NewNotifier(
	bot *tgbotapi.BotAPI,
	weatherSvc *service.WeatherService,
	subRepo repository.TelegramSubscriptionRepository,
	notifRepo repository.TelegramNotificationRepository,
	userRepo repository.TelegramUserRepository,
	interval int,
	logger *slog.Logger,
) *Notifier {
	return &Notifier{
		bot:        bot,
		weatherSvc: weatherSvc,
		subRepo:    subRepo,
		notifRepo:  notifRepo,
		userRepo:   userRepo,
		interval:   time.Duration(interval) * time.Second,
		logger:     logger,
	}
}

// Start запускает фоновый процесс отправки уведомлений
func (n *Notifier) Start(ctx context.Context) {
	ticker := time.NewTicker(n.interval)
	defer ticker.Stop()

	n.logger.Info("notifier started", "interval", n.interval)

	// Первая проверка сразу после запуска
	n.checkAndNotify(ctx)

	for {
		select {
		case <-ctx.Done():
			n.logger.Info("notifier stopped")
			return
		case <-ticker.C:
			n.checkAndNotify(ctx)
		}
	}
}

// checkAndNotify проверяет события и отправляет уведомления
func (n *Notifier) checkAndNotify(ctx context.Context) {
	// Получаем события за последний час
	events, err := n.weatherSvc.GetRecentEvents(ctx, 1)
	if err != nil {
		n.logger.Error("failed to get recent events", "error", err)
		return
	}

	if len(events) == 0 {
		return
	}

	n.logger.Info("processing events", "count", len(events))

	// Обрабатываем каждое событие
	for _, event := range events {
		n.processEvent(ctx, event)
	}
}

// processEvent обрабатывает одно событие
func (n *Notifier) processEvent(ctx context.Context, event models.WeatherEvent) {
	// Определяем тип подписки для этого события
	subscriptionType := getSubscriptionTypeForEvent(event.Type)
	if subscriptionType == "" {
		return
	}

	// Получаем подписчиков для этого типа события
	subscribers, err := n.getSubscribersForEvent(ctx, subscriptionType)
	if err != nil {
		n.logger.Error("failed to get subscribers", "event_type", subscriptionType, "error", err)
		return
	}

	if len(subscribers) == 0 {
		return
	}

	n.logger.Info("sending notifications", "event_type", event.Type, "subscribers", len(subscribers))

	// Отправляем уведомления всем подписчикам
	for _, chatID := range subscribers {
		n.sendNotification(ctx, chatID, event)
	}
}

// getSubscribersForEvent получает список подписчиков для события
func (n *Notifier) getSubscribersForEvent(ctx context.Context, eventType string) ([]int64, error) {
	// Получаем подписчиков на конкретный тип события
	subscribers, err := n.subRepo.GetActiveSubscribers(ctx, eventType)
	if err != nil {
		return nil, err
	}

	// Получаем подписчиков на все события
	allSubscribers, err := n.subRepo.GetActiveSubscribers(ctx, EventAll)
	if err != nil {
		return nil, err
	}

	// Объединяем списки (с удалением дубликатов)
	chatIDMap := make(map[int64]bool)
	for _, chatID := range subscribers {
		chatIDMap[chatID] = true
	}
	for _, chatID := range allSubscribers {
		chatIDMap[chatID] = true
	}

	// Преобразуем в слайс
	result := make([]int64, 0, len(chatIDMap))
	for chatID := range chatIDMap {
		result = append(result, chatID)
	}

	return result, nil
}

// sendNotification отправляет уведомление одному пользователю
func (n *Notifier) sendNotification(ctx context.Context, chatID int64, event models.WeatherEvent) {
	// Получаем user_id по chat_id
	user, err := n.userRepo.GetByChatID(ctx, chatID)
	if err != nil {
		n.logger.Error("failed to get user", "chat_id", chatID, "error", err)
		return
	}

	// Проверяем, не отправляли ли мы это уведомление недавно (за последние 60 минут)
	subscriptionType := getSubscriptionTypeForEvent(event.Type)
	wasSent, err := n.notifRepo.WasRecentlySent(ctx, user.ID, subscriptionType, 60*time.Minute)
	if err != nil {
		n.logger.Error("failed to check recent notification", "user_id", user.ID, "error", err)
		return
	}

	if wasSent {
		n.logger.Debug("notification already sent recently",
			"user_id", user.ID,
			"event_type", subscriptionType)
		return
	}

	// Форматируем сообщение
	text := FormatEventNotification(event)

	// Отправляем сообщение
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	_, err = n.bot.Send(msg)
	if err != nil {
		n.logger.Error("failed to send notification",
			"chat_id", chatID,
			"event_type", event.Type,
			"error", err)
		return
	}

	// Сохраняем запись об отправленном уведомлении
	eventData, _ := json.Marshal(map[string]interface{}{
		"type":        event.Type,
		"description": event.Description,
		"value":       event.Value,
		"time":        event.Time,
	})

	notification := &models.TelegramNotification{
		UserID:    user.ID,
		EventType: subscriptionType,
		EventData: eventData,
		SentAt:    time.Now(),
	}

	if err := n.notifRepo.Create(ctx, notification); err != nil {
		n.logger.Error("failed to save notification", "error", err)
	}

	n.logger.Info("notification sent",
		"chat_id", chatID,
		"event_type", event.Type)
}

// getSubscriptionTypeForEvent возвращает тип подписки для события
func getSubscriptionTypeForEvent(eventType string) string {
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
