# Внешние интеграции

**Последняя сверка:** 2026-08-20  
**Источники истины:** `pkg/openmeteo/`, `pkg/narodmon/`, `pkg/xras/`, `pkg/emercit/`, `pkg/ipgeolocation/`, `pkg/mqttclient/`, `internal/telegram/`, `internal/maxbot/`, `internal/config/config.go`

## Карта интеграций

| Система | Направление | Протокол и auth | Клиент | Потребитель | Частота |
|---|---|---|---|---|---|
| MQTT broker / EcoWitt | Входящее | MQTT over TCP; optional username/password | Eclipse Paho wrapper `pkg/mqttclient` | `mqtt-consumer` | Непрерывная subscription |
| Open-Meteo | Входящее | HTTPS GET; без auth | `pkg/openmeteo` | `forecast-fetcher` | При старте и каждые `FORECAST_UPDATE_INTERVAL` |
| XRAS | Входящее | HTTPS GET; без auth; optional proxy | `pkg/xras` | `geomagnetic-fetcher` | При старте и каждые `GEOMAGNETIC_UPDATE_INTERVAL` |
| Emercom public service | Входящее | HTTPS; JWT bearer после login | `pkg/emercit` | `hydro-fetcher` | При старте и каждые `HYDRO_UPDATE_INTERVAL` |
| IPGeolocation astronomy | Входящее optional | HTTPS GET; API key query parameter | `pkg/ipgeolocation` | Moon service в API и Telegram | По пользовательскому запросу/формированию данных; local fallback без key |
| Telegram Bot API | Двунаправленное | HTTPS; bot token | Telegram Go SDK | `telegram-bot` | Long polling + исходящие сообщения |
| Max Bot API | Двунаправленное | HTTPS JSON; Authorization token | `internal/maxbot.Client` | `max-bot` | Long polling + исходящие сообщения |
| Narodmon | Исходящее | Собственный text protocol over TCP; device MAC/name | `pkg/narodmon` | `narodmon-sender` | При старте и каждые `NARODMON_INTERVAL` |

## MQTT

Client включает `AutoReconnect` и `ConnectRetry`: начальный retry interval 5 секунд, максимальный reconnect interval 1 минута, keepalive 30 секунд, ping timeout 10 секунд, persistent session (`CleanSession=false`). Consumer подписывается на конфигурируемый topic и не подтверждает запись через отдельную application queue: обработчик Paho непосредственно парсит сообщение и выполняет SQL insert.

Недоступность broker не затрагивает API и ботов, но свежесть `weather_data` перестаёт обновляться. После восстановления client сам переподключается; оператор проверяет сообщения `connection lost`, `reconnecting` и `weather data saved`.

## Open-Meteo

Client формирует один forecast request для координат и timezone станции, запрашивает hourly и daily наборы. HTTP timeout задаётся `FORECAST_API_TIMEOUT`; встроенного retry в client нет. Worker повторяет полный fetch на следующем interval. Сохранённые ранее forecast rows остаются доступны, а записи старше 7 дней очищаются после успешной batch save.

## XRAS

Client читает JSON Kp/solar activity с конфигурируемого URL. Данные источника нормализуются из строк, timezone источника разбирается отдельно. Поддерживается optional HTTPS proxy. HTTP timeout задаётся конфигурацией; request-level retry нет. Worker повторит запрос на следующем tick, а предыдущие строки продолжат обслуживать dashboard и alerts. Retention — 90 дней.

## Emercom / уровни воды

Client получает JWT. Если credentials не заданы, он может загрузить preset public credentials источника. Для `/api/actual/` при 401/403 выполняется одна повторная авторизация и один retry. Запрос истории использует текущий bearer token без отдельного повторного login внутри метода.

`hydro-fetcher` различает primary и upstream stations: отсутствие primary station считается ошибкой цикла, проблемы upstream station логируются и пропускаются. После восстановления следующий interval повторяет полную загрузку actual/history.

## IPGeolocation astronomy

API key и timeout задаются конфигурацией. Client передаёт координаты и optional date. Если key отсутствует, `api-server` и `telegram-bot` не создают client, а MoonService использует локальные расчёты. Поэтому недоступность интеграции ухудшает точность/полноту astronomy response, но не должна делать сервис погоды недоступным.

## Telegram Bot API

`telegram-bot` использует официальный-compatible Go SDK и token из окружения. Update loop работает через long polling. Process также отправляет сообщения, callback responses, изображения и файлы; photo path/metadata сохраняются локально. Отдельные notifier и daily summary loops используют тот же client.

Пустой `TELEGRAM_TOKEN` завершает process с fatal error. Ошибки отдельных updates или отправок логируются в channel package; доступность остальных containers от Telegram не зависит.

## Max Bot API

Custom client поддерживает profile, updates с marker, отправку сообщений пользователю и callback answers. Общий HTTP timeout равен long-poll timeout плюс запас. При ошибке `GetUpdates` polling loop ждёт 5 секунд и повторяет запрос; panic обработки одного update перехватывается, чтобы loop продолжился.

Если token пуст, container остаётся запущенным в disabled state и ждёт shutdown signal. Ошибка initial authorization (`GetMe`) завершает process.

## Narodmon

Client открывает TCP connection с timeout, формирует text payload с device identity и configured sensors, читает protocol response. Request-level retry отсутствует. `narodmon-sender` делает следующую попытку по interval и сохраняет success/error, sensor count и текст ошибки в `narodmon_logs`. `NARODMON_ENABLED` управляет публикацией и web status service.

## Секреты и диагностика

Секреты хранятся только в `.env` на deployment host: DB/MQTT credentials, bot tokens, astronomy key и optional hydro credentials. `.env` не входит в Git. В документацию и логи нельзя добавлять значения token/password. Для диагностики достаточно status code, integration name, duration и error class; response body может содержать внешние данные и требует осторожности.

См. [потоки данных](04-data-flows.md), [deployment](07-deployment.md) и [operations](08-operations.md).
