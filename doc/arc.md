# Техническое задание: Система сбора и визуализации данных метеостанции EcoWitt

## Общее описание системы

### Цель проекта

Создание системы для сбора, хранения и визуализации данных с погодной станции EcoWitt через MQTT брокер.

#### Основные функции

Прием данных с метеостанции через MQTT протокол

- Хранение данных в базе данных

- Предоставление REST API для доступа к данным

- Веб-интерфейс для отображения текущих и исторических данных

- Визуализация данных в виде графиков и таблиц

#### Целевая архитектура

┌─────────────────┐     MQTT     ┌─────────────┐     HTTP     ┌─────────────┐
│   EcoWitt       │─────────────►│   MQTT      │◄─────────────│   MQTT      │
│   Метеостанция  │              │   Брокер    │              │   Consumer  │
└─────────────────┘              └─────────────┘              └──────┬──────┘
                                                                      │
                                                                      ▼
                                                               ┌─────────────┐     HTTP     ┌─────────────┐
                                                               │   База      │◄─────────────│   API       │
                                                               │   Данных    │              │   Server    │
                                                               └─────────────┘              └──────┬──────┘
                                                                                                   │
                                                                                                   ▼
                                                               ┌─────────────────────────────────────────┐
                                                               │          Веб-интерфейс (HTMX)           │
                                                               │          Пользовательский интерфейс     │
                                                               └─────────────────────────────────────────┘

### Технологический стек

#### Backend

Язык программирования: Go 1.26

Фреймворки и библиотеки:

Маршрутизация: Чистый golang

MQTT клиент: Eclipse Paho MQTT (paho.mqtt.golang)

База данных: PostgreSQL + TimescaleDB

Миграции БД: goose

Конфигурация: env10, Viper или cleanenv

Логирование: slog

Валидация: go-playground/validator

#### Frontend

Основная технология: HTMX

Дополнительные библиотеки:

Графики: Chart.js или Apache ECharts

Стили: Tailwind CSS

Взаимодействие: Hyperscript или Alpine.js (опционально)

Иконки: Font Awesome или Heroicons

#### Инфраструктура

Контейнеризация: Docker + Docker Compose

База данных: PostgreSQL/TimescaleDB или SQLite

MQTT брокер: Mosquitto (уже настроен)

Reverse proxy: Nginx (для production)

Мониторинг: Prometheus + Grafana (опционально)

### Детальная архитектура

Структура проекта

```

weather/
├── cmd/
│   ├── mqtt-consumer/          # Основной сервис MQTT консьюмера
│   │   └── main.go
│   ├── api-server/             # HTTP API сервер
│   │   └── main.go
│   └── migrator/               # Утилита для миграций БД
│       └── main.go
├── internal/
│   ├── config/                 # Конфигурация приложения
│   │   └── config.go
│   ├── mqtt/                   # MQTT клиент и обработчики
│   │   ├── client.go
│   │   ├── handler.go
│   │   └── parser.go
│   ├── repository/             # Работа с базой данных
│   │   ├── weather_repository.go
│   │   ├── sensor_repository.go
│   │   └── interfaces.go
│   ├── service/                # Бизнес-логика
│   │   ├── weather_service.go
│   │   └── sensor_service.go
│   ├── handler/                # HTTP обработчики
│   │   ├── api/               # REST API handlers
│   │   │   ├── weather_handler.go
│   │   │   └── sensor_handler.go
│   │   └── web/               # HTMX handlers
│   │       ├── dashboard_handler.go
│   │       ├── chart_handler.go
│   │       └── widget_handler.go
│   ├── models/                 # Структуры данных
│   │   ├── weather.go
│   │   ├── sensor.go
│   │   └── mqtt_message.go
│   └── web/                    # Веб-ресурсы
│       ├── templates/          # HTML шаблоны
│       │   ├── base.html
│       │   ├── dashboard.html
│       │   ├── history.html
│       │   └── partials/       # HTMX partials
│       │       ├── current_weather.html
│       │       ├── daily_stats.html
│       │       └── chart.html
│       └── static/             # Статические файлы
│           ├── css/
│           │   └── styles.css
│           ├── js/
│           │   └── charts.js
│           └── favicon.ico
├── migrations/                  # SQL миграции
│   ├── 001_initial.up.sql
│   └── 001_initial.down.sql
├── pkg/                        # Переиспользуемые пакеты
│   ├── database/
│   ├── logger/
│   └── mqttclient/
├── scripts/                    # Вспомогательные скрипты
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

### Компоненты системы

#### MQTT Consumer Service

Назначение: Подписка на MQTT топики, обработка сообщений, сохранение в БД.

Требования:

- Конфигурация через переменные окружения

- Поддержка реконнектов к брокеру

- Обработка ошибок с retry логикой

- Graceful shutdown

- Логирование операций
