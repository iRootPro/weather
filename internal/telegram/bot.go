package telegram

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
)

type BotHandler struct {
	bot         *tgbotapi.BotAPI
	weatherSvc  *service.WeatherService
	sunSvc      *service.SunService
	moonSvc     *service.MoonService
	forecastSvc *service.ForecastService
	userRepo    repository.TelegramUserRepository
	subRepo     repository.TelegramSubscriptionRepository
	notifRepo   repository.TelegramNotificationRepository
	photoRepo   repository.PhotoRepository
	adminIDs    []int64
	logger      *slog.Logger
}

func NewBotHandler(
	bot *tgbotapi.BotAPI,
	weatherSvc *service.WeatherService,
	sunSvc *service.SunService,
	moonSvc *service.MoonService,
	forecastSvc *service.ForecastService,
	userRepo repository.TelegramUserRepository,
	subRepo repository.TelegramSubscriptionRepository,
	notifRepo repository.TelegramNotificationRepository,
	photoRepo repository.PhotoRepository,
	adminIDs []int64,
	logger *slog.Logger,
) *BotHandler {
	return &BotHandler{
		bot:         bot,
		weatherSvc:  weatherSvc,
		sunSvc:      sunSvc,
		moonSvc:     moonSvc,
		forecastSvc: forecastSvc,
		userRepo:    userRepo,
		subRepo:     subRepo,
		notifRepo:   notifRepo,
		photoRepo:   photoRepo,
		adminIDs:    adminIDs,
		logger:      logger,
	}
}

func (h *BotHandler) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
	// Обработка команд
	if update.Message != nil && update.Message.IsCommand() {
		h.handleCommand(ctx, update.Message)
		return
	}

	// Обработка callback кнопок
	if update.CallbackQuery != nil {
		h.handleCallbackQuery(ctx, update.CallbackQuery)
		return
	}

	// Обработка фотографий
	if update.Message != nil && update.Message.Photo != nil {
		h.handlePhoto(ctx, update.Message)
		return
	}

	// Обработка обычных сообщений
	if update.Message != nil {
		h.handleMessage(ctx, update.Message)
	}
}

func (h *BotHandler) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	h.bot.Send(msg)
}

// isAdmin проверяет, является ли пользователь админом
func (h *BotHandler) isAdmin(chatID int64) bool {
	for _, adminID := range h.adminIDs {
		if adminID == chatID {
			return true
		}
	}
	return false
}
