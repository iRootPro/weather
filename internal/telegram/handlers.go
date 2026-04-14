package telegram

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/iRootPro/weather/internal/models"
	"github.com/iRootPro/weather/internal/service"
)

func (h *BotHandler) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	// Регистрация/обновление пользователя
	user := &models.TelegramUser{
		ChatID:       msg.Chat.ID,
		Username:     &msg.From.UserName,
		FirstName:    &msg.From.FirstName,
		LastName:     &msg.From.LastName,
		LanguageCode: msg.From.LanguageCode,
		IsBot:        msg.From.IsBot,
	}

	// Проверяем, новый ли пользователь
	existingUser, _ := h.userRepo.GetByChatID(ctx, msg.Chat.ID)
	isNewUser := existingUser == nil

	if err := h.userRepo.Create(ctx, user); err != nil {
		h.logger.Error("failed to create/update user", "error", err)
	}

	// Если это новый пользователь, автоматически подписываем на утреннюю сводку
	if isNewUser {
		subscription := &models.TelegramSubscription{
			UserID:    user.ID,
			EventType: EventDailySummary,
			IsActive:  true,
		}
		if err := h.subRepo.Create(ctx, subscription); err != nil {
			h.logger.Error("failed to create default subscription", "error", err)
		} else {
			h.logger.Info("auto-subscribed new user to daily summary", "chat_id", msg.Chat.ID)
		}
	}

	h.logger.Info("command received",
		"command", msg.Command(),
		"chat_id", msg.Chat.ID,
		"username", msg.From.UserName,
	)

	switch msg.Command() {
	case CmdStart:
		h.handleStart(ctx, msg)
	case CmdHelp:
		h.handleHelp(ctx, msg)
	case CmdWeather, CmdCurrent:
		h.handleCurrentWeather(ctx, msg)
	case CmdStats:
		h.handleStats(ctx, msg)
	case CmdRecords:
		h.handleRecords(ctx, msg)
	case CmdHistory:
		h.handleHistory(ctx, msg)
	case CmdSun:
		h.handleSun(ctx, msg)
	case CmdMoon:
		h.handleMoon(ctx, msg)
	case CmdSubscribe:
		h.handleSubscribe(ctx, msg)
	case CmdUnsubscribe:
		h.handleUnsubscribe(ctx, msg)
	case CmdUsers:
		h.handleUsers(ctx, msg)
	case CmdMyID:
		h.handleMyID(ctx, msg)
	case CmdTestSummary:
		h.handleTestSummary(ctx, msg)
	case CmdForecast:
		h.handleForecast(ctx, msg)
	case CmdAnnounce:
		h.handleAnnounce(ctx, msg)
	case CmdAnnouncePreview:
		h.handleAnnouncePreview(ctx, msg)
	default:
		h.sendMessage(msg.Chat.ID, "Неизвестная команда. Используйте /help для списка команд.")
	}
}

func (h *BotHandler) handleStart(ctx context.Context, msg *tgbotapi.Message) {
	text := `🌦️ *Добро пожаловать в бот метеостанции города Армавир!*

Я могу предоставить вам актуальную информацию о погоде и отправлять уведомления о важных изменениях.

📸 *Фотогалерея*
Вы можете присылать свои фотографии — они будут добавлены в галерею с привязкой к погодным условиям! Просто отправьте фото в этот бот как документ (без сжатия).
` + fmt.Sprintf("Галерея доступна на сайте: %s/gallery\n", h.websiteURL) + `
Используйте кнопки ниже для быстрого доступа к информации или команды:
/weather - текущая погода
/stats - статистика
/subscribe - подписаться на уведомления

Для полного списка команд используйте /help

💡 Есть идеи для улучшения? Обращайся @iRootPro`

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"

	// Показываем разные клавиатуры для админов и обычных пользователей
	if h.isAdmin(msg.Chat.ID) {
		reply.ReplyMarkup = GetAdminReplyKeyboard()
	} else {
		reply.ReplyMarkup = GetReplyKeyboard()
	}

	h.bot.Send(reply)
}

