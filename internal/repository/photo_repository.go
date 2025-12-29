package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/iRootPro/weather/internal/models"
)

type photoRepository struct {
	pool *pgxpool.Pool
}

func NewPhotoRepository(pool *pgxpool.Pool) PhotoRepository {
	return &photoRepository{pool: pool}
}

func (r *photoRepository) Create(ctx context.Context, photo *models.Photo) error {
	query := `
		INSERT INTO photos (
			filename, file_path, caption, taken_at,
			temperature, humidity, pressure, wind_speed, wind_direction,
			rain_rate, solar_radiation, weather_description,
			camera_make, camera_model,
			telegram_file_id, telegram_user_id,
			is_visible
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11, $12,
			$13, $14,
			$15, $16,
			$17
		)
		RETURNING id, uploaded_at, created_at, updated_at
	`

	return r.pool.QueryRow(ctx, query,
		photo.Filename, photo.FilePath, photo.Caption, photo.TakenAt,
		photo.Temperature, photo.Humidity, photo.Pressure, photo.WindSpeed, photo.WindDirection,
		photo.RainRate, photo.SolarRadiation, photo.WeatherDescription,
		photo.CameraMake, photo.CameraModel,
		photo.TelegramFileID, photo.TelegramUserID,
		photo.IsVisible,
	).Scan(&photo.ID, &photo.UploadedAt, &photo.CreatedAt, &photo.UpdatedAt)
}

func (r *photoRepository) GetByID(ctx context.Context, id int64) (*models.Photo, error) {
	query := `
		SELECT id, filename, file_path, caption, taken_at, uploaded_at,
		       temperature, humidity, pressure, wind_speed, wind_direction,
		       rain_rate, solar_radiation, weather_description,
		       camera_make, camera_model,
		       telegram_file_id, telegram_user_id,
		       is_visible, created_at, updated_at
		FROM photos
		WHERE id = $1
	`

	var photo models.Photo
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&photo.ID, &photo.Filename, &photo.FilePath, &photo.Caption, &photo.TakenAt, &photo.UploadedAt,
		&photo.Temperature, &photo.Humidity, &photo.Pressure, &photo.WindSpeed, &photo.WindDirection,
		&photo.RainRate, &photo.SolarRadiation, &photo.WeatherDescription,
		&photo.CameraMake, &photo.CameraModel,
		&photo.TelegramFileID, &photo.TelegramUserID,
		&photo.IsVisible, &photo.CreatedAt, &photo.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get photo by id: %w", err)
	}

	return &photo, nil
}

func (r *photoRepository) GetAll(ctx context.Context, limit, offset int) ([]models.Photo, error) {
	query := `
		SELECT id, filename, file_path, caption, taken_at, uploaded_at,
		       temperature, humidity, pressure, wind_speed, wind_direction,
		       rain_rate, solar_radiation, weather_description,
		       camera_make, camera_model,
		       telegram_file_id, telegram_user_id,
		       is_visible, created_at, updated_at
		FROM photos
		ORDER BY taken_at DESC
		LIMIT $1 OFFSET $2
	`

	return r.queryPhotos(ctx, query, limit, offset)
}

func (r *photoRepository) GetVisible(ctx context.Context, limit, offset int) ([]models.Photo, error) {
	query := `
		SELECT id, filename, file_path, caption, taken_at, uploaded_at,
		       temperature, humidity, pressure, wind_speed, wind_direction,
		       rain_rate, solar_radiation, weather_description,
		       camera_make, camera_model,
		       telegram_file_id, telegram_user_id,
		       is_visible, created_at, updated_at
		FROM photos
		WHERE is_visible = true
		ORDER BY taken_at DESC
		LIMIT $1 OFFSET $2
	`

	return r.queryPhotos(ctx, query, limit, offset)
}

func (r *photoRepository) GetByUserID(ctx context.Context, userID int64) ([]models.Photo, error) {
	query := `
		SELECT id, filename, file_path, caption, taken_at, uploaded_at,
		       temperature, humidity, pressure, wind_speed, wind_direction,
		       rain_rate, solar_radiation, weather_description,
		       camera_make, camera_model,
		       telegram_file_id, telegram_user_id,
		       is_visible, created_at, updated_at
		FROM photos
		WHERE telegram_user_id = $1
		ORDER BY taken_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get photos by user_id: %w", err)
	}
	defer rows.Close()

	return r.scanPhotos(rows)
}

func (r *photoRepository) UpdateVisibility(ctx context.Context, id int64, isVisible bool) error {
	query := `
		UPDATE photos
		SET is_visible = $1
		WHERE id = $2
	`

	_, err := r.pool.Exec(ctx, query, isVisible, id)
	if err != nil {
		return fmt.Errorf("failed to update photo visibility: %w", err)
	}

	return nil
}

func (r *photoRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM photos WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete photo: %w", err)
	}

	return nil
}

func (r *photoRepository) GetWeatherForTime(ctx context.Context, takenAt time.Time) (*models.WeatherData, error) {
	query := `
		SELECT time, temp_outdoor, humidity_outdoor, pressure_relative,
		       wind_speed, wind_direction, wind_gust, rain_rate,
		       solar_radiation, uv_index, temp_feels_like
		FROM weather_data
		WHERE time >= $1 - INTERVAL '5 minutes'
		  AND time <= $1 + INTERVAL '5 minutes'
		ORDER BY ABS(EXTRACT(EPOCH FROM (time - $1)))
		LIMIT 1
	`

	var data models.WeatherData
	err := r.pool.QueryRow(ctx, query, takenAt).Scan(
		&data.Time, &data.TempOutdoor, &data.HumidityOutdoor, &data.PressureRelative,
		&data.WindSpeed, &data.WindDirection, &data.WindGust, &data.RainRate,
		&data.SolarRadiation, &data.UVIndex, &data.TempFeelsLike,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather for time: %w", err)
	}

	return &data, nil
}

// Вспомогательные методы

func (r *photoRepository) queryPhotos(ctx context.Context, query string, args ...interface{}) ([]models.Photo, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query photos: %w", err)
	}
	defer rows.Close()

	return r.scanPhotos(rows)
}

func (r *photoRepository) scanPhotos(rows interface{
	Next() bool
	Scan(dest ...interface{}) error
}) ([]models.Photo, error) {
	var photos []models.Photo
	for rows.Next() {
		var photo models.Photo
		err := rows.Scan(
			&photo.ID, &photo.Filename, &photo.FilePath, &photo.Caption, &photo.TakenAt, &photo.UploadedAt,
			&photo.Temperature, &photo.Humidity, &photo.Pressure, &photo.WindSpeed, &photo.WindDirection,
			&photo.RainRate, &photo.SolarRadiation, &photo.WeatherDescription,
			&photo.CameraMake, &photo.CameraModel,
			&photo.TelegramFileID, &photo.TelegramUserID,
			&photo.IsVisible, &photo.CreatedAt, &photo.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan photo: %w", err)
		}
		photos = append(photos, photo)
	}

	return photos, nil
}
