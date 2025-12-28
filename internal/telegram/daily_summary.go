package telegram

import (
	"context"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
)

type DailySummaryService struct {
	bot        *tgbotapi.BotAPI
	weatherSvc *service.WeatherService
	sunSvc     *service.SunService
	subRepo    repository.TelegramSubscriptionRepository
	userRepo   repository.TelegramUserRepository
	sendTime   string // Время отправки в формате "07:00"
	logger     *slog.Logger
}

func NewDailySummaryService(
	bot *tgbotapi.BotAPI,
	weatherSvc *service.WeatherService,
	sunSvc *service.SunService,
	subRepo repository.TelegramSubscriptionRepository,
	userRepo repository.TelegramUserRepository,
	sendTime string,
	logger *slog.Logger,
) *DailySummaryService {
	return &DailySummaryService{
		bot:        bot,
		weatherSvc: weatherSvc,
		sunSvc:     sunSvc,
		subRepo:    subRepo,
		userRepo:   userRepo,
		sendTime:   sendTime,
		logger:     logger,
	}
}

// Start запускает фоновый процесс отправки ежедневных сводок
func (s *DailySummaryService) Start(ctx context.Context) {
	s.logger.Info("daily summary service started", "send_time", s.sendTime)

	// Парсим время отправки
	hour, minute := 7, 0 // По умолчанию 07:00
	if _, err := time.Parse("15:04", s.sendTime); err == nil {
		parsedTime, _ := time.Parse("15:04", s.sendTime)
		hour = parsedTime.Hour()
		minute = parsedTime.Minute()
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	lastSent := time.Time{} // Последняя отправка

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("daily summary service stopped")
			return
		case now := <-ticker.C:
			// Проверяем, нужно ли отправлять сводку
			if now.Hour() == hour && now.Minute() == minute {
				// Проверяем, не отправляли ли мы уже сегодня
				if lastSent.Year() == now.Year() && lastSent.YearDay() == now.YearDay() {
					continue
				}

				s.logger.Info("sending daily summary")
				s.sendDailySummary(ctx)
				lastSent = now
			}
		}
	}
}

// sendDailySummary отправляет утреннюю сводку всем подписчикам
func (s *DailySummaryService) sendDailySummary(ctx context.Context) {
	// Получаем подписчиков на ежедневную сводку
	subscribers, err := s.subRepo.GetActiveSubscribers(ctx, EventDailySummary)
	if err != nil {
		s.logger.Error("failed to get daily summary subscribers", "error", err)
		return
	}

	if len(subscribers) == 0 {
		s.logger.Info("no daily summary subscribers")
		return
	}

	s.logger.Info("processing daily summary", "subscribers", len(subscribers))

	// Получаем текущие данные о погоде
	current, err := s.weatherSvc.GetLatest(ctx)
	if err != nil {
		s.logger.Error("failed to get current weather", "error", err)
		return
	}

	// Получаем данные за вчера в это же время
	yesterdaySame, err := s.weatherSvc.GetDataNearTime(ctx, current.Time.Add(-24*time.Hour))
	if err != nil {
		s.logger.Warn("failed to get yesterday weather", "error", err)
	}

	// Получаем min/max за ночь (00:00 - 07:00 сегодня)
	now := time.Now()
	nightStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	nightEnd := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())
	nightMinMax, err := s.weatherSvc.GetMinMaxInRange(ctx, nightStart, nightEnd)
	if err != nil {
		s.logger.Warn("failed to get night min/max", "error", err)
	}

	// Получаем min/max за сегодня
	dailyMinMax, err := s.weatherSvc.GetDailyMinMax(ctx)
	if err != nil {
		s.logger.Warn("failed to get daily min/max", "error", err)
	}

	// Получаем данные о солнце
	sunData := s.sunSvc.GetTodaySunTimesWithComparison()

	// Форматируем сообщение
	text := FormatDailySummary(current, yesterdaySame, nightMinMax, dailyMinMax, sunData)

	// Отправляем всем подписчикам
	for _, chatID := range subscribers {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"

		if _, err := s.bot.Send(msg); err != nil {
			s.logger.Error("failed to send daily summary",
				"chat_id", chatID,
				"error", err)
		} else {
			s.logger.Debug("daily summary sent", "chat_id", chatID)
		}
	}

	s.logger.Info("daily summary sent to all subscribers")
}
