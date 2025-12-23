# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Описание проекта

Система сбора и визуализации данных с метеостанции EcoWitt. Собирает данные о погоде через MQTT, хранит в PostgreSQL/TimescaleDB, предоставляет веб-интерфейс на HTMX.

## Команды сборки и запуска

```bash
# Сборка всех сервисов
go build -o bin/mqtt-consumer ./cmd/mqtt-consumer
go build -o bin/api-server ./cmd/api-server
go build -o bin/migrator ./cmd/migrator

# Запуск сервисов
./bin/mqtt-consumer
./bin/api-server

# Запуск миграций
./bin/migrator up

# Запуск тестов
go test ./...

# Запуск одного теста
go test -run TestName ./path/to/package
```

## Архитектура

Три основных сервиса в `cmd/`:

- **mqtt-consumer**: Подписывается на MQTT топики, обрабатывает сообщения метеостанции, сохраняет в БД
- **api-server**: HTTP сервер с REST API и HTMX веб-интерфейсом
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

Переиспользуемые пакеты в `pkg/`:

- `database/` - Утилиты подключения к БД
- `logger/` - Настройка логирования
- `mqttclient/` - Обёртка MQTT клиента

## Технологический стек

- Go 1.25+ со стандартной библиотекой для роутинга
- PostgreSQL + TimescaleDB для хранения временных рядов
- goose для миграций БД
- Eclipse Paho для MQTT
- HTMX + Tailwind CSS + Chart.js для фронтенда
- slog для логирования
