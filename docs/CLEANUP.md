# Docker Cleanup Commands

Команды для очистки Docker на production сервере.

## Доступные команды

### `make deploy-clean`
**Безопасная очистка** - удаляет неиспользуемые образы и build cache.

```bash
make deploy-clean
```

**Что удаляет:**
- ✅ Неиспользуемые Docker образы (dangling и unused)
- ✅ Build cache
- ✅ Показывает статистику до и после

**Освобождает:** ~1-3 GB

**Безопасно:** Да - не трогает работающие контейнеры и volumes

---

### `make deploy-clean-logs`
**Очистка логов** - удаляет все логи контейнеров.

```bash
make deploy-clean-logs
```

**Что удаляет:**
- ✅ Все логи контейнеров (`*-json.log`)

**Освобождает:** Зависит от размера логов (может быть несколько GB)

**Предупреждение:** ⚠️ Потеряете все логи! Сделайте backup если нужно.

---

### `make deploy-clean-all`
**Полная очистка** - комбинация двух предыдущих команд.

```bash
make deploy-clean-all
```

**Что удаляет:**
- ✅ Неиспользуемые образы
- ✅ Build cache
- ✅ Все логи контейнеров

**Освобождает:** ~2-5 GB

---

## Рекомендации

### Регулярная очистка

Запускайте `make deploy-clean` **раз в месяц** или после нескольких деплоев:

```bash
# После деплоя
make deploy
make deploy-clean
```

### Экстренная очистка

Если диск заполнен >90%:

```bash
# 1. Полная очистка
make deploy-clean-all

# 2. Проверить результат
make deploy-status
docker system df
```

### Предотвращение переполнения

Добавьте ротацию логов в `docker-compose.yml`:

```yaml
services:
  your-service:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

Это ограничит логи каждого сервиса до **30MB** (3 файла по 10MB).

---

## Статистика

### Использование Docker

Проверить использование диска:

```bash
# Локально
docker system df

# На production
ssh root@weather.example.com 'docker system df'

# Или через make
make deploy-status
```

### Размер базы данных

Проверить размер PostgreSQL:

```bash
make deploy-db-size
```

**Показывает:**
- Общий размер базы данных
- Топ-10 самых больших таблиц с размерами
- Количество записей в основных таблицах (weather_data, forecasts, photos)

**Пример вывода:**
```
=== Размер базы данных ===
 db_size
---------
 2.5 GB

=== Размер таблиц (топ-10) ===
 schemaname | tablename    | size
------------+--------------+---------
 public     | weather_data | 2.1 GB
 public     | forecasts    | 300 MB
 public     | photos       | 50 MB

=== Количество записей ===
 table_name   | rows
--------------+--------
 weather_data | 1250000
 forecasts    | 50000
 photos       | 150
```

---

## Troubleshooting

### "No space left on device"

```bash
# 1. Полная очистка
make deploy-clean-all

# 2. Проверить диск
ssh root@weather.example.com 'df -h'

# 3. Если все еще мало места - проверить базу данных
make deploy-check
```

### Большая база данных

Если PostgreSQL занимает много места, настройте retention:

```sql
-- Хранить данные только 1 год
SELECT add_retention_policy('weather_data', INTERVAL '365 days');

-- Удалить старые данные
DELETE FROM weather_data WHERE time < NOW() - INTERVAL '365 days';
VACUUM FULL weather_data;
```

---

## Безопасность

**Что НЕ удаляется:**
- ❌ Работающие контейнеры
- ❌ Docker volumes (postgres_data, photos_data)
- ❌ Образы используемые контейнерами (если они запущены)

**Что удаляется:**
- ✅ Старые версии образов
- ✅ Dangling images (без тега)
- ✅ Build cache
- ✅ Логи (только с `deploy-clean-logs`)
