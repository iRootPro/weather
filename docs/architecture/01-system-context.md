# Системный контекст

**Последняя сверка:** 2026-08-20  
**Источники истины:** `cmd/*/main.go`, `pkg/*/client.go`, `internal/telegram/`, `internal/maxbot/`

## Назначение системы

Погодный сервис собирает локальную телеметрию метеостанции EcoWitt, дополняет её внешними прогнозами, геомагнитными и гидрологическими данными, хранит временные ряды и предоставляет их через веб-интерфейс, REST API, терминальный клиент и ботов.

Система отвечает за приём, нормализацию, хранение, агрегацию и публикацию данных. Она не управляет самой метеостанцией, MQTT broker, внешними API и сетями мессенджеров.

## C4 Level 1

```mermaid
flowchart LR
    station["EcoWitt station"]
    broker["MQTT broker"]
    visitor["Посетитель сайта"]
    telegram_user["Пользователь Telegram"]
    max_user["Пользователь Max"]
    operator["Оператор"]
    tui_user["Пользователь TUI"]

    weather["Погодный сервис\nСбор, хранение, анализ и публикация данных"]

    openmeteo["Open-Meteo"]
    xras["XRAS"]
    emercit["Источник уровней воды МЧС"]
    narodmon["Narodmon"]
    telegram_api["Telegram Bot API"]
    max_api["Max Bot API"]
    geo_api["IP geolocation API"]

    station -->|"MQTT telemetry"| broker
    broker -->|"MQTT subscription"| weather
    openmeteo -->|"HTTPS forecast"| weather
    xras -->|"HTTPS geomagnetic data"| weather
    emercit -->|"HTTPS hydrology data"| weather
    weather -->|"TCP payload"| narodmon

    visitor <-->|"HTTP / HTML / HTMX"| weather
    tui_user <-->|"HTTP / JSON"| weather
    operator -->|"Docker Compose / logs / config"| weather

    telegram_user <-->|"messages and callbacks"| telegram_api
    telegram_api <-->|"HTTPS Bot API"| weather
    max_user <-->|"messages and callbacks"| max_api
    max_api <-->|"HTTPS Bot API"| weather
    weather -->|"HTTPS lookup"| geo_api
```

Диаграмма показывает логическую границу всего репозитория. Внутри неё находятся несколько independently deployed Go-процессов и общая TimescaleDB; они раскрыты в [контейнерной архитектуре](02-containers.md).

## Акторы

| Актор | Что получает или делает |
|---|---|
| Посетитель сайта | Смотрит текущую погоду, прогноз, историю, архив, рекорды, аналитику, геомагнитные и гидрологические данные; загружает HTMX-фрагменты через browser. |
| Пользователь Telegram | Запрашивает погоду и статистику, управляет подписками, получает события и ежедневные сводки; может работать с фотофункциями бота. |
| Пользователь Max | Запрашивает текущую погоду, управляет подписками, получает события и ежедневные сводки. |
| Пользователь TUI | Читает текущие данные и представления через REST API из терминального клиента. |
| Оператор | Настраивает переменные окружения, выполняет deployment и миграции, наблюдает логи и восстанавливает сервисы. |

## Внешние системы

| Система | Направление относительно сервиса | Назначение |
|---|---|---|
| EcoWitt station и MQTT broker | Входящее | Основной поток локальных погодных измерений. Broker остаётся внешней инфраструктурой. |
| Open-Meteo | Входящее | Прогноз погоды для заданных координат. |
| XRAS | Входящее | Геомагнитный прогноз и фактические значения Kp. |
| Источник МЧС | Входящее | Уровни воды гидрологических постов. |
| IP geolocation API | Исходящее чтение | Географический контекст по IP для отдельных пользовательских сценариев. |
| Telegram Bot API | Двунаправленное | Long polling/updates, команды, callback, сообщения и фотографии. |
| Max Bot API | Двунаправленное | Long polling, команды, callback и исходящие сообщения. |
| Narodmon | Исходящее | Публикация последних измерений станции. |

## Границы ответственности

Сервис гарантирует обработку данных после их получения и фиксирует ошибки интеграций в логах. Он не гарантирует:

- доставку телеметрии до внешнего MQTT broker;
- доступность, полноту и точность сторонних прогнозов;
- доставку сообщения после принятия его API мессенджера;
- доступность сети между production host и внешними системами;
- хранение данных вне PostgreSQL volumes и `photos_data` volume.

Следующие уровни: [контейнеры и процессы](02-containers.md), [интеграции](06-integrations.md), [эксплуатация](08-operations.md).
