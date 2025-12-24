package mqtt

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

// Parser парсит сообщения от метеостанции EcoWitt
type Parser struct{}

// Поля для исключения из raw_data (технические данные)
var excludeFields = map[string]bool{
	"PASSKEY":      true,
	"passkey":      true,
	"runtime":      true,
	"interval":     true,
	"wh90batt":     true,
	"wh65batt":     true,
	"wh25batt":     true,
	"batt1":        true,
	"batt2":        true,
	"ws90_ver":     true,
	"dns_err_cnt":  true,
	"ws90cap_volt": true,
	"freq":         true,
}

func NewParser() *Parser {
	return &Parser{}
}

// Parse парсит сырые данные от EcoWitt и возвращает WeatherData
func (p *Parser) Parse(payload []byte) (*models.WeatherData, error) {
	// EcoWitt может отправлять данные в формате URL-encoded или JSON
	data, err := p.parsePayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	weather := &models.WeatherData{
		Time: time.Now().UTC(),
	}

	// Температура (°F -> °C)
	if v, ok := data["tempf"]; ok {
		if temp := p.parseFloatPtr(v); temp != nil {
			celsius := float32(models.FahrenheitToCelsius(float64(*temp)))
			weather.TempOutdoor = &celsius
		}
	}
	if v, ok := data["tempinf"]; ok {
		if temp := p.parseFloatPtr(v); temp != nil {
			celsius := float32(models.FahrenheitToCelsius(float64(*temp)))
			weather.TempIndoor = &celsius
		}
	}

	// Влажность (%)
	if v, ok := data["humidity"]; ok {
		weather.HumidityOutdoor = p.parseInt16Ptr(v)
	}
	if v, ok := data["humidityin"]; ok {
		weather.HumidityIndoor = p.parseInt16Ptr(v)
	}

	// Давление (inHg -> мм рт. ст.)
	if v, ok := data["baromrelin"]; ok {
		if press := p.parseFloatPtr(v); press != nil {
			mmhg := float32(float64(*press) * models.InHgToMmHg)
			weather.PressureRelative = &mmhg
		}
	}
	if v, ok := data["baromabsin"]; ok {
		if press := p.parseFloatPtr(v); press != nil {
			mmhg := float32(float64(*press) * models.InHgToMmHg)
			weather.PressureAbsolute = &mmhg
		}
	}

	// Ветер (mph -> м/с)
	if v, ok := data["windspeedmph"]; ok {
		if speed := p.parseFloatPtr(v); speed != nil {
			ms := float32(float64(*speed) * models.MphToMs)
			weather.WindSpeed = &ms
		}
	}
	if v, ok := data["windgustmph"]; ok {
		if gust := p.parseFloatPtr(v); gust != nil {
			ms := float32(float64(*gust) * models.MphToMs)
			weather.WindGust = &ms
		}
	}
	if v, ok := data["winddir"]; ok {
		weather.WindDirection = p.parseInt16Ptr(v)
	}

	// Осадки (in -> мм) - сначала пробуем piezo датчик, потом обычный
	if v, ok := data["rrain_piezo"]; ok {
		if rate := p.parseFloatPtr(v); rate != nil {
			mm := float32(float64(*rate) * models.InToMm)
			weather.RainRate = &mm
		}
	} else if v, ok := data["rainratein"]; ok {
		if rate := p.parseFloatPtr(v); rate != nil {
			mm := float32(float64(*rate) * models.InToMm)
			weather.RainRate = &mm
		}
	}

	if v, ok := data["drain_piezo"]; ok {
		if daily := p.parseFloatPtr(v); daily != nil {
			mm := float32(float64(*daily) * models.InToMm)
			weather.RainDaily = &mm
		}
	} else if v, ok := data["dailyrainin"]; ok {
		if daily := p.parseFloatPtr(v); daily != nil {
			mm := float32(float64(*daily) * models.InToMm)
			weather.RainDaily = &mm
		}
	}

	if v, ok := data["wrain_piezo"]; ok {
		if weekly := p.parseFloatPtr(v); weekly != nil {
			mm := float32(float64(*weekly) * models.InToMm)
			weather.RainWeekly = &mm
		}
	} else if v, ok := data["weeklyrainin"]; ok {
		if weekly := p.parseFloatPtr(v); weekly != nil {
			mm := float32(float64(*weekly) * models.InToMm)
			weather.RainWeekly = &mm
		}
	}

	if v, ok := data["mrain_piezo"]; ok {
		if monthly := p.parseFloatPtr(v); monthly != nil {
			mm := float32(float64(*monthly) * models.InToMm)
			weather.RainMonthly = &mm
		}
	} else if v, ok := data["monthlyrainin"]; ok {
		if monthly := p.parseFloatPtr(v); monthly != nil {
			mm := float32(float64(*monthly) * models.InToMm)
			weather.RainMonthly = &mm
		}
	}

	if v, ok := data["yrain_piezo"]; ok {
		if yearly := p.parseFloatPtr(v); yearly != nil {
			mm := float32(float64(*yearly) * models.InToMm)
			weather.RainYearly = &mm
		}
	} else if v, ok := data["yearlyrainin"]; ok {
		if yearly := p.parseFloatPtr(v); yearly != nil {
			mm := float32(float64(*yearly) * models.InToMm)
			weather.RainYearly = &mm
		}
	}

	// Солнце
	if v, ok := data["uv"]; ok {
		weather.UVIndex = p.parseFloatPtr(v)
	}
	if v, ok := data["solarradiation"]; ok {
		weather.SolarRadiation = p.parseFloatPtr(v)
	}

	// Вычисляем производные значения
	if weather.TempOutdoor != nil && weather.HumidityOutdoor != nil {
		tempC := float64(*weather.TempOutdoor)
		humidity := float64(*weather.HumidityOutdoor)

		// Точка росы
		dewPoint := float32(models.CalculateDewPoint(tempC, humidity))
		weather.DewPoint = &dewPoint

		// Ощущаемая температура
		windMs := 0.0
		if weather.WindSpeed != nil {
			windMs = float64(*weather.WindSpeed)
		}
		feelsLike := float32(models.CalculateFeelsLike(tempC, humidity, windMs))
		weather.TempFeelsLike = &feelsLike
	}

	// Сохраняем сырые данные (без технических полей)
	filteredData := make(map[string]string)
	for k, v := range data {
		if !excludeFields[k] {
			filteredData[k] = v
		}
	}
	rawJSON, _ := json.Marshal(filteredData)
	weather.RawData = rawJSON

	return weather, nil
}

// parsePayload пытается распарсить payload как URL-encoded или JSON
func (p *Parser) parsePayload(payload []byte) (map[string]string, error) {
	data := make(map[string]string)

	// Сначала пробуем как URL-encoded
	values, err := url.ParseQuery(string(payload))
	if err == nil && len(values) > 0 {
		for k, v := range values {
			if len(v) > 0 {
				data[k] = v[0]
			}
		}
		return data, nil
	}

	// Пробуем как JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(payload, &jsonData); err == nil {
		for k, v := range jsonData {
			data[k] = fmt.Sprintf("%v", v)
		}
		return data, nil
	}

	return nil, fmt.Errorf("unknown payload format")
}

func (p *Parser) parseFloatPtr(s string) *float32 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return nil
	}
	f := float32(v)
	return &f
}

func (p *Parser) parseInt16Ptr(s string) *int16 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 16)
	if err != nil {
		return nil
	}
	i := int16(v)
	return &i
}
