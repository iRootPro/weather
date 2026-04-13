package service

import (
	"context"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

type GeomagneticService struct {
	repo      repository.GeomagneticRepository
	threshold float32
}

func NewGeomagneticService(repo repository.GeomagneticRepository, threshold float32) *GeomagneticService {
	return &GeomagneticService{repo: repo, threshold: threshold}
}

// DashboardSnapshot — данные для карточки на дашборде.
type DashboardSnapshot struct {
	Current    *models.GeomagneticKp
	Status     models.KpStatus
	TodayMaxKp *models.GeomagneticKp
	NextStorm  *models.GeomagneticKp
	HasData    bool
}

// GetDashboardSnapshot собирает текущий слот, максимум за сегодня и ближайший
// прогнозируемый слот с Kp >= threshold (в окне 72 часа вперёд).
func (s *GeomagneticService) GetDashboardSnapshot(ctx context.Context, now time.Time) (*DashboardSnapshot, error) {
	snap := &DashboardSnapshot{}

	current, err := s.repo.GetCurrentKp(ctx, now)
	if err != nil {
		return nil, err
	}
	snap.Current = current
	if current != nil {
		snap.Status = models.ClassifyKp(current.Kp)
		snap.HasData = true
	}

	todayMax, err := s.repo.GetMaxKpForDay(ctx, now)
	if err != nil {
		return nil, err
	}
	snap.TodayMaxKp = todayMax

	storms, err := s.repo.GetForecastedStorms(ctx, now, now.Add(72*time.Hour), s.threshold)
	if err != nil {
		return nil, err
	}
	if len(storms) > 0 {
		// Берём первую запись после now — это ближайшая прогнозируемая буря.
		first := storms[0]
		snap.NextStorm = &first
	}

	return snap, nil
}

// DetailData — данные для детальной страницы.
type DetailData struct {
	Kp    []models.GeomagneticKp
	Daily []models.GeomagneticDaily
	Now   time.Time
}

// GetDetail возвращает срез слотов и суточных показателей в указанном окне.
func (s *GeomagneticService) GetDetail(ctx context.Context, from, to time.Time) (*DetailData, error) {
	kp, err := s.repo.GetKpRange(ctx, from, to)
	if err != nil {
		return nil, err
	}
	daily, err := s.repo.GetDailyRange(ctx, from, to)
	if err != nil {
		return nil, err
	}
	return &DetailData{Kp: kp, Daily: daily, Now: time.Now()}, nil
}
