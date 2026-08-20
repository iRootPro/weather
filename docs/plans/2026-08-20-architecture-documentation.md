# Architecture Documentation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Создать актуальный комплект архитектурной документации для разработчиков и эксплуатации, который словами и схемами описывает компоненты погодного сервиса, потоки данных, deployment и эксплуатацию.

**Architecture:** Документация строится как лёгкий C4-комплект в `docs/architecture/`. Схемы хранятся в Mermaid внутри Markdown и сверяются с исполняемыми точками входа, Docker Compose, конфигурацией, миграциями и фактическими зависимостями кода. `doc/arc.md` после переноса полезных фактов удаляется, чтобы остался один источник истины.

**Tech Stack:** Markdown, Mermaid, Go, PostgreSQL/TimescaleDB, Docker Compose, HTMX.

---

## Общие правила готовности

- Описывать только подтверждённое кодом и конфигурацией текущее состояние.
- Для каждого компонента указывать назначение, входы, выходы, зависимости, хранилище и способ запуска.
- На каждой схеме подписывать протокол или тип взаимодействия: MQTT, HTTP, SQL, Bot API, файловый volume.
- Не копировать реализации функций и полный SQL; документировать границы и контракты.
- Все относительные ссылки между документами должны открываться из GitHub.
- Каждая Mermaid-схема должна иметь соседнее текстовое объяснение.
- В заголовке каждой страницы указывать источники истины и дату последней сверки.

### Task 1: Провести инвентаризацию исполняемых компонентов

**Files:**
- Create: `docs/architecture/README.md`
- Reference: `cmd/*/main.go`
- Reference: `docker-compose.yml`
- Reference: `docker-compose.prod.yml`
- Reference: `Dockerfile`

**Steps:**
1. Составить полный список бинарников из `cmd/`: `api-server`, `mqtt-consumer`, `migrator`, `forecast-fetcher`, `narodmon-sender`, `geomagnetic-fetcher`, `hydro-fetcher`, `telegram-bot`, `max-bot`, `weather-tui`.
2. Сверить, какие бинарники собираются Dockerfile и какие запускаются local/prod Compose.
3. Для каждого компонента зафиксировать назначение, режим запуска и статус: production service, local tool или CLI.
4. Создать индекс комплекта с навигацией по восьми архитектурным страницам.
5. Отдельно перечислить известные границы документации: без построчного API reference и без полного справочника переменных окружения.

**Acceptance:** В индексе нет компонентов, отсутствующих в коде; каждый каталог `cmd/*` учтён ровно один раз.

### Task 2: Описать системный контекст

**Files:**
- Create: `docs/architecture/01-system-context.md`
- Reference: `cmd/*/main.go`
- Reference: `pkg/*/client.go`
- Reference: `internal/telegram/`
- Reference: `internal/maxbot/`

**Steps:**
1. Определить акторов: посетитель веб-интерфейса, пользователь Telegram, пользователь Max и оператор системы.
2. Определить внешние системы: EcoWitt/MQTT broker, Open-Meteo, Narodmon, XRAS, Emercom/hydrology sources и bot APIs.
3. Построить Mermaid context diagram с погодным сервисом как одной границей.
4. Словами описать ценность системы, её ответственность и то, что остаётся вне её контроля.
5. Указать направления передачи данных и протоколы.

**Acceptance:** По странице понятно, откуда система получает данные, где их публикует и кто ими пользуется, без знания структуры репозитория.

### Task 3: Описать контейнеры и процессы

**Files:**
- Create: `docs/architecture/02-containers.md`
- Reference: `docker-compose.yml`
- Reference: `docker-compose.prod.yml`
- Reference: `Dockerfile`
- Reference: `cmd/*/main.go`

**Steps:**
1. Построить C4 Level 2 diagram в Mermaid для TimescaleDB и всех долгоживущих процессов.
2. Отделить HTTP/HTMX serving, ingestion, scheduled fetching, outbound publishing и messaging.
3. Для каждого процесса добавить таблицу: ответственность, входы, выходы, БД/volume, startup dependencies.
4. Отметить, какие процессы отсутствуют в local Compose, но присутствуют в production Compose.
5. Зафиксировать роль migrator и порядок запуска относительно PostgreSQL и остальных сервисов.

**Acceptance:** Схема совпадает с production Compose; все сервисы и volumes объяснены текстом.

### Task 4: Описать внутренние компоненты Go-приложения

**Files:**
- Create: `docs/architecture/03-components.md`
- Reference: `internal/handler/`
- Reference: `internal/service/`
- Reference: `internal/repository/`
- Reference: `internal/models/`
- Reference: `internal/mqtt/`
- Reference: `internal/telegram/`
- Reference: `internal/maxbot/`
- Reference: `pkg/`

**Steps:**
1. Описать основной слой зависимостей: `cmd → handler → service → repository → PostgreSQL`.
2. Построить component diagram для `api-server`, разделив web/HTMX handlers и REST API handlers.
3. Описать назначение models, repositories, domain services и внешних clients.
4. Отдельно показать структуру MQTT ingestion и bot processing.
5. Указать допустимое направление зависимостей и места, где текущий код от него отличается.

**Acceptance:** Новый разработчик может определить правильный слой для HTTP handler, бизнес-правила, SQL query и внешнего API client.

### Task 5: Описать основные потоки данных

**Files:**
- Create: `docs/architecture/04-data-flows.md`
- Reference: `internal/mqtt/`
- Reference: `internal/handler/`
- Reference: `internal/service/`
- Reference: `internal/repository/`
- Reference: `internal/telegram/`
- Reference: `internal/maxbot/`

