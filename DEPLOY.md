# Деплой Weather на Proxmox

## Требования
- Docker + Docker Compose
- Git

## Установка

### 1. Клонируем репозиторий
```bash
git clone https://github.com/iRootPro/weather.git
cd weather
```

### 2. Создаём .env файл
```bash
cp .env.example .env
nano .env
```

**Обязательно измени:**
- `DB_PASSWORD` — надёжный пароль для БД
- `MQTT_HOST` — адрес твоего MQTT брокера
- `MQTT_USERNAME` / `MQTT_PASSWORD` — если требуется авторизация

### 3. Запускаем
```bash
docker-compose -f docker-compose.prod.yml up -d
```

Это запустит:
1. **postgres** — TimescaleDB для хранения данных
2. **migrator** — применит миграции (один раз)
3. **mqtt-consumer** — начнёт собирать данные с метеостанции

### 4. Проверяем
```bash
# Логи consumer
docker logs -f weather-mqtt-consumer

# Проверка данных в БД
docker exec weather-postgres psql -U weather -d weather -c \
  "SELECT time, temp_outdoor, humidity_outdoor, pressure_relative FROM weather_data ORDER BY time DESC LIMIT 5;"
```

## Управление

```bash
# Остановить
docker-compose -f docker-compose.prod.yml down

# Перезапустить
docker-compose -f docker-compose.prod.yml restart mqtt-consumer

# Обновить (после git pull)
docker-compose -f docker-compose.prod.yml up -d --build

# Логи
docker-compose -f docker-compose.prod.yml logs -f
```

## Бэкап данных

```bash
# Бэкап БД
docker exec weather-postgres pg_dump -U weather weather > backup_$(date +%Y%m%d).sql

# Восстановление
cat backup_YYYYMMDD.sql | docker exec -i weather-postgres psql -U weather weather
```
