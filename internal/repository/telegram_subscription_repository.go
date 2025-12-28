package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/iRootPro/weather/internal/models"
)

type telegramSubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewTelegramSubscriptionRepository(pool *pgxpool.Pool) TelegramSubscriptionRepository {
	return &telegramSubscriptionRepository{pool: pool}
}

func (r *telegramSubscriptionRepository) Create(ctx context.Context, sub *models.TelegramSubscription) error {
	query := `
		INSERT INTO telegram_subscriptions (user_id, event_type, is_active)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, event_type) DO UPDATE SET
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	return r.pool.QueryRow(ctx, query, sub.UserID, sub.EventType, sub.IsActive).
		Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
}

func (r *telegramSubscriptionRepository) GetByUserID(ctx context.Context, userID int64) ([]models.TelegramSubscription, error) {
	query := `
		SELECT id, user_id, event_type, is_active, created_at, updated_at
		FROM telegram_subscriptions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}
	defer rows.Close()

	var subscriptions []models.TelegramSubscription
	for rows.Next() {
		var sub models.TelegramSubscription
		err := rows.Scan(&sub.ID, &sub.UserID, &sub.EventType, &sub.IsActive, &sub.CreatedAt, &sub.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription: %w", err)
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

func (r *telegramSubscriptionRepository) GetActiveSubscribers(ctx context.Context, eventType string) ([]int64, error) {
	query := `
		SELECT DISTINCT tu.chat_id
		FROM telegram_subscriptions ts
		JOIN telegram_users tu ON ts.user_id = tu.id
		WHERE ts.event_type = $1 AND ts.is_active = true AND tu.is_active = true
	`

	rows, err := r.pool.Query(ctx, query, eventType)
	if err != nil {
		return nil, fmt.Errorf("failed to get active subscribers: %w", err)
	}
	defer rows.Close()

	var chatIDs []int64
	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			return nil, fmt.Errorf("failed to scan chat_id: %w", err)
		}
		chatIDs = append(chatIDs, chatID)
	}

	return chatIDs, nil
}

func (r *telegramSubscriptionRepository) Delete(ctx context.Context, userID int64, eventType string) error {
	query := `
		DELETE FROM telegram_subscriptions
		WHERE user_id = $1 AND event_type = $2
	`

	_, err := r.pool.Exec(ctx, query, userID, eventType)
	return err
}

func (r *telegramSubscriptionRepository) DeleteAll(ctx context.Context, userID int64) error {
	query := `
		DELETE FROM telegram_subscriptions
		WHERE user_id = $1
	`

	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

func (r *telegramSubscriptionRepository) Toggle(ctx context.Context, userID int64, eventType string, isActive bool) error {
	query := `
		UPDATE telegram_subscriptions
		SET is_active = $1, updated_at = NOW()
		WHERE user_id = $2 AND event_type = $3
	`

	_, err := r.pool.Exec(ctx, query, isActive, userID, eventType)
	return err
}
