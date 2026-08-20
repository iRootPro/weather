# Основные потоки данных

**Последняя сверка:** 2026-08-20  
**Источники истины:** `cmd/*/main.go`, `internal/mqtt/`, `internal/handler/`, `internal/service/`, `internal/repository/`, `internal/telegram/`, `internal/maxbot/`

## 1. Приём телеметрии EcoWitt

```mermaid
sequenceDiagram
    participant S as EcoWitt station
    participant B as MQTT broker
    participant C as mqtt-consumer
    participant P as MQTT Parser
    participant R as WeatherRepository
    participant DB as TimescaleDB

    S->>B: MQTT telemetry payload
    B->>C: Message on configured topic
    C->>P: Parse URL-encoded or JSON payload
    P-->>C: WeatherData in metric units
    C->>R: Save(context, WeatherData)
    R->>DB: INSERT weather_data
    DB-->>R: Result
    R-->>C: Success or error
```

Поток асинхронен относительно пользователей. Время записи назначается parser при обработке сообщения. Parse/save error логируется для конкретного сообщения; MQTT process продолжает работу. Точка durable persistence — `weather_data` hypertable.

## 2. Чтение dashboard и архива

```mermaid
sequenceDiagram
    participant U as Browser
    participant H as api-server handler
    participant S as Domain service
    participant R as Repository
    participant DB as TimescaleDB
    participant T as Go template

    U->>H: GET page, widget or JSON endpoint
    H->>S: Request current/history/archive view
    S->>R: Latest, range, aggregate or insight query
    R->>DB: SQL / TimescaleDB aggregate
    DB-->>R: Rows
    R-->>S: Domain models
    S-->>H: View model or API model
    alt HTML or HTMX
        H->>T: Execute template
        T-->>H: HTML
        H-->>U: Full page or fragment
    else REST API
        H-->>U: JSON
    end
```

`api-server` выполняет запрос синхронно. Dashboard snapshot может объединять weather, forecast, geomagnetic и hydro reads. Архив выбирает заданный период/разрешение и рассчитывает события поверх доступных станционных данных. Точка чтения — committed rows PostgreSQL; HTTP response не кэшируется отдельным process-local store.

## 3. Обогащение внешними данными

```mermaid
sequenceDiagram
    participant W as Fetcher process
    participant API as External API
    participant R as Domain repository
    participant DB as TimescaleDB

    Note over W: Run immediately after startup
    W->>API: HTTPS request with configured timeout
    API-->>W: Forecast / Kp / hydro payload
    W->>W: Parse, normalize, classify
    W->>R: Batch save or upsert
    R->>DB: SQL transaction/upserts
    DB-->>R: Result
    R-->>W: Success
    W->>W: Wait configured update interval
```

- `forecast-fetcher` загружает hourly/daily Open-Meteo forecast и сохраняет ограниченные конфигурацией горизонты.
- `geomagnetic-fetcher` преобразует XRAS данные в трёхчасовые Kp slots и daily solar activity; после записи удаляет данные старше 90 дней.
- `hydro-fetcher`, если включён, сохраняет metadata гидропостов, actual reading и доступную историю; retention задаётся конфигурацией.

Первый fetch выполняется сразу после подключения к БД. Ошибка внешнего запроса логируется, затем процесс остаётся жив и ждёт следующего tick. Последние успешно сохранённые данные продолжают читаться API и ботами.

## 4. Команда и уведомления ботов

```mermaid
sequenceDiagram
    participant U as Telegram or Max user
    participant BA as Bot API
    participant B as Bot process
    participant H as Command handler
    participant S as Domain services
    participant R as Bot repositories
    participant DB as PostgreSQL

    U->>BA: Command or callback
    BA-->>B: Update via long polling
    B->>H: Handle update
    H->>R: Upsert user / change subscriptions
    R->>DB: SQL
    H->>S: Read weather/forecast if required
    S->>DB: Repository queries
    S-->>H: Domain data
    H->>BA: Send formatted response
    BA-->>U: Message
```

При старте bot process параллельно запускает:

- update loop для команд и callback;
- notifier с конфигурируемым интервалом;
- daily summary scheduler с конфигурируемым временем.

Notifier читает последние погодные события, получает active subscribers, проверяет таблицу notifications для дедупликации, отправляет сообщение и фиксирует факт отправки. Telegram и Max используют отдельные user/subscription/notification tables. Daily summary читает weather, forecast/astronomy/geomagnetic данные, но не создаёт отдельной materialized summary table.

## 5. Публикация в Narodmon

```mermaid
sequenceDiagram
    participant N as narodmon-sender
    participant WR as WeatherRepository
    participant DB as PostgreSQL
    participant NC as Narodmon TCP client
    participant NR as NarodmonLogRepository
    participant EXT as Narodmon

    N->>WR: GetLatest
    WR->>DB: SELECT latest weather_data
    DB-->>WR: WeatherData
    WR-->>N: Latest measurement
    N->>N: Build configured sensor payload
    N->>NC: SendData
    NC->>EXT: TCP payload
    EXT-->>NC: Protocol result
    N->>NR: Save success/error and sensor count
    NR->>DB: INSERT narodmon_logs
```

Первая отправка выполняется при старте, затем — по `Narodmon.Interval`. Если измерений нет, process логирует отсутствие данных и ждёт следующего цикла. История результата отправок доступна веб-виджету через `narodmon_logs`.

## Синхронные и асинхронные границы

| Поток | Граница | Durable point | Поведение при ошибке |
|---|---|---|---|
| MQTT ingestion | MQTT callback | `weather_data` insert | Сообщение пропускается, process продолжает subscription |
| HTTP/HTMX read | Один HTTP request | Уже committed DB rows | Handler возвращает ошибку; фоновые процессы не затрагиваются |
| External fetch | Один fetch tick | Batch save/upsert | Сохраняются предыдущие данные; retry на следующем tick |
| Bot command | Один incoming update | User/subscription update | Ошибка логируется/возвращается пользователю; polling продолжается |
| Bot notifier | Один notification cycle | Notification dedup row | Неуспешный cycle повторится согласно notifier logic/interval |
| Narodmon | Один send tick | `narodmon_logs` row | Результат логируется; следующая попытка на следующем tick |

Связанные страницы: [контейнеры](02-containers.md), [компоненты](03-components.md), [интеграции](06-integrations.md).
