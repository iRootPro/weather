package models

// SkyCondition представляет тип условий освещенности/неба
type SkyCondition string

const (
	SkyNight           SkyCondition = "night"            // Ночь
	SkyTwilight        SkyCondition = "twilight"         // Сумерки
	SkyClear           SkyCondition = "clear"            // Ясно
	SkyMostlyClear     SkyCondition = "mostly_clear"     // Малооблачно
	SkyPartlyCloudy    SkyCondition = "partly_cloudy"    // Облачно
	SkyMostlyCloudy    SkyCondition = "mostly_cloudy"    // Пасмурно
	SkyOvercast        SkyCondition = "overcast"         // Очень пасмурно
)

// SkyConditionInfo содержит информацию об условиях освещенности
type SkyConditionInfo struct {
	Condition          SkyCondition // Тип условий
	Icon               string       // Иконка для отображения
	Description        string       // Описание на русском
	SolarElevation     float64      // Угол солнца над горизонтом (градусы)
	TheoricalLux       float64      // Теоретическая освещенность (lux)
	ActualLux          float64      // Фактическая освещенность (lux)
	CloudCoverEstimate float64      // Оценка облачности (0-100%)
}

// GetIcon возвращает иконку для типа условий
func (c SkyCondition) GetIcon() string {
	switch c {
	case SkyNight:
		return "🌙"
	case SkyTwilight:
		return "🌆"
	case SkyClear:
		return "☀️"
	case SkyMostlyClear:
		return "🌤️"
	case SkyPartlyCloudy:
		return "⛅"
	case SkyMostlyCloudy:
		return "☁️"
	case SkyOvercast:
		return "🌫️"
	default:
		return "❓"
	}
}

// GetDescription возвращает описание на русском
func (c SkyCondition) GetDescription() string {
	switch c {
	case SkyNight:
		return "Ночь"
	case SkyTwilight:
		return "Сумерки"
	case SkyClear:
		return "Ясно"
	case SkyMostlyClear:
		return "Малооблачно"
	case SkyPartlyCloudy:
		return "Облачно"
	case SkyMostlyCloudy:
		return "Пасмурно"
	case SkyOvercast:
		return "Очень пасмурно"
	default:
		return "Неизвестно"
	}
}
