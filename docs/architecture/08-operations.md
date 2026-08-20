# Эксплуатационный runbook

**Последняя сверка:** 2026-08-20
**Источники истины:** `docker-compose.prod.yml`, `cmd/*/main.go`, `internal/config/config.go`, `pkg/logger/`, `Makefile`, `DEPLOY.md`, `docs/CLEANUP.md`

## Быстрая проверка

```bash
make deploy-status
make deploy-check  # использует default DB user/name weather/weather
make deploy-logs
```

На production host:

```bash
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs --tail=100 api-server mqtt-consumer
docker compose -f docker-compose.prod.yml exec postgres sh -c 'pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
docker compose -f docker-compose.prod.yml exec api-server sh -c 'wget -qO- "http://127.0.0.1:${HTTP_PORT:-8080}/health"'
```

`postgres` имеет formal Docker healthcheck, `api-server` — HTTP `/health`. Endpoint API подтверждает только работу HTTP process и не проверяет DB; поэтому его нужно сочетать с `pg_isready` и freshness queries. Остальные application containers не имеют Compose healthcheck: их состояние определяется process status, logs и свежестью принадлежащих им данных.

## Сигналы здоровья по компонентам

| Container | Признак нормальной работы | Первый диагностический шаг |
|---|---|---|
| `weather-postgres` | `healthy`, `pg_isready` success | `docker compose ... logs postgres`; проверить volume и disk space |
| `weather-migrator` | Завершился code 0 до старта приложений | `docker compose ... logs migrator`; затем `/app/migrator status` |
| `weather-api-server` | `/health` возвращает 2xx; страницы отвечают | Проверить API logs, DB connectivity, templates/static paths |
| `weather-mqtt-consumer` | Логи `weather data saved`; растёт `MAX(weather_data.time)` | Проверить broker reachability, topic, credentials и reconnect logs |
| `weather-forecast-fetcher` | Периодический `forecast fetched and saved successfully` | Проверить Open-Meteo error и `MAX(fetched_at)` в `forecast_data` |
| `weather-narodmon-sender` | Новые строки `narodmon_logs` | Проверить TCP reachability, device config и последний `error_message` |
| `weather-geomagnetic-fetcher` | `geomagnetic fetched and saved` | Проверить source/proxy и `MAX(fetched_at)` в geomagnetic tables |
| `weather-hydro-fetcher` | `hydro fetched and saved` | Проверить enable flag, auth/station IDs и `MAX(fetched_at)` |
| `weather-telegram-bot` | Успешная bot initialization и ongoing updates | Проверить token, Bot API network, process logs и DB connection |
| `weather-max-bot` | `max bot authorized`; polling без постоянных ошибок | Проверить token, `GetMe`, network и 5-second retry loop errors |

## Фоновые процессы

Значения ниже — defaults из `internal/config/config.go`; production `.env` может их переопределить.

| Процесс | Default | Источник | Target | Симптом сбоя |
|---|---:|---|---|---|
| MQTT subscription | Непрерывно | MQTT broker | `weather_data` | Время последней телеметрии не меняется |
| Forecast fetch | 3600 s | Open-Meteo | `forecast_data` | Старый `fetched_at`, прогноз истёк |
| Geomagnetic fetch | 10800 s | XRAS | `geomagnetic_kp`, `geomagnetic_daily` | Старые Kp slots/daily data |
| Hydro fetch | 600 s | Emercom | `hydro_gauges`, `hydro_level_readings` | Старый `fetched_at`/`observed_at` |
| Narodmon send | 300 s | `weather_data` | Narodmon + `narodmon_logs` | Нет новых audit rows или `success=false` |
| Telegram notifier | 300 s | Weather events/subscriptions | Telegram + notification rows | Нет alerts при наличии event; send errors |
| Max notifier | 300 s | Weather events/subscriptions | Max + notification rows | Poll/send errors, нет dedup history |
| Daily summaries | 07:00 local | Weather/forecast/astronomy | Telegram и Max | Нет сообщения после scheduled window |

`narodmon-sender`, `geomagnetic-fetcher` и `hydro-fetcher` завершаются с code 0, если соответствующий `*_ENABLED=false`. При `restart: unless-stopped` это может выглядеть как restart loop. Production Compose рассчитан на включённые workers; для намеренного отключения нужно также остановить/не запускать соответствующий service, а не только менять flag.

## Диагностические SQL-запросы

Выполнять через `psql` внутри PostgreSQL container с production credentials.

```sql
SELECT COUNT(*) AS total, MAX(time) AS last_weather FROM weather_data;
SELECT MAX(fetched_at) AS last_forecast_fetch FROM forecast_data;
SELECT MAX(fetched_at) AS last_geomagnetic_fetch FROM geomagnetic_kp;
SELECT MAX(fetched_at) AS last_hydro_fetch FROM hydro_level_readings;
SELECT sent_at, success, sensors_count, error_message
FROM narodmon_logs ORDER BY sent_at DESC LIMIT 10;
```

