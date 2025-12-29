package telegram

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

// ExifData содержит извлеченные EXIF метаданные
type ExifData struct {
	TakenAt     time.Time
	CameraMake  string
	CameraModel string
}

// ExtractExifData извлекает EXIF метаданные из фотографии
func ExtractExifData(photoData io.Reader) (*ExifData, error) {
	// Читаем данные в buffer для повторного использования
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, photoData)
	if err != nil {
		return nil, fmt.Errorf("failed to read photo data: %w", err)
	}

	// Декодируем EXIF
	x, err := exif.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("failed to decode exif: %w", err)
	}

	data := &ExifData{}

	// Извлекаем время съемки
	takenAt, err := x.DateTime()
	if err == nil {
		data.TakenAt = takenAt
	} else {
		// Если EXIF времени нет, используем текущее время
		data.TakenAt = time.Now()
	}

	// Извлекаем производителя камеры
	if make, err := x.Get(exif.Make); err == nil {
		if makeStr, err := make.StringVal(); err == nil {
			data.CameraMake = makeStr
		}
	}

	// Извлекаем модель камеры
	if model, err := x.Get(exif.Model); err == nil {
		if modelStr, err := model.StringVal(); err == nil {
			data.CameraModel = modelStr
		}
	}

	return data, nil
}
