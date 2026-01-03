package service

import (
	"math"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

type SkyConditionsService struct {
	sunService *SunService
}

func NewSkyConditionsService(sunService *SunService) *SkyConditionsService {
	return &SkyConditionsService{
		sunService: sunService,
	}
}

// DetermineSkyConditions определяет условия освещенности на основе данных
func (s *SkyConditionsService) DetermineSkyConditions(
	t time.Time,
	solarRadiation *float32,
) *models.SkyConditionInfo {
	// Получаем угол солнца над горизонтом
	elevation := s.sunService.GetSolarElevation(t)

	// Рассчитываем теоретическую максимальную освещенность (lux)
	theoreticalLux := s.calculateTheoreticalLux(elevation)

	// Фактическая освещенность из солнечной радиации
	var actualLux float64
	if solarRadiation != nil {
		// Конвертируем W/m² в lux (приблизительно 120 lux на 1 W/m²)
		actualLux = float64(*solarRadiation) * 120.0
	}

	// Определяем тип условий
	condition := s.classifyConditions(elevation, theoreticalLux, actualLux)

	// Оценка облачности (0-100%)
	cloudCover := s.estimateCloudCover(elevation, theoreticalLux, actualLux)

	return &models.SkyConditionInfo{
		Condition:          condition,
		Icon:               condition.GetIcon(),
		Description:        condition.GetDescription(),
		SolarElevation:     elevation,
		TheoricalLux:       theoreticalLux,
		ActualLux:          actualLux,
		CloudCoverEstimate: cloudCover,
	}
}

// calculateTheoreticalLux рассчитывает теоретическую освещенность для данного угла солнца
func (s *SkyConditionsService) calculateTheoreticalLux(elevation float64) float64 {
	if elevation <= 0 {
		// Солнце за горизонтом
		return 0
	}

	// Максимальная освещенность в ясный день (lux)
	// На экваторе в полдень: ~120,000 lux
	// Учитываем атмосферное поглощение
	maxLux := 120000.0

	// Формула: Lux = MaxLux * sin(elevation) * atmospheric_transmission
	// Atmospheric transmission зависит от высоты солнца
	// При малых углах - больше поглощение

	// Air mass (сколько атмосферы проходит свет)
	var airMass float64
	if elevation >= 10 {
		airMass = 1.0 / math.Sin(elevation*math.Pi/180.0)
	} else {
		// При малых углах используем более точную формулу
		airMass = 1.0 / (math.Sin(elevation*math.Pi/180.0) + 0.50572*math.Pow(elevation+6.07995, -1.6364))
	}

	// Transmission через атмосферу (упрощенная модель)
	transmission := math.Pow(0.7, math.Pow(airMass, 0.678))

	// Рассчитываем теоретическую освещенность
	lux := maxLux * math.Sin(elevation*math.Pi/180.0) * transmission

	return lux
}

// classifyConditions классифицирует условия на основе данных
func (s *SkyConditionsService) classifyConditions(
	elevation float64,
	theoreticalLux float64,
	actualLux float64,
) models.SkyCondition {
	// Ночь
	if elevation < -6.0 {
		return models.SkyNight
	}

	// Сумерки
	if elevation < 0 {
		return models.SkyTwilight
	}

	// День - классифицируем по облачности
	if theoreticalLux < 100 {
		// Очень низкое солнце - возвращаем сумерки
		return models.SkyTwilight
	}

	// Рассчитываем отношение фактической освещенности к теоретической
	ratio := actualLux / theoreticalLux

	// Классификация по облачности
	switch {
	case ratio > 0.75:
		return models.SkyClear // Ясно
	case ratio > 0.55:
		return models.SkyMostlyClear // Малооблачно
	case ratio > 0.35:
		return models.SkyPartlyCloudy // Облачно
	case ratio > 0.15:
		return models.SkyMostlyCloudy // Пасмурно
	default:
		return models.SkyOvercast // Очень пасмурно
	}
}

// estimateCloudCover оценивает облачность в процентах
func (s *SkyConditionsService) estimateCloudCover(
	elevation float64,
	theoreticalLux float64,
	actualLux float64,
) float64 {
	if elevation < 0 {
		return 0 // Ночь/сумерки
	}

	if theoreticalLux < 100 {
		return 0
	}

	ratio := actualLux / theoreticalLux

	// Простая линейная модель
	// 1.0 ratio = 0% облачность
	// 0.0 ratio = 100% облачность
	cloudCover := (1.0 - ratio) * 100.0

	// Ограничиваем 0-100%
	if cloudCover < 0 {
		cloudCover = 0
	}
	if cloudCover > 100 {
		cloudCover = 100
	}

	return cloudCover
}
