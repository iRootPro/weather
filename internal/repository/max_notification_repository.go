package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/iRootPro/weather/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type maxNotificationRepository struct{ pool *pgxpool.Pool }

func NewMaxNotificationRepository(pool *pgxpool.Pool) MaxNotificationRepository {
	return &maxNotificationRepository{pool: pool}
}

func (r *maxNotificationRepository) Create(ctx context.Context, notification *models.MaxNotification) error {
	query := `INSERT INTO max_notifications (user_id, event_type, event_data, sent_at) VALUES ($1, $2, $3, $4) RETURNING id`
	return r.pool.QueryRow(ctx, query, notification.UserID, notification.EventType, notification.EventData, notification.SentAt).Scan(&notification.ID)
}

func (r *maxNotificationRepository) WasRecentlySent(ctx context.Context, userID int64, eventType string, within time.Duration) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM max_notifications WHERE user_id = $1 AND event_type = $2 AND sent_at > $3)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, userID, eventType, time.Now().Add(-within)).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check recent max notification: %w", err)
	}
	return exists, nil
}
