package service

import (
	"math"
	"time"
)

type MoonService struct {
	latitude  float64
	longitude float64
	timezone  *time.Location
}

type MoonPhase int

const (
	NewMoon MoonPhase = iota
	WaxingCrescent
	FirstQuarter
	WaxingGibbous
	FullMoon
	WaningGibbous
	LastQuarter
	WaningCrescent
)

type MoonData struct {
	Phase          MoonPhase
	PhaseName      string
	PhaseIcon      string
	Illumination   float64   // 0-100%
	Age            float64   // days since new moon (0-29.53)
	Moonrise       time.Time
	Moonset        time.Time
	IsAboveHorizon bool
}

func NewMoonService(latitude, longitude float64, timezone string) (*MoonService, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	return &MoonService{
		latitude:  latitude,
		longitude: longitude,
		timezone:  loc,
	}, nil
}

func (m *MoonService) GetMoonData(date time.Time) *MoonData {
	date = date.In(m.timezone)

	// Calculate moon age and phase
	age := m.calcMoonAge(date)
	phase := m.calcMoonPhase(age)
	illumination := m.calcIllumination(age)

	// Calculate moonrise and moonset
	year, month, day := date.Date()
	moonrise := m.calcMoonrise(year, int(month), day)
	moonset := m.calcMoonset(year, int(month), day)

	return &MoonData{
		Phase:          phase,
		PhaseName:      m.getPhaseName(phase),
		PhaseIcon:      m.getPhaseIcon(phase),
		Illumination:   illumination,
		Age:            age,
		Moonrise:       moonrise,
		Moonset:        moonset,
		IsAboveHorizon: m.isMoonAboveHorizon(date, moonrise, moonset),
	}
}

func (m *MoonService) GetTodayMoonData() *MoonData {
	return m.GetMoonData(time.Now())
}

// calcMoonAge calculates the age of the moon in days (0-29.53)
// Based on the synodic month (lunar phase cycle)
func (m *MoonService) calcMoonAge(date time.Time) float64 {
	// Known new moon: January 6, 2000, 18:14 UTC
	knownNewMoon := time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC)

	// Synodic month (average time between new moons)
	synodicMonth := 29.53058867

	// Calculate days since known new moon
	daysSince := date.Sub(knownNewMoon).Hours() / 24.0

	// Calculate current age in the lunar cycle
	age := math.Mod(daysSince, synodicMonth)
	if age < 0 {
		age += synodicMonth
	}

	return age
}

// calcMoonPhase determines the moon phase based on age
func (m *MoonService) calcMoonPhase(age float64) MoonPhase {
	// Key lunar phases centered around critical points:
	// New Moon: 0 days
	// First Quarter: 7.38 days (1/4 of cycle)
	// Full Moon: 14.765 days (1/2 of cycle)
	// Last Quarter: 22.14 days (3/4 of cycle)

	const synodicMonth = 29.53058867
	const phaseLength = synodicMonth / 8.0  // ~3.69 days each phase

	// Center each major phase around its key moment
	// Each phase extends ±half phaseLength from center
	const halfPhase = phaseLength / 2.0

	const newMoonCenter = 0.0
	const firstQuarterCenter = synodicMonth / 4.0      // ~7.38
	const fullMoonCenter = synodicMonth / 2.0          // ~14.765
	const lastQuarterCenter = synodicMonth * 3.0 / 4.0 // ~22.14

	// Normalize age for end-of-cycle wrap-around
	normalizedAge := age
	if age > synodicMonth - halfPhase {
		// Near end of cycle, treat as near new moon
		normalizedAge = age - synodicMonth
	}

	switch {
	case math.Abs(normalizedAge-newMoonCenter) <= halfPhase:
		return NewMoon
	case age < firstQuarterCenter-halfPhase:
		return WaxingCrescent
	case math.Abs(age-firstQuarterCenter) <= halfPhase:
		return FirstQuarter
	case age < fullMoonCenter-halfPhase:
		return WaxingGibbous
	case math.Abs(age-fullMoonCenter) <= halfPhase:
		return FullMoon
	case age < lastQuarterCenter-halfPhase:
		return WaningGibbous
	case math.Abs(age-lastQuarterCenter) <= halfPhase:
		return LastQuarter
	default:
		return WaningCrescent
	}
}

// calcIllumination calculates the percentage of the moon's visible disk that is illuminated
func (m *MoonService) calcIllumination(age float64) float64 {
	// The illumination follows a cosine curve
	synodicMonth := 29.53058867
	phase := (age / synodicMonth) * 2 * math.Pi

	// Calculate illumination (0 = new moon, 1 = full moon)
	illumination := (1 - math.Cos(phase)) / 2

	return illumination * 100
}

