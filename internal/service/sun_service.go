package service

import (
	"math"
	"time"
)

type SunService struct {
	latitude  float64
	longitude float64
	timezone  *time.Location
}

type SunTimes struct {
	Dawn        time.Time
	Sunrise     time.Time
	Sunset      time.Time
	Dusk        time.Time
	DayLength   time.Duration
	LightLength time.Duration // from dawn to dusk
}

type SunTimesWithComparison struct {
	SunTimes
	// Day length changes
	DayChangeDay   time.Duration // compared to yesterday
	DayChangeWeek  time.Duration // compared to a week ago
	DayChangeMonth time.Duration // compared to a month ago
	// Light length changes
	LightChangeDay   time.Duration
	LightChangeWeek  time.Duration
	LightChangeMonth time.Duration
}

func NewSunService(latitude, longitude float64, timezone string) (*SunService, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	return &SunService{
		latitude:  latitude,
		longitude: longitude,
		timezone:  loc,
	}, nil
}

func (s *SunService) GetSunTimes(date time.Time) *SunTimes {
	// Use the date in the configured timezone
	date = date.In(s.timezone)
	year, month, day := date.Date()

	sunrise := s.calcSunrise(year, int(month), day)
	sunset := s.calcSunset(year, int(month), day)
	dawn := s.calcDawn(year, int(month), day)
	dusk := s.calcDusk(year, int(month), day)

	return &SunTimes{
		Dawn:        dawn,
		Sunrise:     sunrise,
		Sunset:      sunset,
		Dusk:        dusk,
		DayLength:   sunset.Sub(sunrise),
		LightLength: dusk.Sub(dawn),
	}
}

func (s *SunService) GetTodaySunTimes() *SunTimes {
	return s.GetSunTimes(time.Now())
}

func (s *SunService) GetTodaySunTimesWithComparison() *SunTimesWithComparison {
	now := time.Now()
	today := s.GetSunTimes(now)
	yesterday := s.GetSunTimes(now.AddDate(0, 0, -1))
	weekAgo := s.GetSunTimes(now.AddDate(0, 0, -7))
	monthAgo := s.GetSunTimes(now.AddDate(0, -1, 0))

	return &SunTimesWithComparison{
		SunTimes:         *today,
		DayChangeDay:     today.DayLength - yesterday.DayLength,
		DayChangeWeek:    today.DayLength - weekAgo.DayLength,
		DayChangeMonth:   today.DayLength - monthAgo.DayLength,
		LightChangeDay:   today.LightLength - yesterday.LightLength,
		LightChangeWeek:  today.LightLength - weekAgo.LightLength,
		LightChangeMonth: today.LightLength - monthAgo.LightLength,
	}
}

// calcSunrise calculates sunrise time using NOAA algorithm
func (s *SunService) calcSunrise(year, month, day int) time.Time {
	return s.calcSunTime(year, month, day, true, -0.833)
}

// calcSunset calculates sunset time using NOAA algorithm
func (s *SunService) calcSunset(year, month, day int) time.Time {
	return s.calcSunTime(year, month, day, false, -0.833)
}

// calcDawn calculates civil dawn (sun 6° below horizon)
func (s *SunService) calcDawn(year, month, day int) time.Time {
	return s.calcSunTime(year, month, day, true, -6.0)
}

// calcDusk calculates civil dusk (sun 6° below horizon)
func (s *SunService) calcDusk(year, month, day int) time.Time {
	return s.calcSunTime(year, month, day, false, -6.0)
}

