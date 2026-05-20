package models

import "time"

// HydroGauge — гидропост источника Эмерсит.
type HydroGauge struct {
	StationUUID          string    `json:"station_uuid"`
	WaterLevelUUID       string    `json:"waterlevel_uuid"`
	Name                 string    `json:"name"`
	ShortName            string    `json:"short_name"`
	HolderName           string    `json:"holder_name"`
	Area                 string    `json:"area"`
	District             string    `json:"district"`
	Locality             *string   `json:"locality,omitempty"`
	MonitoringObject     string    `json:"monitoring_object"`
	Latitude             *float64  `json:"latitude,omitempty"`
	Longitude            *float64  `json:"longitude,omitempty"`
	FixBSM               *float32  `json:"fix_bs_m,omitempty"`
	DryBSM               *float32  `json:"dry_bs_m,omitempty"`
	FloodingPreventionBM *float32  `json:"flooding_prevention_bs_m,omitempty"`
	FloodingDangerBSM    *float32  `json:"flooding_danger_bs_m,omitempty"`
	FetchedAt            time.Time `json:"fetched_at"`
}

// HydroLevelReading — измерение уровня воды.
type HydroLevelReading struct {
	StationUUID     string    `json:"station_uuid"`
	WaterLevelUUID  string    `json:"waterlevel_uuid"`
	ObservedAt      time.Time `json:"observed_at"`
	LevelBSM        float32   `json:"level_bs_m"` // метры, Балтийская система высот
	LevelZeroM      *float32  `json:"level_zero_m,omitempty"`
	ChangeCmPerHour *float32  `json:"change_cm_per_hour,omitempty"`
	LeadText        *string   `json:"lead_text,omitempty"`
	StateCode       *int      `json:"state_code,omitempty"`
	LevelCode       *int      `json:"level_code,omitempty"`
	RawData         []byte    `json:"raw_data,omitempty"`
	FetchedAt       time.Time `json:"fetched_at"`
}

// HydroSnapshot — данные для карточки/текущего состояния.
type HydroSnapshot struct {
	Gauge           *HydroGauge        `json:"gauge,omitempty"`
	Current         *HydroLevelReading `json:"current,omitempty"`
	Previous        *HydroLevelReading `json:"previous,omitempty"`
	DayAgo          *HydroLevelReading `json:"day_ago,omitempty"`
	ChangeM         *float32           `json:"change_m,omitempty"`
	Change24hM      *float32           `json:"change_24h_m,omitempty"`
	RelativeLevelCm *float32           `json:"relative_level_cm,omitempty"`
	ToPreventionM   *float32           `json:"to_prevention_m,omitempty"`
	ToDangerM       *float32           `json:"to_danger_m,omitempty"`
	Status          HydroStatus        `json:"status"`
	HasData         bool               `json:"has_data"`
}

type HydroStatus string

const (
	HydroStatusUnknown    HydroStatus = "unknown"
	HydroStatusNormal     HydroStatus = "normal"
	HydroStatusPrevention HydroStatus = "prevention"
	HydroStatusDanger     HydroStatus = "danger"
)

func ClassifyHydroLevel(level float32, prevention, danger *float32) HydroStatus {
	if danger != nil && level >= *danger {
		return HydroStatusDanger
	}
	if prevention != nil && level >= *prevention {
		return HydroStatusPrevention
	}
	return HydroStatusNormal
}

func (s HydroStatus) Label() string {
	switch s {
	case HydroStatusDanger:
		return "опасный уровень"
	case HydroStatusPrevention:
		return "неблагоприятный уровень"
	case HydroStatusNormal:
		return "норма"
	default:
		return "нет данных"
	}
}

func (s HydroStatus) TailwindGradient() string {
	switch s {
	case HydroStatusDanger:
		return "from-red-50 to-red-100 dark:from-red-900/20 dark:to-red-800/20"
	case HydroStatusPrevention:
		return "from-orange-50 to-orange-100 dark:from-orange-900/20 dark:to-orange-800/20"
	case HydroStatusNormal:
		return "from-blue-50 to-cyan-100 dark:from-blue-900/20 dark:to-cyan-800/20"
	default:
		return "from-gray-50 to-gray-100 dark:from-gray-800 dark:to-gray-700"
	}
}

func (s HydroStatus) TextColor() string {
	switch s {
	case HydroStatusDanger:
		return "text-red-700 dark:text-red-300"
	case HydroStatusPrevention:
		return "text-orange-700 dark:text-orange-300"
	case HydroStatusNormal:
		return "text-blue-700 dark:text-blue-300"
	default:
		return "text-gray-700 dark:text-gray-300"
	}
}

func (s HydroStatus) HexColor() string {
	switch s {
	case HydroStatusDanger:
		return "#ef4444"
	case HydroStatusPrevention:
		return "#f59e0b"
	case HydroStatusNormal:
		return "#0ea5e9"
	default:
		return "#6b7280"
	}
}
