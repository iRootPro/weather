package maxbot

import "encoding/json"

type User struct {
	UserID    int64   `json:"user_id"`
	FirstName string  `json:"first_name"`
	LastName  *string `json:"last_name"`
	Username  *string `json:"username"`
	IsBot     bool    `json:"is_bot"`
	Name      *string `json:"name"`
}

type BotInfo struct {
	User
}

type Recipient struct {
	ChatID   *int64 `json:"chat_id"`
	ChatType string `json:"chat_type"`
	UserID   *int64 `json:"user_id"`
}

type MessageBody struct {
	MID  string  `json:"mid"`
	Seq  int64   `json:"seq"`
	Text *string `json:"text"`
}

type Message struct {
	Sender    *User       `json:"sender"`
	Recipient Recipient   `json:"recipient"`
	Timestamp int64       `json:"timestamp"`
	Body      MessageBody `json:"body"`
}

type Callback struct {
	Timestamp  int64  `json:"timestamp"`
	CallbackID string `json:"callback_id"`
	Payload    string `json:"payload"`
	User       User   `json:"user"`
}

type Update struct {
	UpdateType string          `json:"update_type"`
	Timestamp  int64           `json:"timestamp"`
	Message    *Message        `json:"message"`
	Callback   *Callback       `json:"callback"`
	UserLocale *string         `json:"user_locale"`
	Raw        json.RawMessage `json:"-"`
}

type UpdateList struct {
	Updates []Update `json:"updates"`
	Marker  *int64   `json:"marker"`
}

type NewMessageBody struct {
	Text        string        `json:"text,omitempty"`
	Attachments []interface{} `json:"attachments,omitempty"`
	Notify      *bool         `json:"notify,omitempty"`
	Format      string        `json:"format,omitempty"`
}

type InlineKeyboardAttachment struct {
	Type    string                `json:"type"`
	Payload InlineKeyboardPayload `json:"payload"`
}

type InlineKeyboardPayload struct {
	Buttons [][]Button `json:"buttons"`
}

type Button struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Payload string `json:"payload,omitempty"`
	URL     string `json:"url,omitempty"`
}

type SendMessageResult struct {
	Message Message `json:"message"`
}

type CallbackAnswer struct {
	Notification string `json:"notification,omitempty"`
}
