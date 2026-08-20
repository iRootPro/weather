# Внутренние компоненты Go-приложения

**Последняя сверка:** 2026-08-20
**Источники истины:** `cmd/`, `internal/handler/`, `internal/service/`, `internal/repository/`, `internal/models/`, `internal/mqtt/`, `internal/telegram/`, `internal/maxbot/`, `pkg/`

## Основное направление зависимостей

```text
cmd (composition root)
  → transport handlers / workers
    → domain services
      → repository interfaces
        → pgx implementations
          → PostgreSQL / TimescaleDB
```

`cmd/*/main.go` загружает конфигурацию, создаёт pool и concrete dependencies, запускает циклы и обрабатывает shutdown. Бизнес-правила не должны добавляться в composition root.

| Изменение | Правильный слой |
|---|---|
| HTTP route, decode/encode, status code, template response | `internal/handler/api` или `internal/handler/web` |
| Расчёт события, агрегата, архива, dashboard view model | `internal/service` |
| SQL query, upsert, selection policy | `internal/repository` |
| Доменная структура и чистое вычисление | `internal/models` |
| Клиент стороннего HTTP/MQTT/TCP API | `pkg/<integration>` или transport package бота |
| Инициализация и lifecycle процесса | `cmd/<process>/main.go` |

## `api-server`

```mermaid
flowchart TB
    browser["Browser / TUI / API client"]
    mux["net/http ServeMux\nroutes and static files"]
    api_handlers["internal/handler/api\nJSON REST handlers"]
    web_handlers["internal/handler/web\nHTML pages and HTMX widgets"]
    templates["Go templates + static assets"]
    photos["photos directory"]

    dashboard["DashboardService"]
    weather["WeatherService\nWeatherArchiveService\nWeatherInsights"]
    enrich["ForecastService\nGeomagneticService\nHydroService"]
    astronomy["SunService / MoonService"]

    repos["Repository interfaces and pgx implementations"]
    db[("PostgreSQL / TimescaleDB")]
    geo["IP geolocation astronomy API"]

    browser -->|"HTTP"| mux
    mux --> api_handlers
    mux --> web_handlers
    mux -->|"file HTTP"| photos
    web_handlers --> templates
    api_handlers --> dashboard
    api_handlers --> weather
    api_handlers --> enrich
    web_handlers --> dashboard
    web_handlers --> weather
    web_handlers --> enrich
    web_handlers --> astronomy
    dashboard --> weather
    dashboard --> enrich
    weather --> repos
    enrich --> repos
    repos -->|"SQL via pgxpool"| db
    web_handlers -->|"metadata SQL"| repos
    web_handlers <-->|"files"| photos
    astronomy -->|"optional HTTPS; local fallback"| geo
```

### Transport layer

- `internal/handler/api` возвращает JSON для dashboard snapshot, weather, sensors и hydro endpoints.
- `internal/handler/web` рендерит полные страницы, detail pages и HTMX widgets. Он также координирует gallery/photo metadata и читает templates.
- `net/http.ServeMux` и route registration находятся непосредственно в `cmd/api-server/main.go`.

### Service layer

- `WeatherService` — текущие/исторические измерения, статистика, события и derived views.
- `WeatherArchiveService` и weather insights — агрегаты и narrative/архивные представления поверх weather repository.
- `DashboardService` композирует weather, forecast, geomagnetic и optional hydro services в snapshot.
- `ForecastService`, `GeomagneticService`, `HydroService` предоставляют доменные чтения своих таблиц.
- `SunService` и `MoonService` выполняют астрономические расчёты; MoonService может использовать внешний client и локальный fallback.

### Repository layer

`internal/repository/interfaces.go` задаёт contracts для weather, sensors, forecast, photos, Narodmon logs, geomagnetic, hydro, Telegram и Max. Concrete pgx repositories содержат SQL. Один `pgxpool.Pool` создаётся в каждом процессе отдельно; межпроцессного shared memory нет.

## MQTT ingestion

```mermaid
flowchart LR
    client["pkg/mqttclient\nconnection and subscription"]
    handler["internal/mqtt.Handler"]
    parser["internal/mqtt.Parser"]
    model["models.WeatherData"]
    repo["WeatherRepository.Save"]
    db[("weather_data")]

    client -->|"Paho MessageHandler"| handler
    handler --> parser
    parser -->|"unit conversion + derived values"| model
    handler --> repo
    repo --> db
```

Parser принимает URL-encoded или JSON payload, переводит имперские единицы EcoWitt в метрические, вычисляет dew point/feels-like и сохраняет отфильтрованный `raw_data`. Handler логирует parse/save errors и не останавливает subscription loop.

## Боты

```mermaid
flowchart LR
    api["Telegram or Max Bot API"]
    transport["bot client + update loop"]
    handler["command/callback handler"]
    services["weather / forecast / sun / moon / geomagnetic services"]
    botrepos["user / subscription / notification repositories"]
    notifier["Notifier + DailySummary background loops"]
    db[("PostgreSQL")]

    api <-->|"HTTPS updates/messages"| transport
    transport --> handler
    handler --> services
    handler --> botrepos
    notifier --> services
    notifier --> botrepos
    services --> db
    botrepos --> db
    notifier -->|"outgoing message"| transport
```

Telegram package дополнительно содержит photo/EXIF processing, charts, keyboards и richer command set. Max package использует собственный HTTP client и long polling marker. Таблицы пользователей и уведомлений каналов изолированы друг от друга.

## Осознанные отклонения и границы

- `internal/models` содержит не только DTO, но и чистые weather calculations; это допустимая доменная логика без I/O.
- `DashboardService` и web handler зависят от concrete service structs: композиция проста, но замены обычно требуют constructor changes.
- Bot packages совмещают transport, formatting и фоновые application loops; переносить эту логику в общий abstraction имеет смысл только при реальном третьем канале.
- `internal/apiclient.WeatherService` реализует API-backed access для TUI, но использует `repository.DailyMinMax` в return type. Это небольшой leak repository package в client boundary; новый код не должен расширять такую связь.
- External clients живут преимущественно в `pkg/`, но Telegram и Max clients находятся рядом с channel-specific handlers. Это текущая договорённость, а не основание создавать второй общий client framework.

См. также [потоки данных](04-data-flows.md) и [модель данных](05-data-model.md).