// getPhaseName returns the Russian name of the moon phase
func (m *MoonService) getPhaseName(phase MoonPhase) string {
	names := map[MoonPhase]string{
		NewMoon:         "Новолуние",
		WaxingCrescent:  "Растущая луна",
		FirstQuarter:    "Первая четверть",
		WaxingGibbous:   "Прибывающая луна",
		FullMoon:        "Полнолуние",
		WaningGibbous:   "Убывающая луна",
		LastQuarter:     "Последняя четверть",
		WaningCrescent:  "Стареющая луна",
	}
	return names[phase]
}

// getPhaseIcon returns emoji icon for the moon phase
func (m *MoonService) getPhaseIcon(phase MoonPhase) string {
	icons := map[MoonPhase]string{
		NewMoon:         "🌑",
		WaxingCrescent:  "🌒",
		FirstQuarter:    "🌓",
		WaxingGibbous:   "🌔",
		FullMoon:        "🌕",
		WaningGibbous:   "🌖",
		LastQuarter:     "🌗",
		WaningCrescent:  "🌘",
	}
	return icons[phase]
}

// calcMoonrise calculates moonrise time
func (m *MoonService) calcMoonrise(year, month, day int) time.Time {
	return m.calcMoonTime(year, month, day, true)
}

// calcMoonset calculates moonset time
func (m *MoonService) calcMoonset(year, month, day int) time.Time {
	return m.calcMoonTime(year, month, day, false)
}

// calcMoonTime calculates moonrise or moonset time
// This is a simplified calculation - for production use, consider a more accurate library
func (m *MoonService) calcMoonTime(year, month, day int, isRise bool) time.Time {
	// Calculate moon's position
	jd := m.julianDay(year, month, day)
	jc := (jd - 2451545.0) / 36525.0

	// Moon's mean longitude
	meanLong := 218.316 + 13.176396*jc*36525
	meanLong = math.Mod(meanLong, 360)

	// Moon's mean anomaly
	meanAnomaly := 134.963 + 13.064993*jc*36525
	meanAnomaly = math.Mod(meanAnomaly, 360)

	// Moon's argument of latitude
	argLat := 93.272 + 13.229350*jc*36525
	argLat = math.Mod(argLat, 360)

	// Calculate moon's longitude
	rad := func(deg float64) float64 { return deg * math.Pi / 180 }
	longitude := meanLong + 6.289*math.Sin(rad(meanAnomaly))

	// Calculate moon's latitude
	latitude := 5.128 * math.Sin(rad(argLat))

	// Calculate moon's declination (simplified)
	obliquity := 23.439 - 0.0000004*jc*36525
	sinDec := math.Sin(rad(latitude))*math.Cos(rad(obliquity)) +
	          math.Cos(rad(latitude))*math.Sin(rad(obliquity))*math.Sin(rad(longitude))
	declination := math.Asin(sinDec) * 180 / math.Pi

	// Calculate hour angle (similar to sun calculation but for moon)
	cosLat := math.Cos(rad(m.latitude))
	sinLat := math.Sin(rad(m.latitude))
	cosDec := math.Cos(rad(declination))
	sinDec2 := math.Sin(rad(declination))

	// Moon's parallax (approximately 0.9 degrees)
	cosH := (math.Sin(rad(-0.833-0.9)) - sinLat*sinDec2) / (cosLat * cosDec)

	// Check if moon rises/sets
	if cosH > 1 || cosH < -1 {
		// Moon doesn't rise or set today
		if isRise {
			return time.Date(year, time.Month(month), day, 0, 0, 0, 0, m.timezone)
		}
		return time.Date(year, time.Month(month), day, 23, 59, 59, 0, m.timezone)
	}

	hourAngle := math.Acos(cosH) * 180 / math.Pi
	if isRise {
		hourAngle = -hourAngle
	}

	// Calculate time
	// This is simplified - using approximate transit time
	transitTime := 12.0 + (m.longitude / 15.0)
	eventTime := transitTime + (hourAngle / 15.0)

	// Adjust for day boundaries
	for eventTime < 0 {
		eventTime += 24
	}
	for eventTime >= 24 {
		eventTime -= 24
	}

	hours := int(eventTime)
	minutes := int((eventTime - float64(hours)) * 60)

	return time.Date(year, time.Month(month), day, hours, minutes, 0, 0, m.timezone)
}

// isMoonAboveHorizon checks if moon is currently above horizon
func (m *MoonService) isMoonAboveHorizon(date time.Time, moonrise time.Time, moonset time.Time) bool {
	// Simple check: is current time between moonrise and moonset
	if moonrise.Before(moonset) {
		return date.After(moonrise) && date.Before(moonset)
	}
	// Moonset is before moonrise (moon is up during midnight)
	return date.After(moonrise) || date.Before(moonset)
}

// julianDay calculates Julian Day number (reuse from sun service logic)
func (m *MoonService) julianDay(year, month, day int) float64 {
	if month <= 2 {
		year--
		month += 12
	}

	a := year / 100
	b := 2 - a + a/4

	return float64(int(365.25*float64(year+4716))) +
		float64(int(30.6001*float64(month+1))) +
		float64(day) + float64(b) - 1524.5
}