// calcSunTime calculates sun event time using NOAA Solar Calculator algorithm
// isSunrise: true for sunrise/dawn, false for sunset/dusk
// zenith: angle below horizon (-0.833 for sunrise/sunset, -6 for civil twilight)
func (s *SunService) calcSunTime(year, month, day int, isSunrise bool, zenith float64) time.Time {
	// Convert to radians helper
	rad := func(deg float64) float64 { return deg * math.Pi / 180 }
	deg := func(r float64) float64 { return r * 180 / math.Pi }

	// Calculate Julian Day
	jd := s.julianDay(year, month, day)

	// Calculate Julian Century
	jc := (jd - 2451545.0) / 36525.0

	// Calculate sun's mean longitude (degrees)
	meanLong := math.Mod(280.46646+jc*(36000.76983+0.0003032*jc), 360)

	// Calculate sun's mean anomaly (degrees)
	meanAnomaly := 357.52911 + jc*(35999.05029-0.0001537*jc)

	// Calculate Earth's orbit eccentricity
	eccentricity := 0.016708634 - jc*(0.000042037+0.0000001267*jc)

	// Calculate sun's equation of center
	sinM := math.Sin(rad(meanAnomaly))
	sin2M := math.Sin(rad(2 * meanAnomaly))
	sin3M := math.Sin(rad(3 * meanAnomaly))
	eqCenter := sinM*(1.914602-jc*(0.004817+0.000014*jc)) +
		sin2M*(0.019993-0.000101*jc) +
		sin3M*0.000289

	// Calculate sun's true longitude
	trueLong := meanLong + eqCenter

	// Calculate sun's apparent longitude
	omega := 125.04 - 1934.136*jc
	appLong := trueLong - 0.00569 - 0.00478*math.Sin(rad(omega))

	// Calculate mean obliquity of ecliptic
	meanObliq := 23 + (26+((21.448-jc*(46.8150+jc*(0.00059-jc*0.001813))))/60)/60

	// Calculate corrected obliquity
	obliq := meanObliq + 0.00256*math.Cos(rad(omega))

	// Calculate sun's declination
	sinDec := math.Sin(rad(obliq)) * math.Sin(rad(appLong))
	declination := deg(math.Asin(sinDec))

	// Calculate equation of time (minutes)
	y := math.Tan(rad(obliq/2)) * math.Tan(rad(obliq/2))
	sin2L := math.Sin(2 * rad(meanLong))
	cos2L := math.Cos(2 * rad(meanLong))
	sin4L := math.Sin(4 * rad(meanLong))
	eqTime := 4 * deg(y*sin2L-2*eccentricity*sinM+4*eccentricity*y*sinM*cos2L-
		0.5*y*y*sin4L-1.25*eccentricity*eccentricity*sin2M)

	// Calculate hour angle for the given zenith
	cosZenith := math.Cos(rad(90 - zenith))
	cosLat := math.Cos(rad(s.latitude))
	sinLat := math.Sin(rad(s.latitude))
	cosDec := math.Cos(rad(declination))
	sinDecl := math.Sin(rad(declination))

	hourAngleCos := (cosZenith - sinLat*sinDecl) / (cosLat * cosDec)

	// Check if sun never rises or never sets at this location
	if hourAngleCos > 1 {
		// Sun never rises (polar night)
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, s.timezone)
	}
	if hourAngleCos < -1 {
		// Sun never sets (polar day)
		return time.Date(year, time.Month(month), day, 12, 0, 0, 0, s.timezone)
	}

	hourAngle := deg(math.Acos(hourAngleCos))

	// For sunrise, hour angle is negative
	if isSunrise {
		hourAngle = -hourAngle
	}

	// Calculate solar noon (in minutes from midnight UTC)
	solarNoon := 720 - 4*s.longitude - eqTime

	// Calculate event time (in minutes from midnight UTC)
	eventTime := solarNoon + hourAngle*4

	// Convert to hours and minutes
	hours := int(eventTime / 60)
	minutes := int(math.Mod(eventTime, 60))
	seconds := int((eventTime - float64(hours*60) - float64(minutes)) * 60)

	// Create time in UTC then convert to local timezone
	utcTime := time.Date(year, time.Month(month), day, hours, minutes, seconds, 0, time.UTC)
	return utcTime.In(s.timezone)
}

