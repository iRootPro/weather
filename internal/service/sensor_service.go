package service

import (
	"context"

	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

type SensorService struct {
	repo repository.SensorRepository
}

func NewSensorService(repo repository.SensorRepository) *SensorService {
	return &SensorService{repo: repo}
}

func (s *SensorService) GetAll(ctx context.Context) ([]models.Sensor, error) {
	return s.repo.GetAll(ctx)
}

func (s *SensorService) GetByCode(ctx context.Context, code string) (*models.Sensor, error) {
	return s.repo.GetByCode(ctx, code)
}
