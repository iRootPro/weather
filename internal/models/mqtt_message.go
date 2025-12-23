package models

// EcowittMessage представляет сырые данные от метеостанции EcoWitt.
// Формат Ecowitt Protocol передаёт данные в специфичном формате,
// который нужно преобразовать в WeatherData.
type EcowittMessage struct {
	// Идентификация станции
	StationType   string `json:"stationtype,omitempty"`
	PASSKEY       string `json:"PASSKEY,omitempty"`
	DateUTC       string `json:"dateutc,omitempty"`
	Freq          string `json:"freq,omitempty"`
	Model         string `json:"model,omitempty"`

	// Температура (°F в оригинале)
	TempInF       string `json:"tempinf,omitempty"`
	TempF         string `json:"tempf,omitempty"`

	// Влажность (%)
	HumidityIn    string `json:"humidityin,omitempty"`
	Humidity      string `json:"humidity,omitempty"`

	// Давление (inHg в оригинале)
	BaromRelIn    string `json:"baromrelin,omitempty"`
	BaromAbsIn    string `json:"baromabsin,omitempty"`

	// Ветер (mph в оригинале)
	WindDir       string `json:"winddir,omitempty"`
	WindSpeedMPH  string `json:"windspeedmph,omitempty"`
	WindGustMPH   string `json:"windgustmph,omitempty"`
	MaxDailyGust  string `json:"maxdailygust,omitempty"`

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
	UV            string `json:"uv,omitempty"`
	SolarRadiation string `json:"solarradiation,omitempty"`

	// Дополнительные датчики температуры
	Temp1F        string `json:"temp1f,omitempty"`
	Temp2F        string `json:"temp2f,omitempty"`
	Humidity1     string `json:"humidity1,omitempty"`
	Humidity2     string `json:"humidity2,omitempty"`

	// Почва
	SoilMoisture1 string `json:"soilmoisture1,omitempty"`
	SoilMoisture2 string `json:"soilmoisture2,omitempty"`

	// Батарея и статус
	WH65Batt      string `json:"wh65batt,omitempty"`
	WH25Batt      string `json:"wh25batt,omitempty"`
	Batt1         string `json:"batt1,omitempty"`
	Batt2         string `json:"batt2,omitempty"`
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
