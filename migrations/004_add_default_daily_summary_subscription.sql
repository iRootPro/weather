-- +goose Up
-- Добавляем подписку на утреннюю сводку всем существующим активным пользователям
INSERT INTO telegram_subscriptions (user_id, event_type, is_active, created_at, updated_at)
SELECT id, 'daily_summary', true, NOW(), NOW()
FROM telegram_users
WHERE is_active = true
ON CONFLICT (user_id, event_type) DO NOTHING;

-- +goose Down
-- Удаляем подписки на утреннюю сводку
DELETE FROM telegram_subscriptions WHERE event_type = 'daily_summary';
