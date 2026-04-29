package maxbot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
	"github.com/iRootPro/weather/internal/telegram"
)

type BotHandler struct {
	client      *Client
	weatherSvc  *service.WeatherService
	userRepo    repository.MaxUserRepository
	subRepo     repository.MaxSubscriptionRepository
	forecastSvc *service.ForecastService
	logger      *slog.Logger
}

func NewBotHandler(client *Client, weatherSvc *service.WeatherService, forecastSvc *service.ForecastService, userRepo repository.MaxUserRepository, subRepo repository.MaxSubscriptionRepository, logger *slog.Logger) *BotHandler {
	return &BotHandler{client: client, weatherSvc: weatherSvc, forecastSvc: forecastSvc, userRepo: userRepo, subRepo: subRepo, logger: logger}
}

func (h *BotHandler) HandleUpdate(ctx context.Context, update Update) {
	h.logger.Info("max update received", "type", update.UpdateType)
	switch update.UpdateType {
	case "bot_started":
		if update.User != nil {
			h.handleBotStarted(ctx, update.User, update.UserLocale)
		}
	case "bot_stopped":
		if update.User != nil {
			if err := h.userRepo.UpdateActivity(ctx, update.User.UserID, false); err != nil {
				h.logger.Error("failed to mark max user inactive", "user_id", update.User.UserID, "error", err)
			}
		}
	case "message_created":
		if update.Message != nil {
			h.handleMessage(ctx, update.Message, update.UserLocale)
		}
	case "message_callback":
		if update.Callback != nil {
			h.handleCallback(ctx, update.Callback)
		}
	}
}

func (h *BotHandler) handleBotStarted(ctx context.Context, u *User, locale *string) {
	user, isNew := h.registerUser(ctx, u, locale)
	if user == nil {
		return
	}
	if isNew {
		h.subscribeDefault(ctx, user.ID)
	}
	h.handleStart(ctx, user.UserID)
}

func (h *BotHandler) handleMessage(ctx context.Context, msg *Message, locale *string) {
	if msg.Sender == nil || msg.Sender.IsBot {
		return
	}
	user, isNew := h.registerUser(ctx, msg.Sender, locale)
	if user == nil {
		return
	}
	if isNew {
		h.subscribeDefault(ctx, user.ID)
	}

	text := ""
	if msg.Body.Text != nil {
		text = strings.TrimSpace(*msg.Body.Text)
	}
	if text == "" {
		h.send(ctx, user.UserID, "Используйте /help для списка команд")
		return
	}

	if strings.HasPrefix(text, "/") {
		parts := strings.Fields(strings.TrimPrefix(text, "/"))
		cmd := strings.Split(parts[0], "@")[0]
		switch cmd {
		case CmdStart, CmdMenu:
			h.handleStart(ctx, user.UserID)
		case CmdHelp:
			h.handleHelp(ctx, user.UserID)
		case CmdWeather, CmdCurrent:
			h.handleWeather(ctx, user.UserID)
		case CmdSubscribe:
			h.handleSubscribe(ctx, user.UserID)
		case CmdUnsubscribe:
			h.handleUnsubscribe(ctx, user)
		default:
			h.send(ctx, user.UserID, "Неизвестная команда. Используйте /help")
		}
		return
	}

	switch strings.ToLower(text) {
	case "погода", "🌦️ погода":
		h.handleWeather(ctx, user.UserID)
	case "подписки", "🔔 подписки":
		h.handleSubscribe(ctx, user.UserID)
	case "помощь", "📖 помощь", "меню":
		h.handleHelp(ctx, user.UserID)
	default:
		h.sendWithKeyboard(ctx, user.UserID, "Используйте кнопки ниже, /weather для погоды или /subscribe для подписок", inlineMainKeyboard())
	}
}

func (h *BotHandler) registerUser(ctx context.Context, u *User, locale *string) (*models.MaxUser, bool) {
	_, err := h.userRepo.GetByUserID(ctx, u.UserID)
	isNew := err != nil
	language := "ru"
	if locale != nil && *locale != "" {
		language = *locale
	}
	user := &models.MaxUser{UserID: u.UserID, Username: u.Username, FirstName: &u.FirstName, LastName: u.LastName, LanguageCode: language, IsBot: u.IsBot}
	if err := h.userRepo.Create(ctx, user); err != nil {
		h.logger.Error("failed to create/update max user", "user_id", u.UserID, "error", err)
		return nil, false
	}
	return user, isNew
}

func (h *BotHandler) subscribeDefault(ctx context.Context, userID int64) {
	if err := h.subRepo.Create(ctx, &models.MaxSubscription{UserID: userID, EventType: EventDailySummary, IsActive: true}); err != nil {
		h.logger.Error("failed to create default max subscription", "user_id", userID, "error", err)
	}
}

