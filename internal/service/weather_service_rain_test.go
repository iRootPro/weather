package service

import (
	"testing"
	"time"

	"github.com/iRootPro/weather/internal/models"
)

// TestDetectRainEvents_ShortRainIgnored проверяет, что короткий дождь (<15 минут) игнорируется
func TestDetectRainEvents_ShortRainIgnored(t *testing.T) {
	// Arrange: дождь длится 10 минут (3 точки по 5 минут = 10 минут)
	baseTime := time.Now()
	rainRate := float32(0.5) // мм/ч

	data := []models.WeatherData{
		{Time: baseTime, RainRate: &rainRate},
		{Time: baseTime.Add(5 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(10 * time.Minute), RainRate: nil}, // дождь закончился
	}

	// Act
	events := detectRainEvents(data)

	// Assert
	if len(events) != 0 {
		t.Errorf("Ожидалось 0 событий (дождь слишком короткий), получено %d", len(events))
	}
}

// TestDetectRainEvents_LongRainCreatesEvents проверяет, что длинный дождь создаёт события
func TestDetectRainEvents_LongRainCreatesEvents(t *testing.T) {
	// Arrange: дождь длится 30 минут (7 точек по 5 минут)
	baseTime := time.Now()
	rainRate := float32(0.5)
	noRain := float32(0.0)

	data := []models.WeatherData{
		{Time: baseTime, RainRate: &rainRate},
		{Time: baseTime.Add(5 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(10 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(15 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(20 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(25 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(30 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(35 * time.Minute), RainRate: &noRain}, // дождь закончился
	}

	// Act
	events := detectRainEvents(data)

	// Assert
	if len(events) != 1 {
		t.Errorf("Ожидалось 1 событие (rain_end), получено %d", len(events))
	}

	if len(events) > 0 {
		if events[0].Type != "rain_end" {
			t.Errorf("Ожидался тип 'rain_end', получен '%s'", events[0].Type)
		}
		if events[0].Icon != "☁️" {
			t.Errorf("Ожидалась иконка '☁️', получена '%s'", events[0].Icon)
		}

		// Проверяем длительность дождя (должна быть около 30 минут)
		durationHours := events[0].Change
		expectedHours := 30.0 / 60.0 // 0.5 часа
		if durationHours < expectedHours-0.01 || durationHours > expectedHours+0.01 {
			t.Errorf("Ожидалась длительность ~%.2f часа, получено %.2f", expectedHours, durationHours)
		}
	}
}

// TestDetectRainEvents_ShortPauseMerged проверяет объединение дождей с короткой паузой
func TestDetectRainEvents_ShortPauseMerged(t *testing.T) {
	// Arrange: дождь 20м → пауза 10м → дождь 20м (должно объединиться в один дождь ~50м)
	baseTime := time.Now()
	rainRate := float32(0.5)
	noRain := float32(0.0)

	data := []models.WeatherData{
		// Первый дождь: 20 минут (5 точек)
		{Time: baseTime, RainRate: &rainRate},
		{Time: baseTime.Add(5 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(10 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(15 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(20 * time.Minute), RainRate: &rainRate},

		// Пауза: 10 минут (2 точки)
		{Time: baseTime.Add(25 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(30 * time.Minute), RainRate: &noRain},

		// Второй дождь: 20 минут (5 точек)
		{Time: baseTime.Add(35 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(40 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(45 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(50 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(55 * time.Minute), RainRate: &rainRate},

		// Конец
		{Time: baseTime.Add(60 * time.Minute), RainRate: &noRain},
	}

	// Act
	events := detectRainEvents(data)

	// Assert
	if len(events) != 1 {
		t.Errorf("Ожидалось 1 событие (объединенный дождь), получено %d", len(events))
		for i, e := range events {
			t.Logf("Событие %d: Type=%s, Time=%s, Duration=%.2fh", i, e.Type, e.Time.Format("15:04"), e.Change)
		}
	}

	if len(events) > 0 {
		// Проверяем, что это событие конца дождя
		if events[0].Type != "rain_end" {
			t.Errorf("Ожидался тип 'rain_end', получен '%s'", events[0].Type)
		}

		// Длительность должна быть ~55 минут (20 + 10 + 20 + 5)
		durationHours := events[0].Change
		expectedHours := 55.0 / 60.0
		// Допускаем погрешность ±5 минут
		tolerance := 5.0 / 60.0
		if durationHours < expectedHours-tolerance || durationHours > expectedHours+tolerance {
			t.Errorf("Ожидалась длительность ~%.2f часа, получено %.2f", expectedHours, durationHours)
		}
	}
}

// TestDetectRainEvents_LongPauseSeparates проверяет разделение дождей с длинной паузой
func TestDetectRainEvents_LongPauseSeparates(t *testing.T) {
	// Arrange: дождь 30м → пауза 60м → дождь 30м (должно быть 2 отдельных дождя)
	baseTime := time.Now()
	rainRate := float32(0.5)
	noRain := float32(0.0)

	data := []models.WeatherData{
		// Первый дождь: 30 минут (7 точек)
		{Time: baseTime, RainRate: &rainRate},
		{Time: baseTime.Add(5 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(10 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(15 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(20 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(25 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(30 * time.Minute), RainRate: &rainRate},

		// Пауза: 60 минут (12 точек - пропускаем для краткости)
		{Time: baseTime.Add(35 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(40 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(45 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(50 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(55 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(60 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(65 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(70 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(75 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(80 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(85 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(90 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(95 * time.Minute), RainRate: &noRain},

		// Второй дождь: 30 минут (7 точек)
		{Time: baseTime.Add(100 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(105 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(110 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(115 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(120 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(125 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(130 * time.Minute), RainRate: &rainRate},

		// Конец
		{Time: baseTime.Add(135 * time.Minute), RainRate: &noRain},
	}

	// Act
	events := detectRainEvents(data)

	// Assert
	if len(events) != 2 {
		t.Errorf("Ожидалось 2 события (два отдельных дождя), получено %d", len(events))
		for i, e := range events {
			t.Logf("Событие %d: Type=%s, Time=%s, Duration=%.2fh", i, e.Type, e.Time.Format("15:04"), e.Change)
		}
	}

	// Оба события должны быть типа rain_end
	for i, event := range events {
		if event.Type != "rain_end" {
			t.Errorf("Событие %d: ожидался тип 'rain_end', получен '%s'", i, event.Type)
		}
	}
}

// TestDetectRainEvents_EdgeCases проверяет граничные случаи
func TestDetectRainEvents_EdgeCases(t *testing.T) {
	t.Run("Пауза меньше 30 минут - должна объединить", func(t *testing.T) {
		baseTime := time.Now()
		rainRate := float32(0.5)
		noRain := float32(0.0)

		data := []models.WeatherData{
			// Дождь 20 минут
			{Time: baseTime, RainRate: &rainRate},
			{Time: baseTime.Add(5 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(10 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(15 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(20 * time.Minute), RainRate: &rainRate},

			// Пауза 25 минут (20 до 45)
			{Time: baseTime.Add(25 * time.Minute), RainRate: &noRain},
			{Time: baseTime.Add(30 * time.Minute), RainRate: &noRain},
			{Time: baseTime.Add(35 * time.Minute), RainRate: &noRain},
			{Time: baseTime.Add(40 * time.Minute), RainRate: &noRain},

			// Дождь 20 минут
			{Time: baseTime.Add(45 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(50 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(55 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(60 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(65 * time.Minute), RainRate: &rainRate},

			{Time: baseTime.Add(70 * time.Minute), RainRate: &noRain},
		}

		events := detectRainEvents(data)

		// Пауза 25 минут < 30 минут, должно объединиться
		if len(events) != 1 {
			t.Errorf("Ожидалось 1 событие (объединенный дождь), получено %d", len(events))
		}
	})

	t.Run("Дождь ровно 15 минут - должен создать событие", func(t *testing.T) {
		baseTime := time.Now()
		rainRate := float32(0.5)
		noRain := float32(0.0)

		data := []models.WeatherData{
			{Time: baseTime, RainRate: &rainRate},
			{Time: baseTime.Add(5 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(10 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(15 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(20 * time.Minute), RainRate: &noRain},
		}

		events := detectRainEvents(data)

		// Дождь >= 15 минут, должен создать событие
		if len(events) != 1 {
			t.Errorf("Ожидалось 1 событие, получено %d", len(events))
		}
	})

	t.Run("Дождь всё ещё идёт", func(t *testing.T) {
		baseTime := time.Now()
		rainRate := float32(0.5)

		data := []models.WeatherData{
			{Time: baseTime, RainRate: &rainRate},
			{Time: baseTime.Add(5 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(10 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(15 * time.Minute), RainRate: &rainRate},
			{Time: baseTime.Add(20 * time.Minute), RainRate: &rainRate},
			// Дождь не закончился
		}

		events := detectRainEvents(data)

		if len(events) != 1 {
			t.Errorf("Ожидалось 1 событие (rain_start), получено %d", len(events))
		}

		if len(events) > 0 && events[0].Type != "rain_start" {
			t.Errorf("Ожидался тип 'rain_start', получен '%s'", events[0].Type)
		}
	})

	t.Run("Пустой массив данных", func(t *testing.T) {
		data := []models.WeatherData{}

		events := detectRainEvents(data)

		if len(events) != 0 {
			t.Errorf("Ожидалось 0 событий для пустого массива, получено %d", len(events))
		}
	})

	t.Run("Нет дождя вообще", func(t *testing.T) {
		baseTime := time.Now()
		noRain := float32(0.0)

		data := []models.WeatherData{
			{Time: baseTime, RainRate: &noRain},
			{Time: baseTime.Add(5 * time.Minute), RainRate: &noRain},
			{Time: baseTime.Add(10 * time.Minute), RainRate: &noRain},
		}

		events := detectRainEvents(data)

		if len(events) != 0 {
			t.Errorf("Ожидалось 0 событий (нет дождя), получено %d", len(events))
		}
	})
}

// TestFindRainPeriods тестирует функцию поиска периодов дождя
func TestFindRainPeriods(t *testing.T) {
	baseTime := time.Now()
	rainRate := float32(0.5)
	noRain := float32(0.0)

	data := []models.WeatherData{
		{Time: baseTime, RainRate: &rainRate},
		{Time: baseTime.Add(5 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(10 * time.Minute), RainRate: &noRain},
		{Time: baseTime.Add(15 * time.Minute), RainRate: &rainRate},
		{Time: baseTime.Add(20 * time.Minute), RainRate: &rainRate},
	}

	periods := findRainPeriods(data)

	if len(periods) != 2 {
		t.Errorf("Ожидалось 2 периода, получено %d", len(periods))
	}
}

// TestMergeRainPeriodsWithShortPauses тестирует объединение периодов
func TestMergeRainPeriodsWithShortPauses(t *testing.T) {
	baseTime := time.Now()

	periods := []rainPeriod{
		{start: baseTime, end: baseTime.Add(20 * time.Minute)},
		{start: baseTime.Add(25 * time.Minute), end: baseTime.Add(45 * time.Minute)}, // пауза 5 минут
		{start: baseTime.Add(80 * time.Minute), end: baseTime.Add(100 * time.Minute)}, // пауза 35 минут
	}

	merged := mergeRainPeriodsWithShortPauses(periods, 30)

	// Первые два периода должны объединиться (пауза 5 минут < 30)
	// Третий период остаётся отдельным (пауза 35 минут > 30)
	if len(merged) != 2 {
		t.Errorf("Ожидалось 2 объединенных периода, получено %d", len(merged))
	}

	if len(merged) >= 1 {
		expectedEnd := baseTime.Add(45 * time.Minute)
		if !merged[0].end.Equal(expectedEnd) {
			t.Errorf("Ожидалось окончание первого периода в %s, получено %s",
				expectedEnd.Format("15:04"), merged[0].end.Format("15:04"))
		}
	}
}

// TestFilterShortRains тестирует фильтрацию коротких дождей
func TestFilterShortRains(t *testing.T) {
	baseTime := time.Now()

	periods := []rainPeriod{
		{start: baseTime, end: baseTime.Add(10 * time.Minute)},  // 10 минут - короткий
		{start: baseTime, end: baseTime.Add(15 * time.Minute)},  // 15 минут - граница
		{start: baseTime, end: baseTime.Add(30 * time.Minute)},  // 30 минут - длинный
	}

	filtered := filterShortRains(periods, 15)

	// Должны остаться только два периода (>= 15 минут)
	if len(filtered) != 2 {
		t.Errorf("Ожидалось 2 периода (>= 15 минут), получено %d", len(filtered))
	}
}
