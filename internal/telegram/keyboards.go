package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// GetReplyKeyboard возвращает постоянную клавиатуру с основными командами
func GetReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🌦️ Погода"),
			tgbotapi.NewKeyboardButton("📈 Статистика"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏆 Рекорды"),
			tgbotapi.NewKeyboardButton("☀️ Солнце"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🌙 Луна"),
			tgbotapi.NewKeyboardButton("🔔 Подписки"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📖 Помощь"),
		),
	)
}

// GetAdminReplyKeyboard возвращает клавиатуру для админов с дополнительными командами
func GetAdminReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🌦️ Погода"),
			tgbotapi.NewKeyboardButton("📈 Статистика"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏆 Рекорды"),
			tgbotapi.NewKeyboardButton("☀️ Солнце"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🌙 Луна"),
			tgbotapi.NewKeyboardButton("🔔 Подписки"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("👥 Пользователи"),
			tgbotapi.NewKeyboardButton("📖 Помощь"),
		),
	)
}

// GetMainKeyboard возвращает главную клавиатуру
func GetMainKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌦️ Погода", "cmd_weather"),
			tgbotapi.NewInlineKeyboardButtonData("📈 Статистика", "cmd_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏆 Рекорды", "cmd_records"),
			tgbotapi.NewInlineKeyboardButtonData("🔔 Подписки", "cmd_subscribe"),
		),
	)
}

// GetWeatherDetailKeyboard возвращает клавиатуру с деталями погоды
func GetWeatherDetailKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("☀️ Солнце", "cmd_sun"),
			tgbotapi.NewInlineKeyboardButtonData("🌙 Луна", "cmd_moon"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📈 Статистика", "cmd_stats"),
			tgbotapi.NewInlineKeyboardButtonData("🏆 Рекорды", "cmd_records"),
		),
	)
}

// GetStatsKeyboard возвращает клавиатуру выбора периода статистики
func GetStatsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 День", "stats_day"),
			tgbotapi.NewInlineKeyboardButtonData("📅 Неделя", "stats_week"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Месяц", "stats_month"),
			tgbotapi.NewInlineKeyboardButtonData("📅 Год", "stats_year"),
		),
	)
}

// GetSubscriptionKeyboard возвращает клавиатуру подписок
func GetSubscriptionKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌅 Утренняя сводка", "sub_daily_summary"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔔 Все события", "sub_all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌧️ Дождь", "sub_rain"),
			tgbotapi.NewInlineKeyboardButtonData("🌡️ Температура", "sub_temperature"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💨 Ветер", "sub_wind"),
			tgbotapi.NewInlineKeyboardButtonData("🔽 Давление", "sub_pressure"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отписаться от всех", "unsub_all"),
		),
	)
}
