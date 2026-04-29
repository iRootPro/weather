package maxbot

import (
	"context"
	"log/slog"
	"time"

	"github.com/iRootPro/weather/internal/repository"
	"github.com/iRootPro/weather/internal/service"
	"github.com/iRootPro/weather/internal/telegram"
)

type DailySummaryService struct {
	client     *Client
	weatherSvc *service.WeatherService
	sunSvc     *service.SunService
	geomagSvc  *service.GeomagneticService
	subRepo    repository.MaxSubscriptionRepository
	sendTime   string
	logger     *slog.Logger
}

func NewDailySummaryService(client *Client, weatherSvc *service.WeatherService, sunSvc *service.SunService, geomagSvc *service.GeomagneticService, subRepo repository.MaxSubscriptionRepository, sendTime string, logger *slog.Logger) *DailySummaryService {
	return &DailySummaryService{client: client, weatherSvc: weatherSvc, sunSvc: sunSvc, geomagSvc: geomagSvc, subRepo: subRepo, sendTime: sendTime, logger: logger}
}

func (s *DailySummaryService) Start(ctx context.Context) {
	s.logger.Info("max daily summary service started", "send_time", s.sendTime)
	hour, minute := 7, 0
	if parsed, err := time.Parse("15:04", s.sendTime); err == nil {
		hour, minute = parsed.Hour(), parsed.Minute()
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	lastSent := time.Time{}
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("max daily summary service stopped")
			return
		case now := <-ticker.C:
			if now.Hour() == hour && now.Minute() == minute && !(lastSent.Year() == now.Year() && lastSent.YearDay() == now.YearDay()) {
				s.sendDailySummary(ctx)
				lastSent = now
			}
		}
	}
}

func (s *DailySummaryService) sendDailySummary(ctx context.Context) {
	subscribers, err := s.subRepo.GetActiveSubscribers(ctx, EventDailySummary)
	if err != nil {
		s.logger.Error("failed to get max daily summary subscribers", "error", err)
		return
	}
	if len(subscribers) == 0 {
		return
	}

	current, err := s.weatherSvc.GetLatest(ctx)
	if err != nil {
		s.logger.Error("failed to get current weather for max daily summary", "error", err)
		return
	}
	yesterdaySame, err := s.weatherSvc.GetDataNearTime(ctx, current.Time.Add(-24*time.Hour))
	if err != nil {
		s.logger.Warn("failed to get yesterday weather", "error", err)
	}

	now := time.Now()
	nightStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	nightEnd := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())
	nightMinMax, err := s.weatherSvc.GetMinMaxInRange(ctx, nightStart, nightEnd)
	if err != nil {
		s.logger.Warn("failed to get night min/max", "error", err)
	}
	dailyMinMax, err := s.weatherSvc.GetDailyMinMax(ctx)
	if err != nil {
		s.logger.Warn("failed to get daily min/max", "error", err)
	}

	var geomagSnap *service.DashboardSnapshot
	if s.geomagSvc != nil {
		if snap, err := s.geomagSvc.GetDashboardSnapshot(ctx, time.Now()); err == nil && snap != nil && snap.HasData {
			geomagSnap = snap
		}
	}

	text := telegram.FormatDailySummary(current, yesterdaySame, nightMinMax, dailyMinMax, s.sunSvc.GetTodaySunTimesWithComparison(), nil, geomagSnap)
	for _, userID := range subscribers {
		if err := s.client.SendMessageToUser(ctx, userID, textMessage(text)); err != nil {
			s.logger.Error("failed to send max daily summary", "user_id", userID, "error", err)
		}
	}
}
