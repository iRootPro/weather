.PHONY: build run-consumer run-api run-tui test lint migrate-up migrate-down docker-up docker-down tidy deploy deploy-logs deploy-status deploy-stop deploy-init

# Сборка
build:
	go build -o bin/mqtt-consumer ./cmd/mqtt-consumer
	go build -o bin/api-server ./cmd/api-server
	go build -o bin/migrator ./cmd/migrator
	go build -o bin/weather-tui ./cmd/weather-tui

build-consumer:
	go build -o bin/mqtt-consumer ./cmd/mqtt-consumer

build-api:
	go build -o bin/api-server ./cmd/api-server

build-migrator:
	go build -o bin/migrator ./cmd/migrator

build-tui:
	go build -o bin/weather-tui ./cmd/weather-tui

# Запуск
run-consumer:
	go run ./cmd/mqtt-consumer

run-api:
	go run ./cmd/api-server

run-tui:
	go run ./cmd/weather-tui

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