func (h *BotHandler) handleHelp(ctx context.Context, msg *tgbotapi.Message) {
	text := `📖 *Справка по командам*

*Основные:*
/weather - текущая погода
/forecast - прогноз на 6 дней
/stats - статистика за период
/records - рекорды за всё время
/history - история данных

*Астрономия:*
/sun - восход и закат
/moon - фаза луны

*Уведомления:*
/subscribe - подписаться на события
/unsubscribe - отписаться

Используйте кнопки внизу экрана для быстрого доступа!

──────────────
🌍 *О прогнозе погоды*
Данные от Open-Meteo — бесплатного метеосервиса с открытым исходным кодом. Использует модели прогнозирования ведущих метеослужб (NOAA, DWD). Обновляется каждый час.

──────────────
💡 *Обратная связь*
Есть идеи для улучшения бота?
Пишите @iRootPro`

	// Добавляем админские команды для администраторов
	if h.isAdmin(msg.Chat.ID) {
		text += `

──────────────
🔧 *Админские команды:*
/users - список пользователей
/announce - массовая рассылка
/announce_preview - превью анонса
/test_summary - тест утренней сводки

📢 Пример использования:
/announce_preview 🔥 Текст анонса
/announce 🔥 Текст анонса`
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetReplyKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleCurrentWeather(ctx context.Context, msg *tgbotapi.Message) {
	current, hourAgo, dailyMinMax, err := h.weatherSvc.GetCurrentWithHourlyChange(ctx)
	if err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка получения данных о погоде")
		h.logger.Error("failed to get current weather", "error", err)
		return
	}

	text := FormatCurrentWeather(current, hourAgo, dailyMinMax)

	// Добавляем прогноз на ближайшее время
	if h.forecastSvc != nil {
		forecast, err := h.forecastSvc.GetTodayForecast(ctx)
		if err == nil && len(forecast) > 0 {
			todayForecast := formatTodayForecast(forecast)
			if len(todayForecast) > 0 {
				text += "\n\n🔮 *ПРОГНОЗ НА СЕГОДНЯ*\n"
				for _, f := range todayForecast {
					text += fmt.Sprintf("%s В %02d:00: %.0f°C", f.Icon, f.Hour, f.Temperature)
					if f.PrecipitationProbability > 0 {
						text += fmt.Sprintf(" · 💧%d%%", f.PrecipitationProbability)
					}
					text += fmt.Sprintf(" · %s\n", f.WeatherDescription)
				}
			}
		}
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetWeatherDetailKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleStats(ctx context.Context, msg *tgbotapi.Message) {
	args := msg.CommandArguments()
	period := "day"
	if args != "" {
		period = args
	}

	stats, err := h.weatherSvc.GetStats(ctx, period)
	if err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка получения статистики")
		h.logger.Error("failed to get stats", "error", err)
		return
	}

	text := FormatStats(stats)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetStatsKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleRecords(ctx context.Context, msg *tgbotapi.Message) {
	records, err := h.weatherSvc.GetRecords(ctx)
	if err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка получения рекордов")
		h.logger.Error("failed to get records", "error", err)
		return
	}

	text := FormatRecords(records)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetMainKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleHistory(ctx context.Context, msg *tgbotapi.Message) {
	h.sendMessage(msg.Chat.ID, "История в разработке. Используйте /stats для статистики.")
}

func (h *BotHandler) handleSun(ctx context.Context, msg *tgbotapi.Message) {
	sunData := h.sunSvc.GetTodaySunTimesWithComparison()

	text := FormatSunData(sunData)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetMainKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleMoon(ctx context.Context, msg *tgbotapi.Message) {
	moonData := h.moonSvc.GetTodayMoonData()

	text := FormatMoonData(moonData)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetMainKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleSubscribe(ctx context.Context, msg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Выберите тип уведомлений:")
	reply.ReplyMarkup = GetSubscriptionKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleUnsubscribe(ctx context.Context, msg *tgbotapi.Message) {
	user, err := h.userRepo.GetByChatID(ctx, msg.Chat.ID)
	if err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка: пользователь не найден")
		return
	}

	if err := h.subRepo.DeleteAll(ctx, user.ID); err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка отписки")
		h.logger.Error("failed to unsubscribe", "error", err)
		return
	}

	h.sendMessage(msg.Chat.ID, "✅ Вы успешно отписались от всех уведомлений")
}

func (h *BotHandler) handleCallbackQuery(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	// Подтверждаем получение callback
	h.bot.Request(tgbotapi.NewCallback(callback.ID, ""))

	user, err := h.userRepo.GetByChatID(ctx, callback.Message.Chat.ID)
	if err != nil {
		h.logger.Error("failed to get user", "error", err)
		return
	}

	// Обработка подписок
	if strings.HasPrefix(data, "sub_") {
		eventType := strings.TrimPrefix(data, "sub_")

		sub := &models.TelegramSubscription{
			UserID:    user.ID,
			EventType: eventType,
			IsActive:  true,
		}

		if err := h.subRepo.Create(ctx, sub); err != nil {
			h.logger.Error("failed to create subscription", "error", err)
			return
		}

		text := fmt.Sprintf("✅ Вы подписались на уведомления: %s", GetEventTypeName(eventType))
		h.bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, text))
		return
	}

	// Обработка отписок
	if strings.HasPrefix(data, "unsub_") {
		eventType := strings.TrimPrefix(data, "unsub_")

		if eventType == "all" {
			h.subRepo.DeleteAll(ctx, user.ID)
			h.bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "✅ Вы отписались от всех уведомлений"))
		} else {
			h.subRepo.Delete(ctx, user.ID, eventType)
			text := fmt.Sprintf("✅ Вы отписались от: %s", GetEventTypeName(eventType))
			h.bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, text))
		}
		return
	}

	// Обработка модерации фото - одобрение
	if strings.HasPrefix(data, "approve_photo_") {
		h.handlePhotoApproval(ctx, callback, data)
		return
	}

	// Обработка модерации фото - отклонение
	if strings.HasPrefix(data, "reject_photo_") {
		h.handlePhotoRejection(ctx, callback, data)
		return
	}

	// Обработка команд через кнопки
	switch data {
	case "cmd_weather":
		h.handleCurrentWeather(ctx, callback.Message)
	case "cmd_stats":
		h.handleStats(ctx, callback.Message)
	case "cmd_records":
		h.handleRecords(ctx, callback.Message)
	case "cmd_sun":
		h.handleSun(ctx, callback.Message)
	case "cmd_moon":
		h.handleMoon(ctx, callback.Message)
	case "cmd_subscribe":
		h.handleSubscribe(ctx, callback.Message)
	case "stats_day", "stats_week", "stats_month", "stats_year":
		period := strings.TrimPrefix(data, "stats_")
		msg := &tgbotapi.Message{
			Chat: callback.Message.Chat,
			From: callback.From,
			Text: "/stats " + period,
		}
		h.handleStats(ctx, msg)
	}
}

