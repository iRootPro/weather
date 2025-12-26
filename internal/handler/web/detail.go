package web

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

// DetailTemperature renders the detailed temperature page
func (h *Handler) DetailTemperature(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current data with hourly change
	current, hourAgo, dailyMinMax, err := h.weatherService.GetCurrentWithHourlyChange(ctx)
	if err != nil {
		slog.Error("failed to get current weather", "error", err)
		http.Error(w, "Failed to load weather data", http.StatusInternalServerError)
		return
	}

	// Get data from 24 hours ago for daily change
	dayAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		slog.Warn("failed to get day ago data", "error", err)
	}

	// Get data from week ago for weekly change
	weekAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		slog.Warn("failed to get week ago data", "error", err)
	}

	// Get records
	records, err := h.weatherService.GetRecords(ctx)
	if err != nil {
		slog.Error("failed to get records", "error", err)
		http.Error(w, "Failed to load records", http.StatusInternalServerError)
		return
	}

	// Get chart data for different periods
	now := time.Now()

	// 24 hours chart
	chart24h, err := h.weatherService.GetHistory(ctx, now.Add(-24*time.Hour), now, "5min")
	if err != nil {
		slog.Error("failed to get 24h chart data", "error", err)
	}

	// 7 days chart
	chart7d, err := h.weatherService.GetHistory(ctx, now.Add(-7*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 7d chart data", "error", err)
	}

	// 30 days chart
	chart30d, err := h.weatherService.GetHistory(ctx, now.Add(-30*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 30d chart data", "error", err)
	}

	// Prepare template data
	templateData := struct {
		ActivePage string
		Data       interface{}
	}{
		ActivePage: "dashboard",
		Data: map[string]interface{}{
			// Current readings
			"Current":     getFloat32Value(current.TempOutdoor),
			"FeelsLike":   getFloat32Value(current.TempFeelsLike),
			"DewPoint":    getFloat32Value(current.DewPoint),
			"IndoorTemp":  getFloat32Value(current.TempIndoor),
			"UpdateTime":  current.Time.Format("15:04"),
			"UpdateDate":  current.Time.Format("2 января 2006"),

			// Calculate differences
			"IndoorDiff": calculateDiff(current.TempOutdoor, current.TempIndoor),

			// Changes
			"ChangeHour":  calculateTempChange(current.TempOutdoor, hourAgo),
			"ChangeDay":   calculateTempChange(current.TempOutdoor, dayAgo),
			"ChangeWeek":  calculateTempChange(current.TempOutdoor, weekAgo),
			"HasHourData": hourAgo != nil && hourAgo.TempOutdoor != nil,
			"HasDayData":  dayAgo != nil && dayAgo.TempOutdoor != nil,
			"HasWeekData": weekAgo != nil && weekAgo.TempOutdoor != nil,

			// Today's stats
			"TodayMin":       getFloat32Value(dailyMinMax.TempMin),
			"TodayMax":       getFloat32Value(dailyMinMax.TempMax),
			"TodayAvg":       calculateAvgFromMinMax(dailyMinMax.TempMin, dailyMinMax.TempMax),
			"TodayAmplitude": calculateAmplitude(dailyMinMax.TempMin, dailyMinMax.TempMax),
			"HasDailyData":   dailyMinMax != nil,

			// Records
			"RecordMax":     records.TempOutdoorMax.Value,
			"RecordMaxTime": records.TempOutdoorMax.Time.Format("2 января 2006, 15:04"),
			"RecordMin":     records.TempOutdoorMin.Value,
			"RecordMinTime": records.TempOutdoorMin.Time.Format("2 января 2006, 15:04"),

			// Chart data (as JSON)
			"Chart24h":  toJSON(prepareChartData(chart24h, "TempOutdoor")),
			"Chart7d":   toJSON(prepareChartData(chart7d, "TempOutdoor")),
			"Chart30d":  toJSON(prepareChartData(chart30d, "TempOutdoor")),
			"HasCharts": len(chart24h) > 0,
		},
	}

	tmpl, err := h.parseTemplate("detail/temperature.html")
	if err != nil {
		slog.Error("failed to parse temperature detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render temperature detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Helper functions

func getFloat32Value(ptr *float32) float32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func calculateDiff(a, b *float32) float32 {
	if a == nil || b == nil {
		return 0
	}
	return *a - *b
}

func calculateTempChange(current *float32, oldData *models.WeatherData) float32 {
	if current == nil || oldData == nil || oldData.TempOutdoor == nil {
		return 0
	}
	return *current - *oldData.TempOutdoor
}

func calculateAvgFromMinMax(min, max *float32) float32 {
	if min == nil || max == nil {
		return 0
	}
	return (*min + *max) / 2
}

func calculateAmplitude(min, max *float32) float32 {
	if min == nil || max == nil {
		return 0
	}
	return *max - *min
}

func prepareChartData(data []models.WeatherData, field string) map[string]interface{} {
	labels := make([]string, 0, len(data))
	temps := make([]float64, 0, len(data))
	feelsLike := make([]float64, 0, len(data))
	dewPoint := make([]float64, 0, len(data))

	for _, item := range data {
		// Format time label
		labels = append(labels, item.Time.Format("15:04"))

		// Temperature
		if item.TempOutdoor != nil {
			temps = append(temps, float64(*item.TempOutdoor))
		} else {
			temps = append(temps, 0)
		}

		// Feels like
		if item.TempFeelsLike != nil {
			feelsLike = append(feelsLike, float64(*item.TempFeelsLike))
		} else {
			feelsLike = append(feelsLike, 0)
		}

		// Dew point
		if item.DewPoint != nil {
			dewPoint = append(dewPoint, float64(*item.DewPoint))
		} else {
			dewPoint = append(dewPoint, 0)
		}
	}

	return map[string]interface{}{
		"labels":    labels,
		"temps":     temps,
		"feelsLike": feelsLike,
		"dewPoint":  dewPoint,
	}
}

func toJSON(data map[string]interface{}) template.JS {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal chart data", "error", err)
		return template.JS("{}")
	}
	return template.JS(jsonBytes)
}

// DetailHumidity renders the detailed humidity page
func (h *Handler) DetailHumidity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current data with hourly change
	current, hourAgo, dailyMinMax, err := h.weatherService.GetCurrentWithHourlyChange(ctx)
	if err != nil {
		slog.Error("failed to get current weather", "error", err)
		http.Error(w, "Failed to load weather data", http.StatusInternalServerError)
		return
	}

	// Get data from 24 hours ago for daily change
	dayAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		slog.Warn("failed to get day ago data", "error", err)
	}

	// Get data from week ago for weekly change
	weekAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		slog.Warn("failed to get week ago data", "error", err)
	}

	// Get records
	records, err := h.weatherService.GetRecords(ctx)
	if err != nil {
		slog.Error("failed to get records", "error", err)
		http.Error(w, "Failed to load records", http.StatusInternalServerError)
		return
	}

	// Get chart data for different periods
	now := time.Now()

	// 24 hours chart
	chart24h, err := h.weatherService.GetHistory(ctx, now.Add(-24*time.Hour), now, "5min")
	if err != nil {
		slog.Error("failed to get 24h chart data", "error", err)
	}

	// 7 days chart
	chart7d, err := h.weatherService.GetHistory(ctx, now.Add(-7*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 7d chart data", "error", err)
	}

	// 30 days chart
	chart30d, err := h.weatherService.GetHistory(ctx, now.Add(-30*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 30d chart data", "error", err)
	}

	// Prepare template data
	templateData := struct {
		ActivePage string
		Data       interface{}
	}{
		ActivePage: "dashboard",
		Data: map[string]interface{}{
			// Current readings
			"Current":    getInt16Value(current.HumidityOutdoor),
			"DewPoint":   getFloat32Value(current.DewPoint),
			"UpdateTime": current.Time.Format("15:04"),
			"UpdateDate": current.Time.Format("2 января 2006"),

			// Changes
			"ChangeHour": calculateHumidityChange(current.HumidityOutdoor, hourAgo),
			"ChangeDay":  calculateHumidityChange(current.HumidityOutdoor, dayAgo),
			"ChangeWeek": calculateHumidityChange(current.HumidityOutdoor, weekAgo),
			"HasHourData": hourAgo != nil && hourAgo.HumidityOutdoor != nil,
			"HasDayData":  dayAgo != nil && dayAgo.HumidityOutdoor != nil,
			"HasWeekData": weekAgo != nil && weekAgo.HumidityOutdoor != nil,

			// Today's stats
			"TodayMin":       getInt16Value(dailyMinMax.HumidityMin),
			"TodayMax":       getInt16Value(dailyMinMax.HumidityMax),
			"TodayAvg":       calculateAvgFromMinMaxInt16(dailyMinMax.HumidityMin, dailyMinMax.HumidityMax),
			"TodayAmplitude": calculateAmplitudeInt16(dailyMinMax.HumidityMin, dailyMinMax.HumidityMax),
			"HasDailyData":   dailyMinMax != nil,

			// Records
			"RecordMax":     records.HumidityOutdoorMax.Value,
			"RecordMaxTime": records.HumidityOutdoorMax.Time.Format("2 января 2006, 15:04"),
			"RecordMin":     records.HumidityOutdoorMin.Value,
			"RecordMinTime": records.HumidityOutdoorMin.Time.Format("2 января 2006, 15:04"),

			// Chart data (as JSON)
			"Chart24h":  toJSON(prepareHumidityChartData(chart24h)),
			"Chart7d":   toJSON(prepareHumidityChartData(chart7d)),
			"Chart30d":  toJSON(prepareHumidityChartData(chart30d)),
			"HasCharts": len(chart24h) > 0,
		},
	}

	tmpl, err := h.parseTemplate("detail/humidity.html")
	if err != nil {
		slog.Error("failed to parse humidity detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render humidity detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// DetailPressure renders the detailed pressure page
func (h *Handler) DetailPressure(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current data with hourly change
	current, hourAgo, dailyMinMax, err := h.weatherService.GetCurrentWithHourlyChange(ctx)
	if err != nil {
		slog.Error("failed to get current weather", "error", err)
		http.Error(w, "Failed to load weather data", http.StatusInternalServerError)
		return
	}

	// Get data from 24 hours ago for daily change
	dayAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		slog.Warn("failed to get day ago data", "error", err)
	}

	// Get data from week ago for weekly change
	weekAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		slog.Warn("failed to get week ago data", "error", err)
	}

	// Get records
	records, err := h.weatherService.GetRecords(ctx)
	if err != nil {
		slog.Error("failed to get records", "error", err)
		http.Error(w, "Failed to load records", http.StatusInternalServerError)
		return
	}

	// Get chart data for different periods
	now := time.Now()

	// 24 hours chart
	chart24h, err := h.weatherService.GetHistory(ctx, now.Add(-24*time.Hour), now, "5min")
	if err != nil {
		slog.Error("failed to get 24h chart data", "error", err)
	}

	// 7 days chart
	chart7d, err := h.weatherService.GetHistory(ctx, now.Add(-7*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 7d chart data", "error", err)
	}

	// 30 days chart
	chart30d, err := h.weatherService.GetHistory(ctx, now.Add(-30*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 30d chart data", "error", err)
	}

	// Prepare template data
	templateData := struct {
		ActivePage string
		Data       interface{}
	}{
		ActivePage: "dashboard",
		Data: map[string]interface{}{
			// Current readings
			"Current":    getFloat32Value(current.PressureRelative),
			"Absolute":   getFloat32Value(current.PressureAbsolute),
			"UpdateTime": current.Time.Format("15:04"),
			"UpdateDate": current.Time.Format("2 января 2006"),

			// Changes
			"ChangeHour": calculatePressureChange(current.PressureRelative, hourAgo),
			"ChangeDay":  calculatePressureChange(current.PressureRelative, dayAgo),
			"ChangeWeek": calculatePressureChange(current.PressureRelative, weekAgo),
			"HasHourData": hourAgo != nil && hourAgo.PressureRelative != nil,
			"HasDayData":  dayAgo != nil && dayAgo.PressureRelative != nil,
			"HasWeekData": weekAgo != nil && weekAgo.PressureRelative != nil,

			// Today's stats
			"TodayMin":       getFloat32Value(dailyMinMax.PressureMin),
			"TodayMax":       getFloat32Value(dailyMinMax.PressureMax),
			"TodayAvg":       calculateAvgFromMinMax(dailyMinMax.PressureMin, dailyMinMax.PressureMax),
			"TodayAmplitude": calculateAmplitude(dailyMinMax.PressureMin, dailyMinMax.PressureMax),
			"HasDailyData":   dailyMinMax != nil,

			// Records
			"RecordMax":     records.PressureMax.Value,
			"RecordMaxTime": records.PressureMax.Time.Format("2 января 2006, 15:04"),
			"RecordMin":     records.PressureMin.Value,
			"RecordMinTime": records.PressureMin.Time.Format("2 января 2006, 15:04"),

			// Chart data (as JSON)
			"Chart24h":  toJSON(preparePressureChartData(chart24h)),
			"Chart7d":   toJSON(preparePressureChartData(chart7d)),
			"Chart30d":  toJSON(preparePressureChartData(chart30d)),
			"HasCharts": len(chart24h) > 0,
		},
	}

	tmpl, err := h.parseTemplate("detail/pressure.html")
	if err != nil {
		slog.Error("failed to parse pressure detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render pressure detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// DetailWind renders the detailed wind page
func (h *Handler) DetailWind(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current data with hourly change
	current, hourAgo, dailyMinMax, err := h.weatherService.GetCurrentWithHourlyChange(ctx)
	if err != nil {
		slog.Error("failed to get current weather", "error", err)
		http.Error(w, "Failed to load weather data", http.StatusInternalServerError)
		return
	}

	// Get data from 24 hours ago for daily change
	dayAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		slog.Warn("failed to get day ago data", "error", err)
	}

	// Get data from week ago for weekly change
	weekAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		slog.Warn("failed to get week ago data", "error", err)
	}

	// Get records
	records, err := h.weatherService.GetRecords(ctx)
	if err != nil {
		slog.Error("failed to get records", "error", err)
		http.Error(w, "Failed to load records", http.StatusInternalServerError)
		return
	}

	// Get chart data for different periods
	now := time.Now()

	// 24 hours chart
	chart24h, err := h.weatherService.GetHistory(ctx, now.Add(-24*time.Hour), now, "5min")
	if err != nil {
		slog.Error("failed to get 24h chart data", "error", err)
	}

	// 7 days chart
	chart7d, err := h.weatherService.GetHistory(ctx, now.Add(-7*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 7d chart data", "error", err)
	}

	// 30 days chart
	chart30d, err := h.weatherService.GetHistory(ctx, now.Add(-30*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 30d chart data", "error", err)
	}

	// Calculate wind direction
	windDir := ""
	if current.WindDirection != nil {
		windDir = getWindDirectionStr(int32(*current.WindDirection))
	}

	// Prepare template data
	templateData := struct {
		ActivePage string
		Data       interface{}
	}{
		ActivePage: "dashboard",
		Data: map[string]interface{}{
			// Current readings
			"Current":          getFloat32Value(current.WindSpeed),
			"Gust":             getFloat32Value(current.WindGust),
			"Direction":        getInt16Value(current.WindDirection),
			"DirectionStr":     windDir,
			"UpdateTime":       current.Time.Format("15:04"),
			"UpdateDate":       current.Time.Format("2 января 2006"),

			// Changes
			"ChangeHour": calculateWindChange(current.WindSpeed, hourAgo),
			"ChangeDay":  calculateWindChange(current.WindSpeed, dayAgo),
			"ChangeWeek": calculateWindChange(current.WindSpeed, weekAgo),
			"HasHourData": hourAgo != nil && hourAgo.WindSpeed != nil,
			"HasDayData":  dayAgo != nil && dayAgo.WindSpeed != nil,
			"HasWeekData": weekAgo != nil && weekAgo.WindSpeed != nil,

			// Today's stats
			"TodayMax":         getFloat32Value(dailyMinMax.WindMax),
			"TodayGustMax":     getFloat32Value(dailyMinMax.GustMax),
			"HasDailyData":     dailyMinMax != nil,

			// Records
			"RecordSpeed":     records.WindSpeedMax.Value,
			"RecordSpeedTime": records.WindSpeedMax.Time.Format("2 января 2006, 15:04"),
			"RecordGust":      records.WindGustMax.Value,
			"RecordGustTime":  records.WindGustMax.Time.Format("2 января 2006, 15:04"),

			// Chart data (as JSON)
			"Chart24h":  toJSON(prepareWindChartData(chart24h)),
			"Chart7d":   toJSON(prepareWindChartData(chart7d)),
			"Chart30d":  toJSON(prepareWindChartData(chart30d)),
			"HasCharts": len(chart24h) > 0,
		},
	}

	tmpl, err := h.parseTemplate("detail/wind.html")
	if err != nil {
		slog.Error("failed to parse wind detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render wind detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// DetailRain renders the detailed rain page
func (h *Handler) DetailRain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current data with hourly change
	current, hourAgo, dailyMinMax, err := h.weatherService.GetCurrentWithHourlyChange(ctx)
	if err != nil {
		slog.Error("failed to get current weather", "error", err)
		http.Error(w, "Failed to load weather data", http.StatusInternalServerError)
		return
	}

	// Get data from 24 hours ago for daily change
	dayAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		slog.Warn("failed to get day ago data", "error", err)
	}

	// Get data from week ago for weekly change
	weekAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		slog.Warn("failed to get week ago data", "error", err)
	}

	// Get records
	records, err := h.weatherService.GetRecords(ctx)
	if err != nil {
		slog.Error("failed to get records", "error", err)
		http.Error(w, "Failed to load records", http.StatusInternalServerError)
		return
	}

	// Get chart data for different periods
	now := time.Now()

	// 24 hours chart
	chart24h, err := h.weatherService.GetHistory(ctx, now.Add(-24*time.Hour), now, "5min")
	if err != nil {
		slog.Error("failed to get 24h chart data", "error", err)
	}

	// 7 days chart
	chart7d, err := h.weatherService.GetHistory(ctx, now.Add(-7*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 7d chart data", "error", err)
	}

	// 30 days chart
	chart30d, err := h.weatherService.GetHistory(ctx, now.Add(-30*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 30d chart data", "error", err)
	}

	// Prepare template data
	templateData := struct {
		ActivePage string
		Data       interface{}
	}{
		ActivePage: "dashboard",
		Data: map[string]interface{}{
			// Current readings
			"Daily":      getFloat32Value(current.RainDaily),
			"Monthly":    getFloat32Value(current.RainMonthly),
			"Rate":       getFloat32Value(current.RainRate),
			"UpdateTime": current.Time.Format("15:04"),
			"UpdateDate": current.Time.Format("2 января 2006"),

			// Changes (for daily rain)
			"ChangeHour": calculateRainChange(current.RainDaily, hourAgo),
			"ChangeDay":  calculateRainChange(current.RainDaily, dayAgo),
			"ChangeWeek": calculateRainChange(current.RainDaily, weekAgo),
			"HasHourData": hourAgo != nil && hourAgo.RainDaily != nil,
			"HasDayData":  dayAgo != nil && dayAgo.RainDaily != nil,
			"HasWeekData": weekAgo != nil && weekAgo.RainDaily != nil,

			// Today's stats - No daily max rain rate in DailyMinMax
			"HasDailyData": dailyMinMax != nil,

			// Records
			"RecordRate":     records.RainDailyMax.Value,
			"RecordRateTime": records.RainDailyMax.Time.Format("2 января 2006, 15:04"),

			// Chart data (as JSON)
			"Chart24h":  toJSON(prepareRainChartData(chart24h)),
			"Chart7d":   toJSON(prepareRainChartData(chart7d)),
			"Chart30d":  toJSON(prepareRainChartData(chart30d)),
			"HasCharts": len(chart24h) > 0,
		},
	}

	tmpl, err := h.parseTemplate("detail/rain.html")
	if err != nil {
		slog.Error("failed to parse rain detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render rain detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// DetailSolar renders the detailed solar radiation page
func (h *Handler) DetailSolar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current data with hourly change
	current, hourAgo, dailyMinMax, err := h.weatherService.GetCurrentWithHourlyChange(ctx)
	if err != nil {
		slog.Error("failed to get current weather", "error", err)
		http.Error(w, "Failed to load weather data", http.StatusInternalServerError)
		return
	}

	// Get data from 24 hours ago for daily change
	dayAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		slog.Warn("failed to get day ago data", "error", err)
	}

	// Get data from week ago for weekly change
	weekAgo, err := h.weatherService.GetDataAt(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		slog.Warn("failed to get week ago data", "error", err)
	}

	// Get records
	records, err := h.weatherService.GetRecords(ctx)
	if err != nil {
		slog.Error("failed to get records", "error", err)
		http.Error(w, "Failed to load records", http.StatusInternalServerError)
		return
	}

	// Get chart data for different periods
	now := time.Now()

	// 24 hours chart
	chart24h, err := h.weatherService.GetHistory(ctx, now.Add(-24*time.Hour), now, "5min")
	if err != nil {
		slog.Error("failed to get 24h chart data", "error", err)
	}

	// 7 days chart
	chart7d, err := h.weatherService.GetHistory(ctx, now.Add(-7*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 7d chart data", "error", err)
	}

	// 30 days chart
	chart30d, err := h.weatherService.GetHistory(ctx, now.Add(-30*24*time.Hour), now, "1hour")
	if err != nil {
		slog.Error("failed to get 30d chart data", "error", err)
	}

	// Prepare template data
	templateData := struct {
		ActivePage string
		Data       interface{}
	}{
		ActivePage: "dashboard",
		Data: map[string]interface{}{
			// Current readings
			"SolarRadiation": getFloat32Value(current.SolarRadiation),
			"UVIndex":        getFloat32Value(current.UVIndex),
			"UpdateTime":     current.Time.Format("15:04"),
			"UpdateDate":     current.Time.Format("2 января 2006"),

			// Changes (for solar radiation)
			"ChangeHour": calculateSolarChange(current.SolarRadiation, hourAgo),
			"ChangeDay":  calculateSolarChange(current.SolarRadiation, dayAgo),
			"ChangeWeek": calculateSolarChange(current.SolarRadiation, weekAgo),
			"HasHourData": hourAgo != nil && hourAgo.SolarRadiation != nil,
			"HasDayData":  dayAgo != nil && dayAgo.SolarRadiation != nil,
			"HasWeekData": weekAgo != nil && weekAgo.SolarRadiation != nil,

			// Today's stats - No daily max in DailyMinMax for solar
			"HasDailyData":  dailyMinMax != nil,

			// Records
			"RecordSolar":     records.SolarRadiationMax.Value,
			"RecordSolarTime": records.SolarRadiationMax.Time.Format("2 января 2006, 15:04"),
			"RecordUV":        records.UVIndexMax.Value,
			"RecordUVTime":    records.UVIndexMax.Time.Format("2 января 2006, 15:04"),

			// Chart data (as JSON)
			"Chart24h":  toJSON(prepareSolarChartData(chart24h)),
			"Chart7d":   toJSON(prepareSolarChartData(chart7d)),
			"Chart30d":  toJSON(prepareSolarChartData(chart30d)),
			"HasCharts": len(chart24h) > 0,
		},
	}

	tmpl, err := h.parseTemplate("detail/solar.html")
	if err != nil {
		slog.Error("failed to parse solar detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, templateData); err != nil {
		slog.Error("failed to render solar detail", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Helper functions for new handlers

func getInt16Value(ptr *int16) int16 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func calculateAvgFromMinMaxInt16(min, max *int16) int16 {
	if min == nil || max == nil {
		return 0
	}
	return (*min + *max) / 2
}

func calculateAmplitudeInt16(min, max *int16) int16 {
	if min == nil || max == nil {
		return 0
	}
	return *max - *min
}

func calculateHumidityChange(current *int16, oldData *models.WeatherData) int16 {
	if current == nil || oldData == nil || oldData.HumidityOutdoor == nil {
		return 0
	}
	return *current - *oldData.HumidityOutdoor
}

func calculatePressureChange(current *float32, oldData *models.WeatherData) float32 {
	if current == nil || oldData == nil || oldData.PressureRelative == nil {
		return 0
	}
	return *current - *oldData.PressureRelative
}

func calculateWindChange(current *float32, oldData *models.WeatherData) float32 {
	if current == nil || oldData == nil || oldData.WindSpeed == nil {
		return 0
	}
	return *current - *oldData.WindSpeed
}

func calculateRainChange(current *float32, oldData *models.WeatherData) float32 {
	if current == nil || oldData == nil || oldData.RainDaily == nil {
		return 0
	}
	return *current - *oldData.RainDaily
}

func calculateSolarChange(current *float32, oldData *models.WeatherData) float32 {
	if current == nil || oldData == nil || oldData.SolarRadiation == nil {
		return 0
	}
	return *current - *oldData.SolarRadiation
}

func prepareHumidityChartData(data []models.WeatherData) map[string]interface{} {
	labels := make([]string, 0, len(data))
	humidity := make([]float64, 0, len(data))
	dewPoint := make([]float64, 0, len(data))

	for _, item := range data {
		labels = append(labels, item.Time.Format("15:04"))

		if item.HumidityOutdoor != nil {
			humidity = append(humidity, float64(*item.HumidityOutdoor))
		} else {
			humidity = append(humidity, 0)
		}

		if item.DewPoint != nil {
			dewPoint = append(dewPoint, float64(*item.DewPoint))
		} else {
			dewPoint = append(dewPoint, 0)
		}
	}

	return map[string]interface{}{
		"labels":   labels,
		"humidity": humidity,
		"dewPoint": dewPoint,
	}
}

func preparePressureChartData(data []models.WeatherData) map[string]interface{} {
	labels := make([]string, 0, len(data))
	pressure := make([]float64, 0, len(data))

	for _, item := range data {
		labels = append(labels, item.Time.Format("15:04"))

		if item.PressureRelative != nil {
			pressure = append(pressure, float64(*item.PressureRelative))
		} else {
			pressure = append(pressure, 0)
		}
	}

	return map[string]interface{}{
		"labels":   labels,
		"pressure": pressure,
	}
}

func prepareWindChartData(data []models.WeatherData) map[string]interface{} {
	labels := make([]string, 0, len(data))
	windSpeed := make([]float64, 0, len(data))
	windGust := make([]float64, 0, len(data))

	for _, item := range data {
		labels = append(labels, item.Time.Format("15:04"))

		if item.WindSpeed != nil {
			windSpeed = append(windSpeed, float64(*item.WindSpeed))
		} else {
			windSpeed = append(windSpeed, 0)
		}

		if item.WindGust != nil {
			windGust = append(windGust, float64(*item.WindGust))
		} else {
			windGust = append(windGust, 0)
		}
	}

	return map[string]interface{}{
		"labels":    labels,
		"windSpeed": windSpeed,
		"windGust":  windGust,
	}
}

func prepareRainChartData(data []models.WeatherData) map[string]interface{} {
	labels := make([]string, 0, len(data))
	rainDaily := make([]float64, 0, len(data))
	rainRate := make([]float64, 0, len(data))

	for _, item := range data {
		labels = append(labels, item.Time.Format("15:04"))

		if item.RainDaily != nil {
			rainDaily = append(rainDaily, float64(*item.RainDaily))
		} else {
			rainDaily = append(rainDaily, 0)
		}

		if item.RainRate != nil {
			rainRate = append(rainRate, float64(*item.RainRate))
		} else {
			rainRate = append(rainRate, 0)
		}
	}

	return map[string]interface{}{
		"labels":    labels,
		"rainDaily": rainDaily,
		"rainRate":  rainRate,
	}
}

func prepareSolarChartData(data []models.WeatherData) map[string]interface{} {
	labels := make([]string, 0, len(data))
	solarRadiation := make([]float64, 0, len(data))
	uvIndex := make([]float64, 0, len(data))

	for _, item := range data {
		labels = append(labels, item.Time.Format("15:04"))

		if item.SolarRadiation != nil {
			solarRadiation = append(solarRadiation, float64(*item.SolarRadiation))
		} else {
			solarRadiation = append(solarRadiation, 0)
		}

		if item.UVIndex != nil {
			uvIndex = append(uvIndex, float64(*item.UVIndex))
		} else {
			uvIndex = append(uvIndex, 0)
		}
	}

	return map[string]interface{}{
		"labels":         labels,
		"solarRadiation": solarRadiation,
		"uvIndex":        uvIndex,
	}
}

func getWindDirectionStr(degrees int32) string {
	directions := []string{"С", "ССВ", "СВ", "ВСВ", "В", "ВЮВ", "ЮВ", "ЮЮВ", "Ю", "ЮЮЗ", "ЮЗ", "ЗЮЗ", "З", "ЗСЗ", "СЗ", "ССЗ"}
	index := int((float64(degrees) + 11.25) / 22.5)
	return directions[index%16]
}
