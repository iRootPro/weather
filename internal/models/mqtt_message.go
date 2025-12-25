package models

import "math"

// EcowittMessage представляет сырые данные от метеостанции EcoWitt.
// Формат Ecowitt Protocol передаёт данные в специфичном формате,
// который нужно преобразовать в WeatherData.
type EcowittMessage struct {
	// Идентификация станции
	StationType string `json:"stationtype,omitempty"`
	PASSKEY     string `json:"PASSKEY,omitempty"`
	DateUTC     string `json:"dateutc,omitempty"`
	Freq        string `json:"freq,omitempty"`
	Model       string `json:"model,omitempty"`

	// Температура (°F в оригинале)
	TempInF string `json:"tempinf,omitempty"`
	TempF   string `json:"tempf,omitempty"`

	// Влажность (%)
	HumidityIn string `json:"humidityin,omitempty"`
	Humidity   string `json:"humidity,omitempty"`

	// Давление (inHg в оригинале)
	BaromRelIn string `json:"baromrelin,omitempty"`
	BaromAbsIn string `json:"baromabsin,omitempty"`

	// Ветер (mph в оригинале)
	WindDir      string `json:"winddir,omitempty"`
	WindSpeedMPH string `json:"windspeedmph,omitempty"`
	WindGustMPH  string `json:"windgustmph,omitempty"`
	MaxDailyGust string `json:"maxdailygust,omitempty"`

	// Осадки (in в оригинале)
	RainRateIn    string `json:"rainratein,omitempty"`
	EventRainIn   string `json:"eventrainin,omitempty"`
	HourlyRainIn  string `json:"hourlyrainin,omitempty"`
	DailyRainIn   string `json:"dailyrainin,omitempty"`
	WeeklyRainIn  string `json:"weeklyrainin,omitempty"`
	MonthlyRainIn string `json:"monthlyrainin,omitempty"`
	YearlyRainIn  string `json:"yearlyrainin,omitempty"`
	TotalRainIn   string `json:"totalrainin,omitempty"`

	// Солнце
	UV             string `json:"uv,omitempty"`
	SolarRadiation string `json:"solarradiation,omitempty"`

	// Дополнительные датчики температуры
	Temp1F    string `json:"temp1f,omitempty"`
	Temp2F    string `json:"temp2f,omitempty"`
	Humidity1 string `json:"humidity1,omitempty"`
	Humidity2 string `json:"humidity2,omitempty"`

	// Почва
	SoilMoisture1 string `json:"soilmoisture1,omitempty"`
	SoilMoisture2 string `json:"soilmoisture2,omitempty"`

	// Батарея и статус
	WH65Batt string `json:"wh65batt,omitempty"`
	WH25Batt string `json:"wh25batt,omitempty"`
	Batt1    string `json:"batt1,omitempty"`
	Batt2    string `json:"batt2,omitempty"`
}

// Коэффициенты для конвертации единиц измерения
const (
	// Давление: inHg -> мм рт. ст.
	InHgToMmHg = 25.4

	// Скорость: mph -> м/с
	MphToMs = 0.44704

	// Осадки: in -> мм
	InToMm = 25.4
)

// FahrenheitToCelsius конвертирует температуру из °F в °C
func FahrenheitToCelsius(f float64) float64 {
	return (f - 32) * 5 / 9
}

// CalculateDewPoint вычисляет точку росы по формуле Магнуса
// tempC - температура в °C, humidity - относительная влажность в %
func CalculateDewPoint(tempC float64, humidity float64) float64 {
	if humidity <= 0 || humidity > 100 {
		return tempC
	}

	const a = 17.27
	const b = 237.7

	alpha := (a*tempC)/(b+tempC) + math.Log(humidity/100.0)
	return (b * alpha) / (a - alpha)
}

// CalculateFeelsLike вычисляет "ощущаемую" температуру
// Учитывает Wind Chill при холоде и Heat Index при жаре
// tempC - температура в °C, humidity - влажность %, windMs - скорость ветра м/с
func CalculateFeelsLike(tempC float64, humidity float64, windMs float64) float64 {
	// Wind Chill: при T < 10°C и ветре > 1.3 м/с
	if tempC <= 10 && windMs > 1.3 {
		// Конвертируем м/с в км/ч
		windKmh := windMs * 3.6
		// Формула Wind Chill (канадская)
		wc := 13.12 + 0.6215*tempC - 11.37*math.Pow(windKmh, 0.16) + 0.3965*tempC*math.Pow(windKmh, 0.16)
		return wc
	}

	// Heat Index: при T > 27°C и влажности > 40%
	if tempC >= 27 && humidity >= 40 {
		// Упрощённая формула Heat Index (Steadman)
		hi := -8.784695 +
			1.61139411*tempC +
			2.338549*humidity -
			0.14611605*tempC*humidity -
			0.012308094*tempC*tempC -
			0.016424828*humidity*humidity +
			0.002211732*tempC*tempC*humidity +
			0.00072546*tempC*humidity*humidity -
			0.000003582*tempC*tempC*humidity*humidity
		return hi
	}

	// В остальных случаях возвращаем реальную температуру
	return tempC
}

// IsFoggy определяет, есть ли туман
// Туман возникает когда разница между температурой и точкой росы < 1°C
func IsFoggy(tempC, dewPoint float64) bool {
	return (tempC - dewPoint) < 1
}