func (h *BotHandler) handleUsers(ctx context.Context, msg *tgbotapi.Message) {
	// Проверка прав админа
	if !h.isAdmin(msg.Chat.ID) {
		h.sendMessage(msg.Chat.ID, "❌ У вас нет доступа к этой команде")
		return
	}

	users, err := h.userRepo.GetAll(ctx)
	if err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка получения списка пользователей")
		h.logger.Error("failed to get users", "error", err)
		return
	}

	h.logger.Info("formatting users list", "count", len(users))
	text := FormatUsersList(users)
	h.logger.Debug("formatted text", "length", len(text))

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"

	if _, err := h.bot.Send(reply); err != nil {
		h.logger.Error("failed to send users list", "error", err, "text_length", len(text))
		h.sendMessage(msg.Chat.ID, "❌ Ошибка отправки списка пользователей")
		return
	}

	h.logger.Info("users list sent successfully", "count", len(users))
}

func (h *BotHandler) handleMyID(ctx context.Context, msg *tgbotapi.Message) {
	text := fmt.Sprintf("🆔 *Ваш Chat ID:* `%d`\n\nИспользуйте этот ID для настройки админских прав в переменной окружения TELEGRAM_ADMIN_IDS", msg.Chat.ID)
	h.sendMessage(msg.Chat.ID, text)
}

func (h *BotHandler) handleTestSummary(ctx context.Context, msg *tgbotapi.Message) {
	// Проверка прав админа
	if !h.isAdmin(msg.Chat.ID) {
		h.sendMessage(msg.Chat.ID, "❌ У вас нет доступа к этой команде")
		return
	}

	// Получаем текущие данные о погоде
	current, err := h.weatherSvc.GetLatest(ctx)
	if err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка получения данных о погоде")
		h.logger.Error("failed to get current weather", "error", err)
		return
	}

	// Получаем данные за вчера в это же время
	yesterdaySame, err := h.weatherSvc.GetDataNearTime(ctx, current.Time.Add(-24*time.Hour))
	if err != nil {
		h.logger.Warn("failed to get yesterday weather", "error", err)
	}

	// Получаем min/max за ночь (00:00 - 07:00 сегодня)
	now := time.Now()
	nightStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	nightEnd := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())
	nightMinMax, err := h.weatherSvc.GetMinMaxInRange(ctx, nightStart, nightEnd)
	if err != nil {
		h.logger.Warn("failed to get night min/max", "error", err)
	}

	// Получаем min/max за сегодня
	dailyMinMax, err := h.weatherSvc.GetDailyMinMax(ctx)
	if err != nil {
		h.logger.Warn("failed to get daily min/max", "error", err)
	}

	// Получаем данные о солнце
	sunData := h.sunSvc.GetTodaySunTimesWithComparison()

	// Получаем прогноз на сегодня
	var todayForecast []DayForecastInfo
	if h.forecastSvc != nil {
		forecast, err := h.forecastSvc.GetTodayForecast(ctx)
		if err != nil {
			h.logger.Warn("failed to get today forecast", "error", err)
		} else if len(forecast) > 0 {
			todayForecast = formatTodayForecast(forecast)
		}
	}

	// Получаем магнитную обстановку (если сервис подключён)
	var geomagSnap *service.DashboardSnapshot
	if h.geomagSvc != nil {
		snap, err := h.geomagSvc.GetDashboardSnapshot(ctx, time.Now())
		if err != nil {
			h.logger.Warn("failed to get geomagnetic snapshot", "error", err)
		} else if snap != nil && snap.HasData {
			geomagSnap = snap
		}
	}

	// Форматируем сообщение
	text := FormatDailySummary(current, yesterdaySame, nightMinMax, dailyMinMax, sunData, todayForecast, geomagSnap)

	// Добавляем пометку о тестовой рассылке
	testNote := "\n\n🧪 *Тестовая рассылка* (только для админа)"

	reply := tgbotapi.NewMessage(msg.Chat.ID, text+testNote)
	reply.ParseMode = "Markdown"
	h.bot.Send(reply)

	h.logger.Info("test summary sent", "chat_id", msg.Chat.ID)
}

func (h *BotHandler) handleForecast(ctx context.Context, msg *tgbotapi.Message) {
	if h.forecastSvc == nil {
		h.sendMessage(msg.Chat.ID, "❌ Прогноз погоды временно недоступен")
		return
	}

	// Получаем прогноз на 5 дней
	forecast, err := h.forecastSvc.GetDailyForecast(ctx, 5)
	if err != nil {
		h.sendMessage(msg.Chat.ID, "❌ Ошибка получения прогноза")
		h.logger.Error("failed to get forecast", "error", err)
		return
	}

	if len(forecast) == 0 {
		h.sendMessage(msg.Chat.ID, "Прогноз пока недоступен. Данные обновляются каждый час.")
		return
	}

	text := FormatForecast(forecast)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = GetMainKeyboard()
	h.bot.Send(reply)
}

func (h *BotHandler) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	// Обработка нажатий на кнопки постоянной клавиатуры
	switch msg.Text {
	case "🌦️ Погода":
		h.handleCurrentWeather(ctx, msg)
	case "📈 Статистика":
		h.handleStats(ctx, msg)
	case "🏆 Рекорды":
		h.handleRecords(ctx, msg)
	case "📊 Прогноз":
		h.handleForecast(ctx, msg)
	case "☀️ Солнце":
		h.handleSun(ctx, msg)
	case "🌙 Луна":
		h.handleMoon(ctx, msg)
	case "🔔 Подписки":
		h.handleSubscribe(ctx, msg)
	case "👥 Пользователи":
		h.handleUsers(ctx, msg)
	case "📖 Помощь":
		h.handleHelp(ctx, msg)
	default:
		h.sendMessage(msg.Chat.ID, "Используйте кнопки ниже или /help для списка команд")
	}
}

