package maxbot

func mainKeyboard() []interface{} {
	return []interface{}{
		ReplyKeyboardAttachment{
			Type: "reply_keyboard",
			Buttons: [][]ReplyButton{
				{{Type: "message", Text: "🌦️ Погода", Payload: "menu_weather"}},
				{{Type: "message", Text: "🔔 Подписки", Payload: "menu_subscribe"}, {Type: "message", Text: "📖 Помощь", Payload: "menu_help"}},
			},
		},
	}
}

func inlineMainKeyboard() []interface{} {
	return []interface{}{
		InlineKeyboardAttachment{
			Type: "inline_keyboard",
			Payload: InlineKeyboardPayload{Buttons: [][]Button{
				{{Type: "callback", Text: "🌦️ Погода", Payload: "cmd_weather"}},
				{{Type: "callback", Text: "🔔 Подписки", Payload: "cmd_subscribe"}},
			}},
		},
	}
}

func subscriptionKeyboard() []interface{} {
	return []interface{}{
		InlineKeyboardAttachment{
			Type: "inline_keyboard",
			Payload: InlineKeyboardPayload{Buttons: [][]Button{
				{{Type: "callback", Text: "🌅 Утренняя сводка", Payload: "sub_daily_summary"}},
				{{Type: "callback", Text: "🔔 Все события", Payload: "sub_all"}},
				{{Type: "callback", Text: "🌧️ Дождь", Payload: "sub_rain"}, {Type: "callback", Text: "🌡️ Температура", Payload: "sub_temperature"}},
				{{Type: "callback", Text: "💨 Ветер", Payload: "sub_wind"}, {Type: "callback", Text: "🔽 Давление", Payload: "sub_pressure"}},
				{{Type: "callback", Text: "❌ Отписаться от всех", Payload: "unsub_all"}},
			}},
		},
	}
}
