package telegram

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/iRootPro/weather/internal/models"
)

func (h *BotHandler) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	// Регистрация/обновление пользователя
	user := &models.TelegramUser{
		ChatID:       msg.Chat.ID,
		Username:     &msg.From.UserName,
		FirstName:    &msg.From.FirstName,
		LastName:     &msg.From.LastName,
		LanguageCode: msg.From.LanguageCode,
		IsBot:        msg.From.IsBot,
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		h.logger.Error("failed to create/update user", "error", err)
	}

	h.logger.Info("command received",
		"command", msg.Command(),
		"chat_id", msg.Chat.ID,
		"username", msg.From.UserName,
	)

	switch msg.Command() {
	case CmdStart:
		h.handleStart(ctx, msg)
	case CmdHelp:
		h.handleHelp(ctx, msg)
	case CmdWeather, CmdCurrent:
		h.handleCurrentWeather(ctx, msg)
	case CmdStats:
		h.handleStats(ctx, msg)
	case CmdRecords:
		h.handleRecords(ctx, msg)
	case CmdHistory:
		h.handleHistory(ctx, msg)
	case CmdSun:
		h.handleSun(ctx, msg)
	case CmdMoon:
		h.handleMoon(ctx, msg)
	case CmdSubscribe:
		h.handleSubscribe(ctx, msg)
	case CmdUnsubscribe:
		h.handleUnsubscribe(ctx, msg)
	default:
		h.sendMessage(msg.Chat.ID, "Неизвестная команда. Используйте /help для списка команд.")
	}
}

func (h *BotHandler) handleStart(ctx context.Context, msg *tgbotapi.Message) {
	text := `🌦️ *Добро пожаловать в бот метеостанции города Армавир!*

Я могу предоставить вам актуальную информацию о погоде и отправлять уведомления о важных изменениях.

Используйте кнопки ниже для быстрого доступа к информации или команды:
/weather - текущая погода
/stats - статистика
/subscribe - подписаться на уведомления

Для полного списка команд используйте /help`

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetReplyKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleHelp(ctx context.Context, msg *tgbotapi.Message) {
	text := `📖 *Справка по командам*

*Основные:*
/weather - текущая погода
/stats - статистика за период
/records - рекорды за всё время
/history - история данных

*Астрономия:*
/sun - восход и закат
/moon - фаза луны

*Уведомления:*
/subscribe - подписаться на события
/unsubscribe - отписаться

Используйте кнопки внизу экрана для быстрого доступа!`

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetReplyKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleCurrentWeather(ctx context.Context, msg *tgbotapi.Message) {
	current, hourAgo, dailyMinMax, err := h.weatherSvc.GetCurrentWithHourlyChange(ctx)
	if err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка получения данных о погоде")
		h.logger.Error("failed to get current weather", "error", err)
		return
	}

	text := FormatCurrentWeather(current, hourAgo, dailyMinMax)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetWeatherDetailKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleStats(ctx context.Context, msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	period := "day"
	if args != "" {
		period = args
	}

	stats, err := h.weatherSvc.GetStats(ctx, period)
	if err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка получения статистики")
		h.logger.Error("failed to get stats", "error", err)
		return
	}

	text := FormatStats(stats)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetStatsKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleRecords(ctx context.Context, msg *tgbotapi.Message) {
	records, err := h.weatherSvc.GetRecords(ctx)
	if err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка получения рекордов")
		h.logger.Error("failed to get records", "error", err)
		return
	}

	text := FormatRecords(records)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetMainKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleHistory(ctx context.Context, msg *tgbotapi.Message) {
	h.sendMessage(msg.Chat.ID, "История в разработке. Используйте /stats для статистики.")
}

func (h *BotHandler) handleSun(ctx context.Context, msg *tgbotapi.Message) {
	sunData := h.sunSvc.GetTodaySunTimesWithComparison()

	text := FormatSunData(sunData)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetMainKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleMoon(ctx context.Context, msg *tgbotapi.Message) {
	moonData := h.moonSvc.GetTodayMoonData()

	text := FormatMoonData(moonData)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetMainKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleSubscribe(ctx context.Context, msg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Выберите тип уведомлений:")
	reply.ReplyMarkup = GetSubscriptionKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleUnsubscribe(ctx context.Context, msg *tgbotapi.Message) {
	user, err := h.userRepo.GetByChatID(ctx, msg.Chat.ID)
	if err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка: пользователь не найден")
		return
	}

	if err := h.subRepo.DeleteAll(ctx, user.ID); err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка отписки")
		h.logger.Error("failed to unsubscribe", "error", err)
		return
	}

	h.sendMessage(msg.Chat.ID, "✅ Вы успешно отписались от всех уведомлений")
}

func (h *BotHandler) handleCallbackQuery(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	// Подтверждаем получение callback
	h.bot.Request(tgbotapi.NewCallback(callback.ID, ""))

	user, err := h.userRepo.GetByChatID(ctx, callback.Message.Chat.ID)
	if err != nil {
		h.logger.Error("failed to get user", "error", err)
		return
	}

	// Обработка подписок
	if strings.HasPrefix(data, "sub_") {
		eventType := strings.TrimPrefix(data, "sub_")

		sub := &models.TelegramSubscription{
			UserID:    user.ID,
			EventType: eventType,
			IsActive:  true,
		}

		if err := h.subRepo.Create(ctx, sub); err != nil {
			h.logger.Error("failed to create subscription", "error", err)
			return
		}

		text := fmt.Sprintf("✅ Вы подписались на уведомления: %s", GetEventTypeName(eventType))
		h.bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, text))
		return
	}

	// Обработка отписок
	if strings.HasPrefix(data, "unsub_") {
		eventType := strings.TrimPrefix(data, "unsub_")

		if eventType == "all" {
			h.subRepo.DeleteAll(ctx, user.ID)
			h.bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "✅ Вы отписались от всех уведомлений"))
		} else {
			h.subRepo.Delete(ctx, user.ID, eventType)
			text := fmt.Sprintf("✅ Вы отписались от: %s", GetEventTypeName(eventType))
			h.bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, text))
		}
		return
	}

	// Обработка команд через кнопки
	switch data {
	case "cmd_weather":
		h.handleCurrentWeather(ctx, callback.Message)
	case "cmd_stats":
		h.handleStats(ctx, callback.Message)
	case "cmd_records":
		h.handleRecords(ctx, callback.Message)
	case "cmd_sun":
		h.handleSun(ctx, callback.Message)
	case "cmd_moon":
		h.handleMoon(ctx, callback.Message)
	case "cmd_subscribe":
		h.handleSubscribe(ctx, callback.Message)
	case "stats_day", "stats_week", "stats_month", "stats_year":
		period := strings.TrimPrefix(data, "stats_")
		msg := &tgbotapi.Message{
			Chat: callback.Message.Chat,
			From: callback.From,
			Text: "/stats " + period,
		}
		h.handleStats(ctx, msg)
	}
}

func (h *BotHandler) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	// Обработка нажатий на кнопки постоянной клавиатуры
	switch msg.Text {
	case "🌦️ Погода":
		h.handleCurrentWeather(ctx, msg)
	case "📈 Статистика":
		h.handleStats(ctx, msg)
	case "🏆 Рекорды":
		h.handleRecords(ctx, msg)
	case "☀️ Солнце":
		h.handleSun(ctx, msg)
	case "🌙 Луна":
		h.handleMoon(ctx, msg)
	case "🔔 Подписки":
		h.handleSubscribe(ctx, msg)
	default:
		h.sendMessage(msg.Chat.ID, "Используйте кнопки ниже или /help для списка команд")
	}
}
