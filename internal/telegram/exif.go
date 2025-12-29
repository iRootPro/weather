package telegram

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ExifData содержит извлеченные EXIF метаданные
type ExifData struct {
	TakenAt     time.Time
	CameraMake  string
	CameraModel string
}

// ExtractExifDataFromFile извлекает EXIF метаданные из файла используя exiftool
func ExtractExifDataFromFile(filePath string) (*ExifData, error) {
	// Запускаем exiftool для извлечения метаданных в JSON формате
	cmd := exec.Command("exiftool", "-j", "-DateTimeOriginal", "-CreateDate", "-Make", "-Model", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run exiftool: %w", err)
	}

	// Парсим JSON вывод
	var results []map[string]interface{}
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("failed to parse exiftool output: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no exif data found")
	}

	result := results[0]
	data := &ExifData{
		TakenAt: time.Now(), // fallback
	}

	// Извлекаем время съемки - пробуем разные поля
	dateFormats := []string{
		"2006:01:02 15:04:05",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}

	// DateTimeOriginal - основное поле для времени съемки
	if dateStr, ok := result["DateTimeOriginal"].(string); ok && dateStr != "" {
		for _, format := range dateFormats {
			if t, err := time.Parse(format, dateStr); err == nil {
				data.TakenAt = t
				break
			}
		}
	} else if dateStr, ok := result["CreateDate"].(string); ok && dateStr != "" {
		// CreateDate - альтернативное поле
		for _, format := range dateFormats {
			if t, err := time.Parse(format, dateStr); err == nil {
				data.TakenAt = t
				break
			}
		}
	}

	// Извлекаем производителя камеры
	if make, ok := result["Make"].(string); ok {
		data.CameraMake = strings.TrimSpace(make)
	}

	// Извлекаем модель камеры
	if model, ok := result["Model"].(string); ok {
		data.CameraModel = strings.TrimSpace(model)
	}

	return data, nil
}