**Steps:**
1. Создать sequence diagram приёма телеметрии: EcoWitt → MQTT → consumer → repository → TimescaleDB.
2. Создать sequence diagram чтения dashboard/archive: browser → API server → service → repository → TimescaleDB → HTML/HTMX.
3. Описать обновление forecast, geomagnetic и hydro данных внешними fetchers.
4. Описать генерацию уведомления и ежедневной сводки для Telegram и Max.
5. Описать отправку текущих данных в Narodmon.
6. Для каждого потока указать синхронные границы, фоновые интервалы и точку сохранения данных.

**Acceptance:** Документ покрывает ingestion, чтение, enrichment и outbound delivery; каждый поток связан с конкретными компонентами из container diagram.

### Task 6: Описать логическую модель данных

**Files:**
- Create: `docs/architecture/05-data-model.md`
- Reference: `migrations/*.sql`
- Reference: `internal/models/`
- Reference: `internal/repository/`

**Steps:**
1. Сгруппировать таблицы по доменам: weather/sensors, forecast, photos, Telegram, Max, geomagnetic, hydro и Narodmon.
2. Построить упрощённую Mermaid ER diagram только с ключевыми сущностями и связями.
3. Для каждого домена указать владельца записи и основных читателей.
4. Описать роль TimescaleDB, временных колонок, индексов и политики выбора временных записей.
5. Указать, какие данные хранятся в PostgreSQL, а какие — в `photos_data` volume.

**Acceptance:** Все миграции `001`–`010` отражены в доменной карте; диаграмма не превращается в копию SQL schema.

### Task 7: Описать внешние интеграции

**Files:**
- Create: `docs/architecture/06-integrations.md`
- Reference: `pkg/openmeteo/`
- Reference: `pkg/narodmon/`
- Reference: `pkg/xras/`
- Reference: `pkg/emercit/`
- Reference: `pkg/ipgeolocation/`
- Reference: `pkg/mqttclient/`
- Reference: `internal/maxbot/client.go`
- Reference: `internal/telegram/bot.go`

**Steps:**
1. Создать таблицу внешних систем с направлением вызова, протоколом, аутентификацией и потребителем внутри проекта.
2. Для каждой интеграции описать используемые данные и частоту обращения.
3. Зафиксировать timeout/retry/fallback поведение только там, где оно реализовано.
4. Указать влияние недоступности внешней системы на остальные компоненты.
5. Не публиковать токены, реальные секреты и чувствительные URL.

**Acceptance:** Для каждой директории интеграционного клиента указан реальный вызывающий компонент и ожидаемое поведение при ошибке.

### Task 8: Описать deployment и конфигурацию

**Files:**
- Create: `docs/architecture/07-deployment.md`
- Reference: `docker-compose.yml`
- Reference: `docker-compose.prod.yml`
- Reference: `Dockerfile`
- Reference: `.env.example`
- Reference: `internal/config/config.go`
- Reference: `Makefile`
- Reference: `DEPLOY.md`

**Steps:**
1. Построить deployment diagram для production host, containers, ports и volumes.
2. Описать различия local и production Compose.
3. Описать startup order: PostgreSQL healthcheck → migrator → application services.
4. Сгруппировать конфигурацию по компонентам без копирования секретов и значений production.
5. Описать сборку multi-stage Docker targets и команды Makefile для запуска/deploy.
6. Зафиксировать persistent data и требования к backup.

**Acceptance:** Оператор понимает, что запускается, в каком порядке, где лежат данные и какие внешние порты открыты.

### Task 9: Создать эксплуатационный runbook

**Files:**
- Create: `docs/architecture/08-operations.md`
- Reference: `docker-compose.prod.yml`
- Reference: `internal/config/config.go`
- Reference: `pkg/logger/`
- Reference: `docs/CLEANUP.md`
- Reference: `DEPLOY.md`

**Steps:**
1. Описать health signals, readiness dependencies и формат логирования.
2. Создать таблицу фоновых процессов: расписание/интервал, источник данных, целевая таблица, симптомы сбоя.
3. Описать диагностику типовых отказов: PostgreSQL, MQTT, внешний fetcher, bot API, photos volume.
4. Описать безопасный restart и восстановление после миграционной ошибки.
5. Добавить короткие команды проверки состояния через Docker Compose и существующий Makefile.
6. Сослаться на backup/deploy/cleanup инструкции вместо дублирования процедур.

**Acceptance:** Для каждого production container есть наблюдаемый признак работы и первый диагностический шаг.

### Task 10: Провести сквозную сверку и удалить устаревшую архитектуру

**Files:**
- Modify: `docs/architecture/*.md`
- Remove: `doc/arc.md`

**Steps:**
1. Сверить список бинарников с `cmd/`, список containers с production Compose и домены данных со всеми миграциями.
2. Проверить единообразие названий компонентов на всех схемах.
3. Проверить относительные Markdown-ссылки и Mermaid syntax.
4. Убедиться, что документация не содержит секретов и неподтверждённых планов.
5. Перенести из `doc/arc.md` только всё ещё верные факты, отсутствующие в новом комплекте.
6. Удалить `doc/arc.md` как устаревший параллельный источник истины.
7. Добавить в `docs/architecture/README.md` правило обновления документации при изменении `cmd/`, Compose, migrations или внешних integrations.
8. Выполнить `git diff --check` и просмотреть итоговый комплект как читатель.

**Acceptance:** В репозитории остаётся один актуальный комплект архитектурной документации; все схемы и списки подтверждены кодом и deployment-конфигурацией.
