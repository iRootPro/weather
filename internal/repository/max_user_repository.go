package repository

import (
	"context"
	"fmt"

	"github.com/iRootPro/weather/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type maxUserRepository struct{ pool *pgxpool.Pool }

func NewMaxUserRepository(pool *pgxpool.Pool) MaxUserRepository {
	return &maxUserRepository{pool: pool}
}

func (r *maxUserRepository) Create(ctx context.Context, user *models.MaxUser) error {
	query := `
		INSERT INTO max_users (user_id, username, first_name, last_name, language_code, is_bot)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			language_code = EXCLUDED.language_code,
			is_bot = EXCLUDED.is_bot,
			is_active = true,
			updated_at = NOW()
		RETURNING id, created_at, updated_at, is_active
	`
	return r.pool.QueryRow(ctx, query, user.UserID, user.Username, user.FirstName, user.LastName, user.LanguageCode, user.IsBot).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.IsActive)
}

func (r *maxUserRepository) GetByID(ctx context.Context, id int64) (*models.MaxUser, error) {
	query := `SELECT id, user_id, username, first_name, last_name, language_code, is_bot, is_active, created_at, updated_at FROM max_users WHERE id = $1`
	var user models.MaxUser
	err := r.pool.QueryRow(ctx, query, id).Scan(&user.ID, &user.UserID, &user.Username, &user.FirstName, &user.LastName, &user.LanguageCode, &user.IsBot, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get max user by id: %w", err)
	}
	return &user, nil
}

func (r *maxUserRepository) GetByUserID(ctx context.Context, userID int64) (*models.MaxUser, error) {
	query := `SELECT id, user_id, username, first_name, last_name, language_code, is_bot, is_active, created_at, updated_at FROM max_users WHERE user_id = $1`
	var user models.MaxUser
	err := r.pool.QueryRow(ctx, query, userID).Scan(&user.ID, &user.UserID, &user.Username, &user.FirstName, &user.LastName, &user.LanguageCode, &user.IsBot, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get max user by user_id: %w", err)
	}
	return &user, nil
}

func (r *maxUserRepository) GetAllActive(ctx context.Context) ([]models.MaxUser, error) {
	query := `SELECT id, user_id, username, first_name, last_name, language_code, is_bot, is_active, created_at, updated_at FROM max_users WHERE is_active = true ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active max users: %w", err)
	}
	defer rows.Close()
	users := []models.MaxUser{}
	for rows.Next() {
		var user models.MaxUser
		if err := rows.Scan(&user.ID, &user.UserID, &user.Username, &user.FirstName, &user.LastName, &user.LanguageCode, &user.IsBot, &user.IsActive, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan max user: %w", err)
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *maxUserRepository) UpdateActivity(ctx context.Context, userID int64, isActive bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE max_users SET is_active = $1, updated_at = NOW() WHERE user_id = $2`, isActive, userID)
	return err
}