// handlePhotoDocument обрабатывает фото отправленное как документ (без сжатия)
func (h *BotHandler) handlePhotoDocument(ctx context.Context, msg *tgbotapi.Message) {
	// Получаем пользователя
	user, err := h.userRepo.GetByChatID(ctx, msg.Chat.ID)
	if err != nil {
		h.logger.Error("failed to get user", "error", err)
		h.sendMessage(msg.Chat.ID, "❌ Ошибка при обработке фотографии")
		return
	}

	// Отправляем уведомление о начале обработки
	processingMsg := tgbotapi.NewMessage(msg.Chat.ID, "⏳ Обрабатываю фотографию...")
	processingMsg.ParseMode = "Markdown"
	sentMsg, _ := h.bot.Send(processingMsg)

	document := msg.Document

	// Логируем информацию о документе
	h.logger.Info("received document",
		"mime_type", document.MimeType,
		"file_name", document.FileName,
		"file_size", document.FileSize,
		"file_id", document.FileID)

	// Скачиваем документ
	fileConfig := tgbotapi.FileConfig{FileID: document.FileID}
	file, err := h.bot.GetFile(fileConfig)
	if err != nil {
		h.logger.Error("failed to get file", "error", err)
		h.sendMessage(msg.Chat.ID, "❌ Ошибка при скачивании фотографии")
		return
	}

	// Получаем URL файла
	fileURL := file.Link(h.bot.Token)

	// Скачиваем файл через http.Get
	httpResp, err := http.Get(fileURL)
	if err != nil {
		h.logger.Error("failed to download file", "error", err)
		h.sendMessage(msg.Chat.ID, "❌ Ошибка при скачивании фотографии")
		return
	}
	defer httpResp.Body.Close()

	// Читаем данные в буфер
	fileData := new(bytes.Buffer)
	_, err = io.Copy(fileData, httpResp.Body)
	if err != nil {
		h.logger.Error("failed to read file data", "error", err)
		h.sendMessage(msg.Chat.ID, "❌ Ошибка при чтении фотографии")
		return
	}

	// Определяем расширение файла на основе MIME типа
	originalExt := getFileExtension(document.MimeType, document.FileName)
	isHEIC := document.MimeType == "image/heic" || document.MimeType == "image/heif"

	// Создаем временный файл для оригинала
	tempFilename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), document.FileUniqueID, originalExt)
	tempFilepath := fmt.Sprintf("photos/%s", tempFilename)

	// Создаем директорию если её нет
	if err := os.MkdirAll("photos", 0755); err != nil {
		h.logger.Error("failed to create photos directory", "error", err)
		h.sendMessage(msg.Chat.ID, "❌ Ошибка при создании директории для фото")
		return
	}

	h.logger.Info("saving temporary file to disk", "filename", tempFilename, "filepath", tempFilepath)

	// Сохраняем временный файл
	tempFile, err := os.Create(tempFilepath)
	if err != nil {
		h.logger.Error("failed to create temp file", "error", err, "filepath", tempFilepath)
		h.sendMessage(msg.Chat.ID, "❌ Ошибка при сохранении фотографии")
		return
	}

	bytesWritten, err := io.Copy(tempFile, bytes.NewReader(fileData.Bytes()))
	tempFile.Close()
	if err != nil {
		h.logger.Error("failed to write temp file", "error", err)
		h.sendMessage(msg.Chat.ID, "❌ Ошибка при сохранении фотографии")
		return
	}

	h.logger.Info("temp file saved", "filepath", tempFilepath, "bytes", bytesWritten)

	// Проверяем что файл существует и имеет размер
	fileInfo, err := os.Stat(tempFilepath)
	if err != nil {
		h.logger.Error("failed to stat temp file", "error", err, "filepath", tempFilepath)
		h.sendMessage(msg.Chat.ID, "❌ Ошибка при проверке файла")
		return
	}
	h.logger.Info("temp file verified", "size", fileInfo.Size(), "name", fileInfo.Name())

	// Извлекаем EXIF данные из временного файла
	exifData, err := ExtractExifDataFromFile(tempFilepath, h.timezone)
	if err != nil {
		h.logger.Warn("failed to extract exif from file", "error", err, "filepath", tempFilepath)
		// Продолжаем без EXIF данных - используем текущее время
		exifData = &ExifData{
			TakenAt: time.Now(),
		}
	}

	h.logger.Info("exif extracted", "taken_at", exifData.TakenAt, "camera", fmt.Sprintf("%s %s", exifData.CameraMake, exifData.CameraModel))

	// Определяем финальное имя файла (для веба нужен JPEG)
	var finalFilename string
	var finalFilepath string

	if isHEIC {
		// Конвертируем HEIC в JPEG используя Python скрипт с pillow-heif
		finalFilename = fmt.Sprintf("%d_%s.jpg", time.Now().Unix(), document.FileUniqueID)
		finalFilepath = fmt.Sprintf("photos/%s", finalFilename)

		h.logger.Info("converting HEIC to JPEG using Python", "input", tempFilepath, "output", finalFilepath)

		// Вызываем Python скрипт для конвертации
		convertCmd := exec.Command("python3", "/app/convert_heic.py", tempFilepath, finalFilepath)
		convertOutput, err := convertCmd.CombinedOutput()
		if err != nil {
			h.logger.Error("failed to convert HEIC to JPEG", "error", err, "output", string(convertOutput))
			h.sendMessage(msg.Chat.ID, "❌ Ошибка при конвертации HEIC в JPEG")
			// Удаляем временный файл
			os.Remove(tempFilepath)
			return
		}

		h.logger.Info("HEIC converted to JPEG successfully", "filepath", finalFilepath, "output", string(convertOutput))

		// Удаляем временный HEIC файл после конвертации
		os.Remove(tempFilepath)
	} else {
		// Для других форматов просто используем временный файл как финальный
		finalFilename = tempFilename
		finalFilepath = tempFilepath
	}

	// Получаем погоду на момент съемки
	weather, err := h.photoRepo.GetWeatherForTime(ctx, exifData.TakenAt)
	if err != nil {
		h.logger.Warn("failed to get weather for photo time", "error", err, "taken_at", exifData.TakenAt)
	}

	// Проверяем, является ли пользователь админом
	isAdmin := h.isAdmin(msg.Chat.ID)

	// Создаем запись в БД
	photoModel := &models.Photo{
		Filename:       finalFilename,
		FilePath:       finalFilepath,
		Caption:        msg.Caption,
		TakenAt:        exifData.TakenAt,
		CameraMake:     exifData.CameraMake,
		CameraModel:    exifData.CameraModel,
		TelegramFileID: document.FileID,
		TelegramUserID: &user.ID,
		IsVisible:      isAdmin, // Админские фото сразу видны, остальные - на модерации
	}

	// Добавляем погодные данные если есть
	if weather != nil {
		if weather.TempOutdoor != nil {
			temp := float64(*weather.TempOutdoor)
			photoModel.Temperature = &temp
		}
		if weather.HumidityOutdoor != nil {
			humidity := float64(*weather.HumidityOutdoor)
			photoModel.Humidity = &humidity
		}
		if weather.PressureRelative != nil {
			pressure := float64(*weather.PressureRelative)
			photoModel.Pressure = &pressure
		}
		if weather.WindSpeed != nil {
			windSpeed := float64(*weather.WindSpeed)
			photoModel.WindSpeed = &windSpeed
		}
		if weather.WindDirection != nil {
			windDir := int(*weather.WindDirection)
			photoModel.WindDirection = &windDir
		}
		if weather.RainRate != nil {
			rainRate := float64(*weather.RainRate)
			photoModel.RainRate = &rainRate
		}
		if weather.SolarRadiation != nil {
			solarRad := float64(*weather.SolarRadiation)
			photoModel.SolarRadiation = &solarRad
		}
		photoModel.WeatherDescription = formatWeatherDescription(weather)
	}

	// Сохраняем в БД
	err = h.photoRepo.Create(ctx, photoModel)
	if err != nil {
		h.logger.Error("failed to save photo to db", "error", err)
		h.sendMessage(msg.Chat.ID, "❌ Ошибка при сохранении фотографии в базу данных")
		return
	}

	// Удаляем сообщение о обработке
	deleteMsg := tgbotapi.NewDeleteMessage(msg.Chat.ID, sentMsg.MessageID)
	h.bot.Send(deleteMsg)

	var confirmText string
	if isAdmin {
		// Подтверждение для админа - фото сразу добавлено
		confirmText = "✅ *Фотография добавлена!*\n\n"
		confirmText += fmt.Sprintf("📅 Дата съемки: %s\n", exifData.TakenAt.Format("02.01.2006 15:04"))

		if exifData.CameraMake != "" || exifData.CameraModel != "" {
			confirmText += fmt.Sprintf("📷 Камера: %s %s\n", exifData.CameraMake, exifData.CameraModel)
		}

		if weather != nil {
			confirmText += "\n🌡️ Погода на момент съемки:\n"
			if weather.TempOutdoor != nil {
				confirmText += fmt.Sprintf("• Температура: %.1f°C\n", *weather.TempOutdoor)
			}
			if weather.HumidityOutdoor != nil {
				confirmText += fmt.Sprintf("• Влажность: %d%%\n", *weather.HumidityOutdoor)
			}
			if weather.PressureRelative != nil {
				confirmText += fmt.Sprintf("• Давление: %.0f мм рт.ст.\n", *weather.PressureRelative)
			}
			if weather.RainRate != nil && *weather.RainRate >= 0.1 {
				confirmText += fmt.Sprintf("• Дождь: %.1f мм/ч\n", *weather.RainRate)
			}
		}

		h.logger.Info("admin photo uploaded directly", "chat_id", msg.Chat.ID, "photo_id", photoModel.ID, "taken_at", exifData.TakenAt)
	} else {
		// Подтверждение для обычного пользователя - отправлено на модерацию
		confirmText = "✅ *Фотография получена!*\n\n"
		confirmText += "📋 Ваша фотография отправлена на модерацию.\n"
		confirmText += "⏳ Модератор рассмотрит её в ближайшее время.\n\n"
		confirmText += "📬 Вы получите уведомление о результате проверки."

		// Отправляем уведомление админам для модерации
		h.sendPhotoModerationToAdmins(ctx, photoModel, exifData, weather, finalFilepath)

		h.logger.Info("photo uploaded and sent for moderation", "chat_id", msg.Chat.ID, "photo_id", photoModel.ID, "taken_at", exifData.TakenAt)
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, confirmText)
	reply.ParseMode = "Markdown"
	h.bot.Send(reply)
}

