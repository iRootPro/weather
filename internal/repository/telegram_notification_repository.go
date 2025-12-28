package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/iRootPro/weather/internal/models"
)

type telegramNotificationRepository struct {
	pool *pgxpool.Pool
}

func NewTelegramNotificationRepository(pool *pgxpool.Pool) TelegramNotificationRepository {
	return &telegramNotificationRepository{pool: pool}
}

func (r *telegramNotificationRepository) Create(ctx context.Context, notification *models.TelegramNotification) error {
	query := `
		INSERT INTO telegram_notifications (user_id, event_type, event_data, sent_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	return r.pool.QueryRow(ctx, query,
		notification.UserID,
		notification.EventType,
		notification.EventData,
		notification.SentAt,
	).Scan(&notification.ID)
}

func (r *telegramNotificationRepository) WasRecentlySent(ctx context.Context, userID int64, eventType string, within time.Duration) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM telegram_notifications
			WHERE user_id = $1
			  AND event_type = $2
			  AND sent_at > $3
		)
	`

	since := time.Now().Add(-within)
	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, eventType, since).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check recent notification: %w", err)
	}

	return exists, nil
}