func (h *BotHandler) handleStart(ctx context.Context, userID int64) {
	text := "🌦️ *Добро пожаловать в бот метеостанции города Армавир!*\n\n" +
		"Я показываю текущую погоду и могу присылать уведомления о важных изменениях.\n\n" +
		"Что можно сделать:\n" +
		"• нажать кнопку *Погода* — получить текущую погоду;\n" +
		"• нажать *Подписки* — выбрать уведомления;\n" +
		"• написать /weather — текущая погода;\n" +
		"• написать /subscribe — настройки подписок;\n" +
		"• написать /unsubscribe — отписаться от всех уведомлений.\n\n" +
		"Новые пользователи автоматически подписываются на утреннюю сводку."
	h.sendWithKeyboard(ctx, userID, text, inlineMainKeyboard())
}

func (h *BotHandler) handleHelp(ctx context.Context, userID int64) {
	text := "📖 *Справка*\n\n" +
		"/weather - текущая погода\n" +
		"/subscribe - выбрать уведомления\n" +
		"/unsubscribe - отписаться от всех уведомлений\n" +
		"/start - показать главное меню\n\n" +
		"Также можно пользоваться кнопками ниже."
	h.sendWithKeyboard(ctx, userID, text, inlineMainKeyboard())
}

func (h *BotHandler) handleWeather(ctx context.Context, userID int64) {
	current, hourAgo, dailyMinMax, err := h.weatherSvc.GetCurrentWithHourlyChange(ctx)
	if err != nil {
		h.logger.Error("failed to get current weather", "error", err)
		h.send(ctx, userID, "❌ Ошибка получения данных о погоде")
		return
	}
	text := telegram.FormatCurrentWeather(current, hourAgo, dailyMinMax)
	h.send(ctx, userID, text)
}

func (h *BotHandler) handleSubscribe(ctx context.Context, userID int64) {
	body := textMessage("Выберите тип уведомлений:")
	body.Attachments = subscriptionKeyboard()
	if err := h.client.SendMessageToUser(ctx, userID, body); err != nil {
		h.logger.Error("failed to send max subscription keyboard", "user_id", userID, "error", err)
	}
}

func (h *BotHandler) handleUnsubscribe(ctx context.Context, user *models.MaxUser) {
	if err := h.subRepo.DeleteAll(ctx, user.ID); err != nil {
		h.logger.Error("failed to unsubscribe max user", "user_id", user.ID, "error", err)
		h.send(ctx, user.UserID, "❌ Ошибка отписки")
		return
	}
	h.send(ctx, user.UserID, "✅ Вы успешно отписались от всех уведомлений")
}

func (h *BotHandler) handleCallback(ctx context.Context, cb *Callback) {
	user, err := h.userRepo.GetByUserID(ctx, cb.User.UserID)
	if err != nil {
		created, _ := h.registerUser(ctx, &cb.User, nil)
		user = created
	}
	if user == nil {
		return
	}
	_ = h.client.AnswerCallback(ctx, cb.CallbackID, "")
	data := cb.Payload
	switch data {
	case "cmd_weather":
		h.handleWeather(ctx, user.UserID)
		return
	case "cmd_subscribe":
		h.handleSubscribe(ctx, user.UserID)
		return
	case "cmd_help":
		h.handleHelp(ctx, user.UserID)
		return
	}

	if strings.HasPrefix(data, "sub_") {
		eventType := strings.TrimPrefix(data, "sub_")
		if err := h.subRepo.Create(ctx, &models.MaxSubscription{UserID: user.ID, EventType: eventType, IsActive: true}); err != nil {
			h.logger.Error("failed to create max subscription", "user_id", user.ID, "event_type", eventType, "error", err)
			return
		}
		h.send(ctx, user.UserID, fmt.Sprintf("✅ Вы подписались на уведомления: %s", GetEventTypeName(eventType)))
		return
	}
	if data == "unsub_all" {
		_ = h.subRepo.DeleteAll(ctx, user.ID)
		h.send(ctx, user.UserID, "✅ Вы отписались от всех уведомлений")
	}
}

func (h *BotHandler) send(ctx context.Context, userID int64, text string) {
	h.sendWithKeyboard(ctx, userID, text, nil)
}

func (h *BotHandler) sendWithKeyboard(ctx context.Context, userID int64, text string, attachments []interface{}) {
	body := textMessage(text)
	body.Attachments = attachments
	if err := h.client.SendMessageToUser(ctx, userID, body); err != nil {
		h.logger.Error("failed to send max message", "user_id", userID, "error", err)
	}
}