func (h *BotHandler) handlePhoto(ctx context.Context, msg *tgbotapi.Message) {
	// Сжатые фото не содержат EXIF данных, поэтому мы не можем получить реальное время съемки
	// Инструктируем пользователя отправлять как документ
	instructionText := `❌ *Фото не добавлено*

Сжатые фото не содержат информацию о времени съемки (EXIF), поэтому не могут быть добавлены в галерею.

📎 *Как правильно загрузить фото:*
1. Нажмите на скрепку 📎
2. Выберите "Файл" или "Document"
3. Выберите фото из галереи
4. Отправьте как файл (не сжимая)

Так будет сохранена информация о времени съемки и погода будет привязана корректно! 📸`

	reply := tgbotapi.NewMessage(msg.Chat.ID, instructionText)
	reply.ParseMode = "Markdown"
	h.bot.Send(reply)

	h.logger.Info("rejected compressed photo upload", "chat_id", msg.Chat.ID, "username", msg.From.UserName)
}

// formatWeatherDescription формирует описание погоды
func formatWeatherDescription(w *models.WeatherData) string {
	desc := ""

	if w.TempOutdoor != nil {
		desc = fmt.Sprintf("%.1f°C", *w.TempOutdoor)
	}

	if w.RainRate != nil && *w.RainRate > 0.1 {
		desc += ", дождь"
	} else if w.HumidityOutdoor != nil {
		if *w.HumidityOutdoor > 80 {
			desc += ", влажно"
		} else if *w.HumidityOutdoor < 30 {
			desc += ", сухо"
		}
	}

	if w.WindSpeed != nil && *w.WindSpeed > 5 {
		desc += fmt.Sprintf(", ветер %.1f м/с", *w.WindSpeed)
	}

	return desc
}