Интерпретировать freshness относительно configured interval, а не жёстких defaults. `observed_at` описывает время источника, `fetched_at` — время загрузки.

## Типовые отказы

### PostgreSQL недоступен

1. Проверить `docker compose ... ps postgres` и `pg_isready`.
2. Прочитать postgres logs; проверить свободное место и состояние `postgres_data`.
3. Не перезапускать все containers до понимания причины: application processes обычно завершаются при initial DB connection failure.
4. После восстановления DB запустить failed migrator, затем `docker compose ... up -d`.
5. Проверить `/health`, свежесть `weather_data` и worker logs.

### MQTT telemetry устарела

1. Сравнить текущее время и `MAX(weather_data.time)`.
2. Проверить consumer logs на `connection lost`, `reconnecting`, parse/save errors.
3. Проверить доступность `MQTT_HOST:MQTT_PORT`, topic и credentials.
4. Перезапустить только consumer, если client reconnect не восстановился:

```bash
docker compose -f docker-compose.prod.yml restart mqtt-consumer
```

Перезапуск не восполняет сообщения, которые broker не сохранил для persistent session.

### External fetcher не обновляется

1. Проверить, включена ли интеграция и какой interval задан в `.env`.
2. Прочитать logs конкретного fetcher: HTTP status, timeout, parse/auth error.
3. Для XRAS проверить optional proxy; для Emercom — credentials и station UUID; для Open-Meteo — coordinates/timezone.
4. Если исправлена только transient-проблема, выполнить `restart` нужного worker. Если изменён `.env`, пересоздать container через `up -d --force-recreate` по инструкции ниже. Worker делает fetch сразу при старте.
5. Проверить `MAX(fetched_at)` и count новых rows.

Старые успешно сохранённые данные остаются доступны; outage fetcher не требует остановки API.

### Bot не отвечает

1. Проверить process status и initial authorization logs.
2. Проверить наличие token без вывода его значения.
3. Проверить outbound HTTPS/DNS до Bot API.
4. Проверить DB connectivity и состояние user/subscription tables.
5. Если конфигурация не менялась, перезапустить только нужный bot. После изменения token или другого значения `.env` пересоздать container. Для Max постоянные `GetUpdates` errors должны сопровождаться 5-second retry; при пустом token process находится в disabled state.

### Фото не открываются или не загружаются

1. Проверить mount `photos_data` у `api-server` и `telegram-bot`.
2. Проверить disk usage, permissions и наличие файла по `photos.file_path`.
3. Не удалять строку или файл до backup: metadata и binary должны оставаться согласованными.
4. После восстановления volume проверить `/photos/` и gallery.

## Restart и применение конфигурации

При неизменной конфигурации перезапускайте только затронутый process:

```bash
docker compose -f docker-compose.prod.yml restart <service>
docker compose -f docker-compose.prod.yml logs -f --tail=100 <service>
```

`restart` не перечитывает `.env` и не пересоздаёт container. После изменения `.env`, image или Compose definition используйте:

```bash
docker compose -f docker-compose.prod.yml up -d --force-recreate <service>
docker compose -f docker-compose.prod.yml logs -f --tail=100 <service>
```

Не выполнять `down -v`: ключ `-v` удалит persistent volumes. Полный `down` также создаёт ненужный outage и не является первым средством диагностики.

## Ошибка миграции

1. Не запускать application containers против частично обновлённой schema.
2. Сохранить migrator logs и проверить status:

```bash
docker compose -f docker-compose.prod.yml run --rm migrator /app/migrator status
```

3. Сделать DB backup перед ручным исправлением или rollback.
4. Исправить migration/code; затем повторить:

```bash
docker compose -f docker-compose.prod.yml run --rm migrator /app/migrator up
docker compose -f docker-compose.prod.yml up -d
```

5. Проверить startup dependencies, API health и данные. `down` migrations могут удалять таблицы/колонки; применять их без отдельного recovery plan нельзя.

## Backup, restore и cleanup

- DB backup/restore команды: [DEPLOY.md](../../DEPLOY.md).
- Docker image/build-cache cleanup: [docs/CLEANUP.md](../CLEANUP.md) и `make deploy-clean`.
- `make deploy-clean-logs` необратимо обнуляет Docker JSON logs; сначала сохранить нужные incident logs.
- `make deploy-clean-all` объединяет image/cache и log cleanup; это не routine health action.
- Фото требуют отдельного backup `photos_data`; DB dump их не содержит.

После restore проверять не только row counts, но и запуск migrator status, `/health`, свежесть workers, bot authorization и выборочные photo files.