// julianDay calculates Julian Day number for a given date
func (s *SunService) julianDay(year, month, day int) float64 {
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

// GetSolarElevation calculates the sun's elevation angle above the horizon in degrees
// Returns negative values when sun is below horizon (night)
func (s *SunService) GetSolarElevation(t time.Time) float64 {
	// Helper functions
	rad := func(deg float64) float64 { return deg * math.Pi / 180 }
	deg := func(r float64) float64 { return r * 180 / math.Pi }

	// Use local timezone for date
	localTime := t.In(s.timezone)
	year, month, day := localTime.Date()

	// Use UTC time for solar calculations
	utcTime := t.UTC()
	hour, min, sec := utcTime.Clock()

	// Calculate decimal hour (in UTC)
	decimalHour := float64(hour) + float64(min)/60.0 + float64(sec)/3600.0

	// Calculate Julian Day
	jd := s.julianDay(year, int(month), day)
	// Add time of day
	jd += (decimalHour - 12.0) / 24.0

	// Calculate Julian Century
	jc := (jd - 2451545.0) / 36525.0

	// Calculate sun's mean longitude
	meanLong := math.Mod(280.46646+jc*(36000.76983+0.0003032*jc), 360)

	// Calculate sun's mean anomaly
	meanAnomaly := 357.52911 + jc*(35999.05029-0.0001537*jc)

	// Calculate sun's equation of center
	sinM := math.Sin(rad(meanAnomaly))
	sin2M := math.Sin(rad(2 * meanAnomaly))
	sin3M := math.Sin(rad(3 * meanAnomaly))
	eqCenter := sinM*(1.914602-jc*(0.004817+0.000014*jc)) +
		sin2M*(0.019993-0.000101*jc) +
		sin3M*0.000289

	// Calculate sun's true longitude
	trueLong := meanLong + eqCenter

	// Calculate sun's apparent longitude
	omega := 125.04 - 1934.136*jc
	appLong := trueLong - 0.00569 - 0.00478*math.Sin(rad(omega))

	// Calculate mean obliquity of ecliptic
	meanObliq := 23 + (26+((21.448-jc*(46.8150+jc*(0.00059-jc*0.001813))))/60)/60

	// Calculate corrected obliquity
	obliq := meanObliq + 0.00256*math.Cos(rad(omega))

	// Calculate sun's declination
	sinDec := math.Sin(rad(obliq)) * math.Sin(rad(appLong))
	declination := deg(math.Asin(sinDec))

	// Calculate equation of time (minutes)
	y := math.Tan(rad(obliq/2)) * math.Tan(rad(obliq/2))
	sin2L := math.Sin(2 * rad(meanLong))
	cos2L := math.Cos(2 * rad(meanLong))
	sin4L := math.Sin(4 * rad(meanLong))
	eqTime := 4 * deg(y*sin2L-2*0.016708634*sinM+4*0.016708634*y*sinM*cos2L-
		0.5*y*y*sin4L-1.25*0.016708634*0.016708634*sin2M)

	// Calculate local solar time
	offset := eqTime + 4*s.longitude
	trueSolarTime := decimalHour*60.0 + offset

	// Calculate hour angle
	hourAngle := trueSolarTime/4.0 - 180.0
	if hourAngle < -180 {
		hourAngle += 360
	}

	// Calculate solar zenith angle
	zenithCos := math.Sin(rad(s.latitude))*math.Sin(rad(declination)) +
		math.Cos(rad(s.latitude))*math.Cos(rad(declination))*math.Cos(rad(hourAngle))

	zenith := deg(math.Acos(zenithCos))

	// Convert to elevation (90 - zenith)
	elevation := 90.0 - zenith

	// Apply atmospheric refraction correction for low angles
	if elevation > -0.83 {
		// Approximate atmospheric refraction
		if elevation > 85 {
			// No correction needed
		} else if elevation > 5 {
			elevation += 0.0167 / math.Tan(rad(elevation+10.3/(elevation+5.11)))
		} else if elevation > -0.575 {
			elevation += 1.02 / math.Tan(rad(elevation+10.3/(elevation+5.11)))
		}
	}

	return elevation
}
