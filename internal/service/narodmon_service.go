package service

import (
	"context"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

type NarodmonService struct {
	narodmonLogRepo repository.NarodmonLogRepository
}

func NewNarodmonService(narodmonLogRepo repository.NarodmonLogRepository) *NarodmonService {
	return &NarodmonService{
		narodmonLogRepo: narodmonLogRepo,
	}
}

func (s *NarodmonService) GetStatus(ctx context.Context) (*models.NarodmonStatus, error) {
	latest, err := s.narodmonLogRepo.GetLatest(ctx)
	if err != nil {
		return nil, err
	}

	if latest == nil {
		return nil, nil // Нет отправок
	}

	status := &models.NarodmonStatus{
		LastSentAt:   &latest.SentAt,
		Success:      latest.Success,
		SensorsCount: latest.SensorsCount,
	}

	if latest.ErrorMessage != nil {
		status.ErrorMessage = *latest.ErrorMessage
	}

	return status, nil
}
