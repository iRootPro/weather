package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/iRootPro/weather/internal/models"
)

type telegramUserRepository struct {
	pool *pgxpool.Pool
}

func NewTelegramUserRepository(pool *pgxpool.Pool) TelegramUserRepository {
	return &telegramUserRepository{pool: pool}
}

func (r *telegramUserRepository) Create(ctx context.Context, user *models.TelegramUser) error {
	query := `
		INSERT INTO telegram_users (chat_id, username, first_name, last_name, language_code, is_bot)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (chat_id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			language_code = EXCLUDED.language_code,
			updated_at = NOW()
		RETURNING id, created_at, updated_at, is_active
	`

	return r.pool.QueryRow(ctx, query,
		user.ChatID, user.Username, user.FirstName, user.LastName,
		user.LanguageCode, user.IsBot,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.IsActive)
}

func (r *telegramUserRepository) GetByID(ctx context.Context, id int64) (*models.TelegramUser, error) {
	query := `
		SELECT id, chat_id, username, first_name, last_name, language_code,
		       is_bot, is_active, created_at, updated_at
		FROM telegram_users
		WHERE id = $1
	`

	var user models.TelegramUser
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.ChatID, &user.Username, &user.FirstName, &user.LastName,
		&user.LanguageCode, &user.IsBot, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &user, nil
}

func (r *telegramUserRepository) GetByChatID(ctx context.Context, chatID int64) (*models.TelegramUser, error) {
	query := `
		SELECT id, chat_id, username, first_name, last_name, language_code,
		       is_bot, is_active, created_at, updated_at
		FROM telegram_users
		WHERE chat_id = $1
	`

	var user models.TelegramUser
	err := r.pool.QueryRow(ctx, query, chatID).Scan(
		&user.ID, &user.ChatID, &user.Username, &user.FirstName, &user.LastName,
		&user.LanguageCode, &user.IsBot, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by chat_id: %w", err)
	}

	return &user, nil
}

func (r *telegramUserRepository) GetAll(ctx context.Context) ([]models.TelegramUser, error) {
	query := `
		SELECT id, chat_id, username, first_name, last_name, language_code,
		       is_bot, is_active, created_at, updated_at
		FROM telegram_users
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	defer rows.Close()

	var users []models.TelegramUser
	for rows.Next() {
		var user models.TelegramUser
		err := rows.Scan(
			&user.ID, &user.ChatID, &user.Username, &user.FirstName, &user.LastName,
			&user.LanguageCode, &user.IsBot, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *telegramUserRepository) GetAllActive(ctx context.Context) ([]models.TelegramUser, error) {
	query := `
		SELECT id, chat_id, username, first_name, last_name, language_code,
		       is_bot, is_active, created_at, updated_at
		FROM telegram_users
		WHERE is_active = true
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}
	defer rows.Close()

	var users []models.TelegramUser
	for rows.Next() {
		var user models.TelegramUser
		err := rows.Scan(
			&user.ID, &user.ChatID, &user.Username, &user.FirstName, &user.LastName,
			&user.LanguageCode, &user.IsBot, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *telegramUserRepository) UpdateActivity(ctx context.Context, chatID int64, isActive bool) error {
	query := `
		UPDATE telegram_users
		SET is_active = $1, updated_at = NOW()
		WHERE chat_id = $2
	`

	_, err := r.pool.Exec(ctx, query, isActive, chatID)
	return err
}
