package maxbot

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
