package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/iRootPro/weather/pkg/ipgeolocation"
)

type MoonService struct {
	latitude       float64
	longitude      float64
	timezone       *time.Location
	astronomyClient *ipgeolocation.Client
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

func NewMoonService(latitude, longitude float64, timezone string, astronomyClient *ipgeolocation.Client) (*MoonService, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	return &MoonService{
		latitude:        latitude,
		longitude:       longitude,
		timezone:        loc,
		astronomyClient: astronomyClient,
	}, nil
}

func (m *MoonService) GetMoonData(date time.Time) *MoonData {
	date = date.In(m.timezone)

	// Calculate moon age and phase (our calculations are accurate)
	age := m.calcMoonAge(date)
	phase := m.calcMoonPhase(age)
	illumination := m.calcIllumination(age)

	// Get moonrise and moonset from API
	moonrise, moonset := m.getMoonriseMoonsetFromAPI(date)

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
// Uses a verified known new moon as reference
func (m *MoonService) calcMoonAge(date time.Time) float64 {
	// Known new moon: December 20, 2025, 01:43 UTC (verified from astronomical tables)
	// This new moon was BEFORE the full moon on January 3, 2026
	knownNewMoon := time.Date(2025, 12, 20, 1, 43, 0, 0, time.UTC)

	// Synodic month (average time between new moons)
	const synodicMonth = 29.53058867

	// Calculate days since or until known new moon
	daysDiff := date.Sub(knownNewMoon).Hours() / 24.0

	// Calculate current age in the lunar cycle
	age := math.Mod(daysDiff, synodicMonth)
	if age < 0 {
		age += synodicMonth
	}

	return age
}

// calcMoonPhase determines the moon phase based on age
func (m *MoonService) calcMoonPhase(age float64) MoonPhase {
	const synodicMonth = 29.53058867
	const newMoonCenter = 0.0
	const firstQuarterCenter = synodicMonth / 4.0      // ~7.38
	const fullMoonCenter = synodicMonth / 2.0          // ~14.765
	const lastQuarterCenter = synodicMonth * 3.0 / 4.0 // ~22.14

	// Very narrow window for key phases (±0.25 day = ±6 hours)
	const keyPhaseWindow = 0.25

	// Normalize age for end-of-cycle wrap-around
	normalizedAge := age
	if age > synodicMonth-keyPhaseWindow {
		normalizedAge = age - synodicMonth
	}

	// Check key phases first (in narrow windows)
	switch {
	case math.Abs(normalizedAge-newMoonCenter) <= keyPhaseWindow:
		return NewMoon
	case math.Abs(age-firstQuarterCenter) <= keyPhaseWindow:
		return FirstQuarter
	case math.Abs(age-fullMoonCenter) <= keyPhaseWindow:
		return FullMoon
	case math.Abs(age-lastQuarterCenter) <= keyPhaseWindow:
		return LastQuarter
	}

	// Transitional phases between key moments
	switch {
	case age < firstQuarterCenter:
		return WaxingCrescent
	case age < fullMoonCenter:
		return WaxingGibbous
	case age < lastQuarterCenter:
		return WaningGibbous
	default:
		return WaningCrescent
	}
}

// calcIllumination calculates accurate illumination percentage
func (m *MoonService) calcIllumination(age float64) float64 {
	// Standard astronomical formula for moon illumination
	// Based on the phase angle
	const synodicMonth = 29.53058867

	// Phase angle in radians (0 at new moon, π at full moon, 2π at next new moon)
	phaseAngle := (age / synodicMonth) * 2 * math.Pi

	// Illuminated fraction formula: (1 - cos(i)) / 2
	// where i is the phase angle
	illumination := (1 - math.Cos(phaseAngle)) / 2

	return illumination * 100.0
}

// getMoonriseMoonsetFromAPI получает точные времена восхода и захода луны из IPGeolocation API
func (m *MoonService) getMoonriseMoonsetFromAPI(date time.Time) (time.Time, time.Time) {
	// Если клиент не настроен, используем fallback расчёты
	if m.astronomyClient == nil {
		year, month, day := date.Date()
		age := m.calcMoonAge(date)
		return m.calcMoonriseMoonset(year, int(month), day, age)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := m.astronomyClient.GetAstronomy(ctx, ipgeolocation.AstronomyRequest{
		Latitude:  m.latitude,
		Longitude: m.longitude,
		Date:      date,
	})

	if err != nil {
		slog.Error("failed to get astronomy data from API, using fallback calculations",
			"error", err,
			"date", date.Format("2006-01-02"))

		// Fallback to our calculations
		year, month, day := date.Date()
		age := m.calcMoonAge(date)
		return m.calcMoonriseMoonset(year, int(month), day, age)
	}

	// Parse moonrise and moonset times
	year, month, day := date.Date()

	moonrise, err := m.parseTimeInDate(resp.Moonrise, year, int(month), day)
	if err != nil {
		slog.Warn("failed to parse moonrise time", "time", resp.Moonrise, "error", err)
		moonrise = time.Date(year, time.Month(month), day, 0, 0, 0, 0, m.timezone)
	}

	moonset, err := m.parseTimeInDate(resp.Moonset, year, int(month), day)
	if err != nil {
		slog.Warn("failed to parse moonset time", "time", resp.Moonset, "error", err)
		moonset = time.Date(year, time.Month(month), day, 23, 59, 59, 0, m.timezone)
	}

	return moonrise, moonset
}

// parseTimeInDate парсит время в формате "HH:MM" и создаёт time.Time для указанной даты
func (m *MoonService) parseTimeInDate(timeStr string, year, month, day int) (time.Time, error) {
	if timeStr == "" || timeStr == "-" {
		return time.Time{}, fmt.Errorf("empty or invalid time string")
	}

	// Parse time in format "HH:MM"
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse time: %w", err)
	}

	// Create time with the given date in the configured timezone
	return time.Date(year, time.Month(month), day, t.Hour(), t.Minute(), 0, 0, m.timezone), nil
}

// calcMoonriseMoonset calculates moonrise and moonset times (fallback method)
func (m *MoonService) calcMoonriseMoonset(year, month, day int, age float64) (time.Time, time.Time) {
	// Moon's orbital position calculation
	const synodicMonth = 29.53058867
	const moonDailyMotion = 360.0 / 27.32166 // Sidereal month in degrees per day

	// Moon's mean longitude based on age
	moonLongitude := age * moonDailyMotion

	// Moon's mean anomaly
	meanAnomaly := (age / synodicMonth) * 2 * math.Pi

	// Moon's ecliptic latitude (simplified)
	eclipticLat := 5.145 * math.Sin(2*meanAnomaly)

	// Earth's axial tilt
	const obliquity = 23.4397 * math.Pi / 180

	// Convert ecliptic to equatorial coordinates
	moonLongRad := moonLongitude * math.Pi / 180
	eclipticLatRad := eclipticLat * math.Pi / 180

	ra := math.Atan2(
		math.Sin(moonLongRad)*math.Cos(obliquity)-math.Tan(eclipticLatRad)*math.Sin(obliquity),
		math.Cos(moonLongRad),
	)

	dec := math.Asin(
		math.Sin(eclipticLatRad)*math.Cos(obliquity)+
			math.Cos(eclipticLatRad)*math.Sin(obliquity)*math.Sin(moonLongRad),
	)

	// Standard altitude for moonrise/moonset (including parallax and refraction)
	const h0 = (-0.833 - 0.9 - 0.6) * math.Pi / 180

	lat := m.latitude * math.Pi / 180
	cosH := (math.Sin(h0) - math.Sin(lat)*math.Sin(dec)) / (math.Cos(lat) * math.Cos(dec))

	var moonrise, moonset time.Time

	if cosH > 1 || cosH < -1 {
		// Moon doesn't rise or set today
		moonrise = time.Date(year, time.Month(month), day, 0, 0, 0, 0, m.timezone)
		moonset = time.Date(year, time.Month(month), day, 23, 59, 59, 0, m.timezone)
	} else {
		hourAngle := math.Acos(cosH) * 180 / math.Pi

		// Transit time (when moon crosses meridian)
		transitTime := 12.0 - (ra*180/math.Pi)/15.0 - m.longitude/15.0

		// Normalize transit time
		for transitTime < 0 {
			transitTime += 24
		}
		for transitTime >= 24 {
			transitTime -= 24
		}

		// Rise and set times
		riseTime := transitTime - hourAngle/15.0
		setTime := transitTime + hourAngle/15.0

		// Normalize times
		for riseTime < 0 {
			riseTime += 24
		}
		for riseTime >= 24 {
			riseTime -= 24
		}
		for setTime < 0 {
			setTime += 24
		}
		for setTime >= 24 {
			setTime -= 24
		}

		riseHour := int(riseTime)
		riseMin := int((riseTime - float64(riseHour)) * 60)
		setHour := int(setTime)
		setMin := int((setTime - float64(setHour)) * 60)

		moonrise = time.Date(year, time.Month(month), day, riseHour, riseMin, 0, 0, time.UTC)
		moonset = time.Date(year, time.Month(month), day, setHour, setMin, 0, 0, time.UTC)

		moonrise = moonrise.In(m.timezone)
		moonset = moonset.In(m.timezone)
	}

	return moonrise, moonset
}

// isMoonAboveHorizon checks if moon is currently above horizon
func (m *MoonService) isMoonAboveHorizon(date time.Time, moonrise time.Time, moonset time.Time) bool {
	if moonrise.Before(moonset) {
		return date.After(moonrise) && date.Before(moonset)
	}
	return date.After(moonrise) || date.Before(moonset)
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
