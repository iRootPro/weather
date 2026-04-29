package maxbot

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
	"github.com/iRootPro/weather/internal/telegram"
)

type Notifier struct {
	client     *Client
	weatherSvc *service.WeatherService
	subRepo    repository.MaxSubscriptionRepository
	notifRepo  repository.MaxNotificationRepository
	userRepo   repository.MaxUserRepository
	interval   time.Duration
	logger     *slog.Logger
}

func NewNotifier(client *Client, weatherSvc *service.WeatherService, subRepo repository.MaxSubscriptionRepository, notifRepo repository.MaxNotificationRepository, userRepo repository.MaxUserRepository, interval int, logger *slog.Logger) *Notifier {
	return &Notifier{client: client, weatherSvc: weatherSvc, subRepo: subRepo, notifRepo: notifRepo, userRepo: userRepo, interval: time.Duration(interval) * time.Second, logger: logger}
}

func (n *Notifier) Start(ctx context.Context) {
	ticker := time.NewTicker(n.interval)
	defer ticker.Stop()
	n.logger.Info("max notifier started", "interval", n.interval)
	n.checkAndNotify(ctx)
	for {
		select {
		case <-ctx.Done():
			n.logger.Info("max notifier stopped")
			return
		case <-ticker.C:
			n.checkAndNotify(ctx)
		}
	}
}

func (n *Notifier) checkAndNotify(ctx context.Context) {
	events, err := n.weatherSvc.GetRecentEvents(ctx, 1)
	if err != nil {
		n.logger.Error("failed to get recent weather events for max", "error", err)
		return
	}
	for _, event := range events {
		n.processEvent(ctx, event)
	}
}

func (n *Notifier) processEvent(ctx context.Context, event models.WeatherEvent) {
	subscriptionType := subscriptionTypeForWeatherEvent(event.Type)
	if subscriptionType == "" {
		return
	}
	subscribers, err := n.getSubscribersForEvent(ctx, subscriptionType)
	if err != nil {
		n.logger.Error("failed to get max subscribers", "event_type", subscriptionType, "error", err)
		return
	}
	for _, userID := range subscribers {
		n.sendNotification(ctx, userID, event)
	}
}

func (n *Notifier) getSubscribersForEvent(ctx context.Context, eventType string) ([]int64, error) {
	subscribers, err := n.subRepo.GetActiveSubscribers(ctx, eventType)
	if err != nil {
		return nil, err
	}
	allSubscribers, err := n.subRepo.GetActiveSubscribers(ctx, EventAll)
	if err != nil {
		return nil, err
	}
	seen := map[int64]bool{}
	for _, id := range subscribers {
		seen[id] = true
	}
	for _, id := range allSubscribers {
		seen[id] = true
	}
	result := make([]int64, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	return result, nil
}

func (n *Notifier) sendNotification(ctx context.Context, maxUserID int64, event models.WeatherEvent) {
	user, err := n.userRepo.GetByUserID(ctx, maxUserID)
	if err != nil {
		n.logger.Error("failed to get max user", "max_user_id", maxUserID, "error", err)
		return
	}
	subscriptionType := subscriptionTypeForWeatherEvent(event.Type)
	wasSent, err := n.notifRepo.WasRecentlySent(ctx, user.ID, subscriptionType, 60*time.Minute)
	if err != nil {
		n.logger.Error("failed to check max notification dedup", "user_id", user.ID, "error", err)
		return
	}
	if wasSent {
		return
	}

	if err := n.client.SendMessageToUser(ctx, maxUserID, textMessage(telegram.FormatEventNotification(event))); err != nil {
		n.logger.Error("failed to send max notification", "max_user_id", maxUserID, "event_type", event.Type, "error", err)
		return
	}
	eventData, _ := json.Marshal(map[string]interface{}{"type": event.Type, "description": event.Description, "value": event.Value, "time": event.Time})
	if err := n.notifRepo.Create(ctx, &models.MaxNotification{UserID: user.ID, EventType: subscriptionType, EventData: eventData, SentAt: time.Now()}); err != nil {
		n.logger.Error("failed to save max notification", "error", err)
	}
}
