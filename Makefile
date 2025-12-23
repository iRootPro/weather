.PHONY: build run-consumer run-api test lint migrate-up migrate-down docker-up docker-down tidy

# Сборка
build:
	go build -o bin/mqtt-consumer ./cmd/mqtt-consumer
	go build -o bin/api-server ./cmd/api-server
	go build -o bin/migrator ./cmd/migrator

build-consumer:
	go build -o bin/mqtt-consumer ./cmd/mqtt-consumer

build-api:
	go build -o bin/api-server ./cmd/api-server

build-migrator:
	go build -o bin/migrator ./cmd/migrator

# Запуск
run-consumer:
	go run ./cmd/mqtt-consumer

run-api:
	go run ./cmd/api-server

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
