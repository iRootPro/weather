.PHONY: build build-consumer build-api build-migrator build-tui build-bot build-forecast run-consumer run-api run-tui run-bot run-forecast test lint migrate-up migrate-down docker-up docker-down tidy deploy deploy-logs deploy-status deploy-stop deploy-init deploy-check deploy-db-size deploy-clean deploy-clean-logs deploy-clean-all

# Сборка
build:
	go build -o bin/mqtt-consumer ./cmd/mqtt-consumer
	go build -o bin/api-server ./cmd/api-server
	go build -o bin/migrator ./cmd/migrator
	go build -o bin/weather-tui ./cmd/weather-tui
	go build -o bin/telegram-bot ./cmd/telegram-bot
	go build -o bin/forecast-fetcher ./cmd/forecast-fetcher

build-consumer:
	go build -o bin/mqtt-consumer ./cmd/mqtt-consumer

build-api:
	go build -o bin/api-server ./cmd/api-server

build-migrator:
	go build -o bin/migrator ./cmd/migrator

build-tui:
	go build -o bin/weather-tui ./cmd/weather-tui

build-bot:
	go build -o bin/telegram-bot ./cmd/telegram-bot

build-forecast:
	go build -o bin/forecast-fetcher ./cmd/forecast-fetcher

# Запуск
run-consumer:
	go run ./cmd/mqtt-consumer

run-api:
	go run ./cmd/api-server

run-tui:
	go run ./cmd/weather-tui

run-bot:
	go run ./cmd/telegram-bot

run-forecast:
	go run ./cmd/forecast-fetcher

# Тесты
test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Миграции
migrate-up:
	go run ./cmd/migrator up

migrate-down:
	go run ./cmd/migrator down

migrate-status:
	go run ./cmd/migrator status

# Docker
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Зависимости
tidy:
	go mod tidy

# Линтер
lint:
	golangci-lint run ./...

# ============================================
# Деплой на удалённый сервер
# ============================================

# Загружаем конфигурацию деплоя
-include deploy.conf
export

SSH_CMD := ssh -p $(or $(DEPLOY_PORT),22) $(DEPLOY_USER)@$(DEPLOY_HOST)

# Полный деплой (git pull + rebuild + restart)
deploy:
	@chmod +x scripts/deploy.sh
	@./scripts/deploy.sh

# Первоначальная настройка сервера
deploy-init:
	@echo "=== Первоначальная настройка сервера ==="
	$(SSH_CMD) " \
		apt-get update && apt-get install -y docker.io docker-compose git && \
		systemctl enable docker && \
		systemctl start docker && \
		echo 'Docker установлен' \
	"

# Логи с сервера
deploy-logs:
	$(SSH_CMD) "cd $(DEPLOY_PATH) && docker compose -f docker-compose.prod.yml logs -f --tail=100"

# Статус контейнеров
deploy-status:
	$(SSH_CMD) "cd $(DEPLOY_PATH) && docker compose -f docker-compose.prod.yml ps"

# Остановить сервисы
deploy-stop:
	$(SSH_CMD) "cd $(DEPLOY_PATH) && docker compose -f docker-compose.prod.yml down"

# Перезапустить consumer
deploy-restart:
	$(SSH_CMD) "cd $(DEPLOY_PATH) && docker compose -f docker-compose.prod.yml restart mqtt-consumer"

# Проверить данные в БД
deploy-check:
	$(SSH_CMD) "cd $(DEPLOY_PATH) && docker exec weather-postgres psql -U weather -d weather -c 'SELECT COUNT(*) as total, MAX(time) as last_update FROM weather_data;'"

# Проверить размер базы данных
deploy-db-size:
	@echo "=== Размер базы данных ==="
	$(SSH_CMD) "docker exec weather-postgres psql -U weather -d weather -c \"SELECT pg_size_pretty(pg_database_size('weather')) as db_size;\""

# Очистка Docker (удаление неиспользуемых образов и кеша)
deploy-clean:
	@echo "=== Статистика Docker ДО очистки ==="
	$(SSH_CMD) "docker system df"
	@echo ""
	@echo "=== Удаление неиспользуемых образов ==="
	$(SSH_CMD) "docker image prune -a -f"
	@echo ""
	@echo "=== Удаление build cache ==="
	$(SSH_CMD) "docker builder prune -a -f"
	@echo ""
	@echo "=== Статистика Docker ПОСЛЕ очистки ==="
	$(SSH_CMD) "docker system df"
	@echo ""
	@echo "✅ Очистка завершена!"

# Очистка Docker логов
deploy-clean-logs:
	@echo "=== Очистка логов контейнеров ==="
	$(SSH_CMD) "truncate -s 0 /var/lib/docker/containers/*/*-json.log && echo 'Логи очищены'"

# Полная очистка (образы + кеш + логи)
deploy-clean-all: deploy-clean deploy-clean-logs
	@echo "✅ Полная очистка завершена!"