// getFileExtension определяет расширение файла на основе MIME типа
func getFileExtension(mimeType, fileName string) string {
	// Маппинг MIME типов на расширения
	mimeToExt := map[string]string{
		"image/jpeg":         ".jpg",
		"image/jpg":          ".jpg",
		"image/png":          ".png",
		"image/heic":         ".heic",
		"image/heif":         ".heic",
		"image/webp":         ".webp",
		"image/avif":         ".avif",
		"image/bmp":          ".bmp",
		"image/gif":          ".gif",
		"image/tiff":         ".tiff",
		"image/x-canon-cr2":  ".cr2",
		"image/x-nikon-nef":  ".nef",
		"image/x-sony-arw":   ".arw",
	}

	// Сначала пробуем по MIME типу
	if ext, ok := mimeToExt[mimeType]; ok {
		return ext
	}

	// Если не нашли, пробуем извлечь из имени файла
	if fileName != "" {
		for i := len(fileName) - 1; i >= 0; i-- {
			if fileName[i] == '.' {
				return fileName[i:]
			}
		}
	}

	// По умолчанию JPEG
	return ".jpg"
}

// sendPhotoModerationToAdmins отправляет уведомление админам для модерации фото
func (h *BotHandler) sendPhotoModerationToAdmins(ctx context.Context, photo *models.Photo, exif *ExifData, weather *models.WeatherData, filePath string) {
	// Формируем текст уведомления
	moderationText := "🔔 *Новое фото на модерацию*\n\n"

	// Получаем информацию об авторе
	if photo.TelegramUserID != nil {
		user, err := h.userRepo.GetByID(ctx, *photo.TelegramUserID)
		if err == nil {
			authorName := ""
			if user.FirstName != nil {
				authorName = *user.FirstName
			}
			if user.LastName != nil {
				authorName += " " + *user.LastName
			}
			if user.Username != nil {
				moderationText += fmt.Sprintf("👤 Автор: %s (@%s)\n", authorName, *user.Username)
			} else {
				moderationText += fmt.Sprintf("👤 Автор: %s\n", authorName)
			}
		}
	}

	moderationText += fmt.Sprintf("📅 Дата съемки: %s\n", exif.TakenAt.Format("02.01.2006 15:04"))

	if exif.CameraMake != "" || exif.CameraModel != "" {
		moderationText += fmt.Sprintf("📷 Камера: %s %s\n", exif.CameraMake, exif.CameraModel)
	}

	if photo.Caption != "" {
		moderationText += fmt.Sprintf("\n💬 Описание: %s\n", photo.Caption)
	}

	if weather != nil {
		moderationText += "\n🌡️ Погода на момент съемки:\n"
		if weather.TempOutdoor != nil {
			moderationText += fmt.Sprintf("• Температура: %.1f°C\n", *weather.TempOutdoor)
		}
		if weather.HumidityOutdoor != nil {
			moderationText += fmt.Sprintf("• Влажность: %d%%\n", *weather.HumidityOutdoor)
		}
		if weather.PressureRelative != nil {
			moderationText += fmt.Sprintf("• Давление: %.0f мм рт.ст.\n", *weather.PressureRelative)
		}
	}

	// Создаем инлайн-клавиатуру с кнопками модерации
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Одобрить", fmt.Sprintf("approve_photo_%d", photo.ID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("reject_photo_%d", photo.ID)),
		),
	)

	// Отправляем уведомление всем админам
	for _, adminID := range h.adminIDs {
		// Открываем файл для отправки
		photoFile, err := os.Open(filePath)
		if err != nil {
			h.logger.Error("failed to open photo for moderation", "error", err, "filepath", filePath)
			continue
		}

		photoBytes := tgbotapi.FileBytes{
			Name:  photo.Filename,
			Bytes: func() []byte {
				defer photoFile.Close()
				data, _ := io.ReadAll(photoFile)
				return data
			}(),
		}

		photoMsg := tgbotapi.NewPhoto(adminID, photoBytes)
		photoMsg.Caption = moderationText
		photoMsg.ParseMode = "Markdown"
		photoMsg.ReplyMarkup = keyboard

		if _, err := h.bot.Send(photoMsg); err != nil {
			h.logger.Error("failed to send moderation message to admin", "error", err, "admin_id", adminID)
		}
	}

	h.logger.Info("moderation request sent to admins", "photo_id", photo.ID, "admins_count", len(h.adminIDs))
}

