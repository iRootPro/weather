package telegram

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
	"os"

	"github.com/fogleman/gg"
	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/repository"
)

// GenerateWeatherImage создает красивую карточку с погодой
func GenerateWeatherImage(current *models.WeatherData, hourAgo *models.WeatherData, dailyMinMax *repository.DailyMinMax) ([]byte, error) {
	if current == nil {
		return nil, fmt.Errorf("no weather data")
	}

	// Размер изображения
	const width = 800
	const height = 600

	// Создаем контекст
	dc := gg.NewContext(width, height)

	// Определяем цвет фона в зависимости от температуры
	bgColor := getBackgroundColor(current.TempOutdoor)

	// Рисуем градиентный фон
	drawGradientBackground(dc, width, height, bgColor)

	// Белый цвет для текста
	dc.SetColor(color.White)

	// Заголовок - город и дата
	months := []string{"", "января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря"}
	day := current.Time.Day()
	month := months[current.Time.Month()]

	if err := dc.LoadFontFace(findFont(true), 32); err != nil {
		return nil, fmt.Errorf("failed to load bold font: %w", err)
	}
	dc.DrawStringAnchored("АРМАВИР", width/2, 50, 0.5, 0.5)

	if err := dc.LoadFontFace(findFont(false), 20); err != nil {
		return nil, fmt.Errorf("failed to load regular font: %w", err)
	}
	dc.DrawStringAnchored(fmt.Sprintf("%d %s", day, month), width/2, 85, 0.5, 0.5)

	// Основная температура - крупно в центре
	if current.TempOutdoor != nil {
		dc.LoadFontFace(findFont(true), 120)
		tempText := fmt.Sprintf("%.1f°", *current.TempOutdoor)
		dc.DrawStringAnchored(tempText, width/2, 220, 0.5, 0.5)

		// Изменение за час
		if hourAgo != nil && hourAgo.TempOutdoor != nil {
			change := *current.TempOutdoor - *hourAgo.TempOutdoor
			dc.LoadFontFace(findFont(false), 24)
			var changeText string
			if change > 0 {
				changeText = fmt.Sprintf("↗ +%.1f° за час", change)
			} else if change < 0 {
				changeText = fmt.Sprintf("↘ %.1f° за час", change)
			} else {
				changeText = "без изменений"
			}
			dc.DrawStringAnchored(changeText, width/2, 290, 0.5, 0.5)
		}
	}

	// Ощущается как
	if current.TempFeelsLike != nil {
		dc.LoadFontFace(findFont(false), 22)
		feelsText := fmt.Sprintf("Ощущается как %.1f°", *current.TempFeelsLike)
		dc.DrawStringAnchored(feelsText, width/2, 325, 0.5, 0.5)
	}

	// Линия разделитель
	dc.SetLineWidth(2)
	dc.DrawLine(100, 360, width-100, 360)
	dc.Stroke()

	// Остальные данные - сетка 2x3
	dc.LoadFontFace(findFont(false), 18)

	y := 400.0
	leftX := 180.0
	rightX := float64(width) - 180.0

	// Левая колонка
	// Мин/Макс
	if dailyMinMax != nil && dailyMinMax.TempMin != nil && dailyMinMax.TempMax != nil {
		text := fmt.Sprintf("📊 %.1f° ... %.1f°", *dailyMinMax.TempMin, *dailyMinMax.TempMax)
		dc.DrawStringAnchored(text, leftX, y, 0.5, 0.5)
	}
	y += 40

	// Влажность
	if current.HumidityOutdoor != nil {
		text := fmt.Sprintf("💧 Влажность: %d%%", *current.HumidityOutdoor)
		dc.DrawStringAnchored(text, leftX, y, 0.5, 0.5)
	}
	y += 40

	// Давление
	if current.PressureRelative != nil {
		text := fmt.Sprintf("🔽 Давление: %.0f мм", *current.PressureRelative)
		dc.DrawStringAnchored(text, leftX, y, 0.5, 0.5)
	}

	// Правая колонка
	y = 400

	// Ветер
	if current.WindSpeed != nil {
		windText := fmt.Sprintf("💨 Ветер: %.1f м/с", *current.WindSpeed)
		if current.WindDirection != nil {
			direction := getWindDirectionShort(*current.WindDirection)
			windText += fmt.Sprintf(" %s", direction)
		}
		dc.DrawStringAnchored(windText, rightX, y, 0.5, 0.5)
	}
	y += 40

	// Точка росы
	if current.DewPoint != nil {
		text := fmt.Sprintf("💦 Роса: %.1f°", *current.DewPoint)
		dc.DrawStringAnchored(text, rightX, y, 0.5, 0.5)
	}
	y += 40

	// UV индекс
	if current.UVIndex != nil {
		text := fmt.Sprintf("☀️ UV: %.1f", *current.UVIndex)
		dc.DrawStringAnchored(text, rightX, y, 0.5, 0.5)
	}

	// Время обновления внизу
	dc.LoadFontFace(findFont(false), 16)
	updateText := fmt.Sprintf("🕐 Обновлено: %s", current.Time.Format("15:04"))
	dc.DrawStringAnchored(updateText, width/2, height-30, 0.5, 0.5)

	// Конвертируем в PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}

// getBackgroundColor возвращает цвет фона в зависимости от температуры
func getBackgroundColor(temp *float32) color.RGBA {
	if temp == nil {
		return color.RGBA{R: 70, G: 130, B: 180, A: 255} // steel blue
	}

	t := *temp
	switch {
	case t < -20:
		return color.RGBA{R: 25, G: 25, B: 112, A: 255} // midnight blue
	case t < -10:
		return color.RGBA{R: 65, G: 105, B: 225, A: 255} // royal blue
	case t < 0:
		return color.RGBA{R: 70, G: 130, B: 180, A: 255} // steel blue
	case t < 10:
		return color.RGBA{R: 100, G: 149, B: 237, A: 255} // cornflower blue
	case t < 20:
		return color.RGBA{R: 60, G: 179, B: 113, A: 255} // medium sea green
	case t < 30:
		return color.RGBA{R: 255, G: 165, B: 0, A: 255} // orange
	default:
		return color.RGBA{R: 220, G: 20, B: 60, A: 255} // crimson
	}
}

// drawGradientBackground рисует градиентный фон
func drawGradientBackground(dc *gg.Context, width, height int, baseColor color.RGBA) {
	// Верхний цвет - светлее
	topColor := color.RGBA{
		R: uint8(min(int(baseColor.R)+30, 255)),
		G: uint8(min(int(baseColor.G)+30, 255)),
		B: uint8(min(int(baseColor.B)+30, 255)),
		A: 255,
	}

	// Нижний цвет - темнее
	bottomColor := color.RGBA{
		R: uint8(max(int(baseColor.R)-30, 0)),
		G: uint8(max(int(baseColor.G)-30, 0)),
		B: uint8(max(int(baseColor.B)-30, 0)),
		A: 255,
	}

	// Рисуем градиент
	for y := 0; y < height; y++ {
		t := float64(y) / float64(height)
		r := uint8(float64(topColor.R)*(1-t) + float64(bottomColor.R)*t)
		g := uint8(float64(topColor.G)*(1-t) + float64(bottomColor.G)*t)
		b := uint8(float64(topColor.B)*(1-t) + float64(bottomColor.B)*t)

		dc.SetColor(color.RGBA{R: r, G: g, B: b, A: 255})
		dc.DrawRectangle(0, float64(y), float64(width), 1)
		dc.Fill()
	}
}

// getWindDirectionShort возвращает короткое направление ветра
func getWindDirectionShort(degrees int16) string {
	directions := []string{"С", "СВ", "В", "ЮВ", "Ю", "ЮЗ", "З", "СЗ"}
	index := int((float64(degrees) + 22.5) / 45.0)
	return directions[index%8]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// findFont ищет подходящий шрифт в системе
func findFont(bold bool) string {
	// Список путей для поиска шрифтов
	var paths []string

	if bold {
		paths = []string{
			"/usr/share/fonts/dejavu/DejaVuSans-Bold.ttf",                 // Alpine Linux (ttf-dejavu)
			"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",        // Debian/Ubuntu
			"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf", // Linux (Liberation)
			"/usr/share/fonts/TTF/DejaVuSans-Bold.ttf",                     // Arch Linux
			"/System/Library/Fonts/Supplemental/Arial Bold.ttf",           // macOS
		}
	} else {
		paths = []string{
			"/usr/share/fonts/dejavu/DejaVuSans.ttf",                      // Alpine Linux (ttf-dejavu)
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",            // Debian/Ubuntu
			"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf", // Linux (Liberation)
			"/usr/share/fonts/TTF/DejaVuSans.ttf",                         // Arch Linux
			"/System/Library/Fonts/Supplemental/Arial.ttf",               // macOS
		}
	}

	// Ищем первый существующий файл
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Если ничего не нашли, возвращаем путь для Alpine (где точно должны быть после apk add ttf-dejavu)
	if bold {
		return "/usr/share/fonts/dejavu/DejaVuSans-Bold.ttf"
	}
	return "/usr/share/fonts/dejavu/DejaVuSans.ttf"
}
