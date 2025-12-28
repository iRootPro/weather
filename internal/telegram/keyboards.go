package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