// handlePhotoApproval обрабатывает одобрение фото админом
func (h *BotHandler) handlePhotoApproval(ctx context.Context, callback *tgbotapi.CallbackQuery, data string) {
	// Проверяем права админа
	if !h.isAdmin(callback.Message.Chat.ID) {
		h.bot.Request(tgbotapi.NewCallback(callback.ID, "❌ У вас нет прав для модерации"))
		return
	}

	// Извлекаем ID фото из callback data
	photoIDStr := strings.TrimPrefix(data, "approve_photo_")
	photoID, err := strconv.ParseInt(photoIDStr, 10, 64)
	if err != nil {
		h.logger.Error("failed to parse photo ID", "error", err, "data", data)
		h.bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Ошибка обработки"))
		return
	}

	// Получаем фото из БД
	photo, err := h.photoRepo.GetByID(ctx, photoID)
	if err != nil {
		h.logger.Error("failed to get photo", "error", err, "photo_id", photoID)
		h.bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Фото не найдено"))
		return
	}

	// Одобряем фото (делаем видимым)
	if err := h.photoRepo.UpdateVisibility(ctx, photoID, true); err != nil {
		h.logger.Error("failed to approve photo", "error", err, "photo_id", photoID)
		h.bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Ошибка одобрения"))
		return
	}

	// Отправляем уведомление пользователю
	if photo.TelegramUserID != nil {
		user, err := h.userRepo.GetByID(ctx, *photo.TelegramUserID)
		if err == nil {
			approvalText := "✅ *Ваше фото одобрено и добавлено в галерею!*\n\n"
			approvalText += fmt.Sprintf("📅 Дата съемки: %s\n\n", photo.TakenAt.Format("02.01.2006 15:04"))
			approvalText += fmt.Sprintf("🖼️ Посмотреть в галерее:\n%s/gallery", h.websiteURL)

			approvalMsg := tgbotapi.NewMessage(user.ChatID, approvalText)
			approvalMsg.ParseMode = "Markdown"
			approvalMsg.DisableWebPagePreview = false
			h.bot.Send(approvalMsg)
		}
	}

	// Редактируем сообщение админа (убираем кнопки)
	editText := callback.Message.Caption + "\n\n✅ *Фото одобрено*"
	editMsg := tgbotapi.NewEditMessageCaption(callback.Message.Chat.ID, callback.Message.MessageID, editText)
	editMsg.ParseMode = "Markdown"
	h.bot.Send(editMsg)

	// Подтверждаем callback
	h.bot.Request(tgbotapi.NewCallback(callback.ID, "✅ Фото одобрено"))

	h.logger.Info("photo approved", "photo_id", photoID, "admin_id", callback.Message.Chat.ID)
}

// handlePhotoRejection обрабатывает отклонение фото админом
func (h *BotHandler) handlePhotoRejection(ctx context.Context, callback *tgbotapi.CallbackQuery, data string) {
	// Проверяем права админа
	if !h.isAdmin(callback.Message.Chat.ID) {
		h.bot.Request(tgbotapi.NewCallback(callback.ID, "❌ У вас нет прав для модерации"))
		return
	}

	// Извлекаем ID фото из callback data
	photoIDStr := strings.TrimPrefix(data, "reject_photo_")
	photoID, err := strconv.ParseInt(photoIDStr, 10, 64)
	if err != nil {
		h.logger.Error("failed to parse photo ID", "error", err, "data", data)
		h.bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Ошибка обработки"))
		return
	}

	// Получаем фото из БД
	photo, err := h.photoRepo.GetByID(ctx, photoID)
	if err != nil {
		h.logger.Error("failed to get photo", "error", err, "photo_id", photoID)
		h.bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Фото не найдено"))
		return
	}

	// Удаляем файл с диска
	if err := os.Remove(photo.FilePath); err != nil {
		h.logger.Warn("failed to delete photo file", "error", err, "filepath", photo.FilePath)
	}

	// Удаляем фото из БД
	if err := h.photoRepo.Delete(ctx, photoID); err != nil {
		h.logger.Error("failed to delete photo from db", "error", err, "photo_id", photoID)
		h.bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Ошибка удаления"))
		return
	}

	// Отправляем уведомление пользователю
	if photo.TelegramUserID != nil {
		user, err := h.userRepo.GetByID(ctx, *photo.TelegramUserID)
		if err == nil {
			rejectionText := "❌ *Ваше фото отклонено*\n\n"
			rejectionText += "К сожалению, модератор не одобрил вашу фотографию.\n"
			rejectionText += "Возможные причины:\n"
			rejectionText += "• Неподходящий контент\n"
			rejectionText += "• Низкое качество изображения\n"
			rejectionText += "• Не относится к погоде\n\n"
			rejectionText += "Вы можете отправить другое фото."

			rejectionMsg := tgbotapi.NewMessage(user.ChatID, rejectionText)
			rejectionMsg.ParseMode = "Markdown"
			h.bot.Send(rejectionMsg)
		}
	}

	// Редактируем сообщение админа (убираем кнопки)
	editText := callback.Message.Caption + "\n\n❌ *Фото отклонено и удалено*"
	editMsg := tgbotapi.NewEditMessageCaption(callback.Message.Chat.ID, callback.Message.MessageID, editText)
	editMsg.ParseMode = "Markdown"
	h.bot.Send(editMsg)

	// Подтверждаем callback
	h.bot.Request(tgbotapi.NewCallback(callback.ID, "❌ Фото отклонено"))

	h.logger.Info("photo rejected and deleted", "photo_id", photoID, "admin_id", callback.Message.Chat.ID)
}

