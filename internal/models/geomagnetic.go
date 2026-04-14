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

// KpStatus — категория геомагнитной активности.
// Совпадает со шкалой xras.ru: спокойно / возмущение / буря.
type KpStatus int

const (
	KpCalm      KpStatus = iota // 0..3 — спокойная магнитосфера
	KpUnsettled                 // 4 — возбуждённая
	KpStorm                     // ≥5 — магнитная буря (G1+)
)

// ClassifyKp возвращает категорию для значения Kp.
func ClassifyKp(kp float32) KpStatus {
	switch {
	case kp >= 5:
		return KpStorm
	case kp >= 4:
		return KpUnsettled
	default:
		return KpCalm
	}
}

// Label — короткое русское название статуса (как у xras.ru).
func (s KpStatus) Label() string {
	switch s {
	case KpStorm:
		return "магнитная буря"
	case KpUnsettled:
		return "возбуждённая"
	default:
		return "спокойная"
	}
}

// HexColor возвращает HEX-цвет статуса (для Chart.js).
func (s KpStatus) HexColor() string {
	switch s {
	case KpStorm:
		return "#ef4444" // red-500
	case KpUnsettled:
		return "#f59e0b" // amber-500
	default:
		return "#22c55e" // green-500
	}
}

// Emoji — цветовой индикатор статуса.
func (s KpStatus) Emoji() string {
	switch s {
	case KpStorm:
		return "🔴"
	case KpUnsettled:
		return "🟠"
	default:
		return "🟢"
	}
}

// TailwindGradient возвращает заранее собранную строку Tailwind-классов для фона
// карточки. Перечисляем классы целиком (а не через интерполяцию), чтобы JIT
// не выпилил их при сборке.
func (s KpStatus) TailwindGradient() string {
	switch s {
	case KpStorm:
		return "from-red-50 to-red-100 dark:from-red-900/20 dark:to-red-800/20"
	case KpUnsettled:
		return "from-orange-50 to-orange-100 dark:from-orange-900/20 dark:to-orange-800/20"
	default:
		return "from-green-50 to-green-100 dark:from-green-900/20 dark:to-green-800/20"
	}
}

// StormLevel возвращает уровень геомагнитной бури по шкале NOAA и его русское
// описание. ok == false, если Kp < 5 (бури нет). Пороги — официальная таблица
// NOAA Space Weather Prediction Center.
func StormLevel(kp float32) (gLevel, description string, ok bool) {
	switch {
	case kp >= 9:
		return "G5", "экстремальная", true
	case kp >= 8:
		return "G4", "очень сильная", true
	case kp >= 7:
		return "G3", "сильная", true
	case kp >= 6:
		return "G2", "средняя", true
	case kp >= 5:
		return "G1", "слабая", true
	}
	return "", "", false
}

// TextColor возвращает Tailwind-классы текста (главное число + подпись).
func (s KpStatus) TextColor() string {
	switch s {
	case KpStorm:
		return "text-red-700 dark:text-red-300"
	case KpUnsettled:
		return "text-orange-700 dark:text-orange-300"
	default:
		return "text-green-700 dark:text-green-300"
	}
}
