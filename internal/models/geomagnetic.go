package models

import "time"

// GeomagneticKp — одна запись 3-часового слота индекса Kp.
type GeomagneticKp struct {
	SlotTime   time.Time
	Kp         float32
	Source     string
	IsForecast bool
	FetchedAt  time.Time
}

// GeomagneticDaily — суточные показатели солнечной активности.
type GeomagneticDaily struct {
	Date      time.Time
	F10       *float32
	Sn        *float32
	Ap        *float32
	MaxKp     *float32
	FetchedAt time.Time
}

// KpStatus — категория геомагнитной активности по шкале NOAA.
type KpStatus int

const (
	KpCalm        KpStatus = iota // 0..3 — спокойно
	KpUnsettled                   // 4 — возмущение
	KpStorm                       // 5..6 — буря (G1–G2)
	KpSevereStorm                 // 7..9 — сильная буря (G3–G5)
)

// ClassifyKp возвращает категорию для значения Kp.
func ClassifyKp(kp float32) KpStatus {
	switch {
	case kp >= 7:
		return KpSevereStorm
	case kp >= 5:
		return KpStorm
	case kp >= 4:
		return KpUnsettled
	default:
		return KpCalm
	}
}

// Label — короткое русское название статуса.
func (s KpStatus) Label() string {
	switch s {
	case KpSevereStorm:
		return "сильная буря"
	case KpStorm:
		return "буря"
	case KpUnsettled:
		return "возмущение"
	default:
		return "спокойно"
	}
}

// Color — базовый Tailwind-цвет статуса (для подстановки в готовые наборы классов).
func (s KpStatus) Color() string {
	switch s {
	case KpSevereStorm:
		return "red"
	case KpStorm:
		return "orange"
	case KpUnsettled:
		return "yellow"
	default:
		return "green"
	}
}

// Emoji — цветовой индикатор статуса.
func (s KpStatus) Emoji() string {
	switch s {
	case KpSevereStorm:
		return "🔴"
	case KpStorm:
		return "🟠"
	case KpUnsettled:
		return "🟡"
	default:
		return "🟢"
	}
}

// TailwindGradient возвращает заранее собранную строку Tailwind-классов для фона
// карточки. Перечисляем классы целиком (а не через интерполяцию), чтобы JIT
// не выпилил их при сборке.
func (s KpStatus) TailwindGradient() string {
	switch s {
	case KpSevereStorm:
		return "from-red-50 to-red-100 dark:from-red-900/20 dark:to-red-800/20"
	case KpStorm:
		return "from-orange-50 to-orange-100 dark:from-orange-900/20 dark:to-orange-800/20"
	case KpUnsettled:
		return "from-yellow-50 to-yellow-100 dark:from-yellow-900/20 dark:to-yellow-800/20"
	default:
		return "from-green-50 to-green-100 dark:from-green-900/20 dark:to-green-800/20"
	}
}

// TextColor возвращает Tailwind-классы текста (главное число + подпись).
func (s KpStatus) TextColor() string {
	switch s {
	case KpSevereStorm:
		return "text-red-700 dark:text-red-300"
	case KpStorm:
		return "text-orange-700 dark:text-orange-300"
	case KpUnsettled:
		return "text-yellow-700 dark:text-yellow-300"
	default:
		return "text-green-700 dark:text-green-300"
	}
}