func (h *BotHandler) handleAnnounce(ctx context.Context, msg *tgbotapi.Message) {
	// 1. Проверка прав
	if !h.isAdmin(msg.Chat.ID) {
		h.sendMessage(msg.Chat.ID, "❌ У вас нет доступа к этой команде")
		return
	}

	// 2. Получение и валидация текста
	announceText := msg.CommandArguments()
	if announceText == "" {
		h.sendMessage(msg.Chat.ID, "❌ Укажите текст анонса после команды\n\nПример:\n/announce 🔥 Новая функция доступна!")
		return
	}

	if len(announceText) > 4096 {
		h.sendMessage(msg.Chat.ID, "❌ Текст анонса слишком длинный (максимум 4096 символов)")
		return
	}

	h.logger.Info("announcement requested",
		"admin_id", msg.Chat.ID,
		"text_length", len(announceText))

	// 3. Получение пользователей
	activeUsers, err := h.userRepo.GetAllActive(ctx)
	if err != nil {
		h.logger.Error("failed to get active users", "error", err)
		h.sendMessage(msg.Chat.ID, "❌ Ошибка получения списка пользователей")
		return
	}

	if len(activeUsers) == 0 {
		h.sendMessage(msg.Chat.ID, "⚠️ Нет активных пользователей для рассылки")
		return
	}

	// 4. Уведомление о начале
	startMsg := fmt.Sprintf("📨 Начинаю рассылку анонса...\n👥 Пользователей: %d", len(activeUsers))
	h.sendMessage(msg.Chat.ID, startMsg)

	// 5. Массовая рассылка
	successCount := 0
	errorCount := 0

	for _, user := range activeUsers {
		message := tgbotapi.NewMessage(user.ChatID, announceText)
		message.ParseMode = "Markdown"

		if _, err := h.bot.Send(message); err != nil {
			h.logger.Error("failed to send announcement",
				"chat_id", user.ChatID,
				"username", user.Username,
				"error", err)
			errorCount++

			// Отметить неактивным если бот заблокирован
			if strings.Contains(err.Error(), "bot was blocked") {
				h.userRepo.UpdateActivity(ctx, user.ChatID, false)
			}
		} else {
			h.logger.Debug("announcement sent", "chat_id", user.ChatID)
			successCount++
		}

		// Rate limiting
		time.Sleep(50 * time.Millisecond)
	}

	// 6. Отчёт
	reportText := fmt.Sprintf("✅ *Рассылка завершена!*\n\n"+
		"📊 *Статистика:*\n"+
		"• Успешно: %d\n"+
		"• Ошибки: %d\n"+
		"• Всего: %d\n",
		successCount, errorCount, len(activeUsers))

	if errorCount > 0 {
		reportText += "\n⚠️ Пользователи с ошибками могли заблокировать бота"
	}

	h.sendMessage(msg.Chat.ID, reportText)

	h.logger.Info("announcement completed",
		"total", len(activeUsers),
		"success", successCount,
		"errors", errorCount)
}

func (h *BotHandler) handleAnnouncePreview(ctx context.Context, msg *tgbotapi.Message) {
	// 1. Проверка прав
	if !h.isAdmin(msg.Chat.ID) {
		h.sendMessage(msg.Chat.ID, "❌ У вас нет доступа к этой команде")
		return
	}

	// 2. Получение и валидация текста
	announceText := msg.CommandArguments()
	if announceText == "" {
		h.sendMessage(msg.Chat.ID, "❌ Укажите текст анонса после команды\n\nПример:\n/announce_preview 🔥 Новая функция доступна!")
		return
	}

	if len(announceText) > 4096 {
		h.sendMessage(msg.Chat.ID, "❌ Текст анонса слишком длинный (максимум 4096 символов)")
		return
	}

	h.logger.Info("announcement preview requested",
		"admin_id", msg.Chat.ID,
		"text_length", len(announceText))

	// 3. Формируем превью с подсказкой
	previewHeader := "👀 *ПРЕДПРОСМОТР АНОНСА*\n"
	previewHeader += "━━━━━━━━━━━━━━━━━━━━\n\n"

	previewFooter := "\n\n━━━━━━━━━━━━━━━━━━━━\n"
	previewFooter += "💡 Для отправки всем пользователям используйте:\n"
	previewFooter += "`/announce " + announceText + "`"

	fullPreview := previewHeader + announceText + previewFooter

	// 4. Отправляем превью
	reply := tgbotapi.NewMessage(msg.Chat.ID, fullPreview)
	reply.ParseMode = "Markdown"

	if _, err := h.bot.Send(reply); err != nil {
		h.logger.Error("failed to send preview", "error", err)
		h.sendMessage(msg.Chat.ID, "❌ Ошибка отправки предпросмотра")
		return
	}

	h.logger.Info("announcement preview sent", "admin_id", msg.Chat.ID)
}
