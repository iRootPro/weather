package repository

import (
	"context"
	"fmt"

	"github.com/iRootPro/weather/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type maxSubscriptionRepository struct{ pool *pgxpool.Pool }

func NewMaxSubscriptionRepository(pool *pgxpool.Pool) MaxSubscriptionRepository {
	return &maxSubscriptionRepository{pool: pool}
}

func (r *maxSubscriptionRepository) Create(ctx context.Context, sub *models.MaxSubscription) error {
	query := `
		INSERT INTO max_subscriptions (user_id, event_type, is_active)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, event_type) DO UPDATE SET
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query, sub.UserID, sub.EventType, sub.IsActive).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
}

func (r *maxSubscriptionRepository) GetByUserID(ctx context.Context, userID int64) ([]models.MaxSubscription, error) {
	query := `SELECT id, user_id, event_type, is_active, created_at, updated_at FROM max_subscriptions WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get max subscriptions: %w", err)
	}
	defer rows.Close()
	subs := []models.MaxSubscription{}
	for rows.Next() {
		var sub models.MaxSubscription
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.EventType, &sub.IsActive, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan max subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

func (r *maxSubscriptionRepository) GetActiveSubscribers(ctx context.Context, eventType string) ([]int64, error) {
	query := `
		SELECT DISTINCT mu.user_id
		FROM max_subscriptions ms
		JOIN max_users mu ON ms.user_id = mu.id
		WHERE ms.event_type = $1 AND ms.is_active = true AND mu.is_active = true
	`
	rows, err := r.pool.Query(ctx, query, eventType)
	if err != nil {
		return nil, fmt.Errorf("failed to get active max subscribers: %w", err)
	}
	defer rows.Close()
	var userIDs []int64
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan max user_id: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	return userIDs, nil
}

func (r *maxSubscriptionRepository) Delete(ctx context.Context, userID int64, eventType string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM max_subscriptions WHERE user_id = $1 AND event_type = $2`, userID, eventType)
	return err
}

func (r *maxSubscriptionRepository) DeleteAll(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM max_subscriptions WHERE user_id = $1`, userID)
	return err
}
