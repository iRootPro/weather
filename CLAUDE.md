# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Язык общения

**ВАЖНО**: Все ответы должны быть на русском языке. Весь текст коммуникации с пользователем должен быть на русском.

## Описание проекта

Система сбора и визуализации данных с метеостанции EcoWitt. Собирает данные о погоде через MQTT, хранит в PostgreSQL/TimescaleDB, предоставляет веб-интерфейс на HTMX.

## Команды сборки и запуска

```bash
# Сборка всех сервисов
go build -o bin/mqtt-consumer ./cmd/mqtt-consumer
go build -o bin/api-server ./cmd/api-server
go build -o bin/migrator ./cmd/migrator
go build -o bin/weather-tui ./cmd/weather-tui

# Запуск сервисов
./bin/mqtt-consumer
./bin/api-server
./bin/weather-tui  # Terminal UI для просмотра погоды

# Запуск миграций
./bin/migrator up

# Запуск тестов
go test ./...

# Запуск одного теста
go test -run TestName ./path/to/package
```

## Архитектура

Основные сервисы в `cmd/`:

- **mqtt-consumer**: Подписывается на MQTT топики, обрабатывает сообщения метеостанции, сохраняет в БД
- **api-server**: HTTP сервер с REST API и HTMX веб-интерфейсом
- **weather-tui**: Terminal UI (TUI) приложение для просмотра погоды в терминале
- **migrator**: Утилита миграций базы данных

Внутренние пакеты в `internal/`:

- `config/` - Конфигурация приложения (переменные окружения)
- `mqtt/` - MQTT клиент, обработчики и парсеры сообщений
- `repository/` - Слой доступа к БД (weather, sensor)
- `service/` - Бизнес-логика
- `handler/api/` - REST API обработчики
- `handler/web/` - HTMX обработчики для веб-интерфейса
- `models/` - Структуры данных
- `web/templates/` - HTML шаблоны с HTMX partials
- `web/static/` - CSS, JS, статические файлы
- `tui/` - Terminal UI компоненты (Bubble Tea)
  - `components/` - Компоненты TUI (dashboard, charts, events)

Переиспользуемые пакеты в `pkg/`:

- `database/` - Утилиты подключения к БД
- `logger/` - Настройка логирования
- `mqttclient/` - Обёртка MQTT клиента

## Технологический стек

- Go 1.26+ со стандартной библиотекой для роутинга
- PostgreSQL + TimescaleDB для хранения временных рядов
- goose для миграций БД
- Eclipse Paho для MQTT
- HTMX + Tailwind CSS + Chart.js для веб-интерфейса
- Bubble Tea + Lipgloss + asciigraph для Terminal UI
- slog для логирования
