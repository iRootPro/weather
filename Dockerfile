# Этап сборки
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарники
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/mqtt-consumer ./cmd/mqtt-consumer
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api-server ./cmd/api-server
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/migrator ./cmd/migrator
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/telegram-bot ./cmd/telegram-bot
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/forecast-fetcher ./cmd/forecast-fetcher
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/narodmon-sender ./cmd/narodmon-sender

# MQTT Consumer
FROM alpine:3.20 AS mqtt-consumer
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/mqtt-consumer /app/mqtt-consumer
CMD ["/app/mqtt-consumer"]

# API Server
FROM alpine:3.20 AS api-server
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/api-server /app/api-server
COPY --from=builder /app/internal/web/templates /app/templates
COPY --from=builder /app/internal/web/static /app/static
EXPOSE 8080
CMD ["/app/api-server"]

# Migrator
FROM alpine:3.20 AS migrator
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/migrator /app/migrator
COPY --from=builder /app/migrations /app/migrations
CMD ["/app/migrator", "up"]

# Telegram Bot
FROM alpine:3.20 AS telegram-bot
RUN apk --no-cache add ca-certificates tzdata exiftool python3 py3-pip
RUN pip3 install pillow-heif --break-system-packages
WORKDIR /app
COPY --from=builder /bin/telegram-bot /app/telegram-bot
COPY scripts/convert_heic.py /app/convert_heic.py
RUN chmod +x /app/convert_heic.py
CMD ["/app/telegram-bot"]

# Forecast Fetcher
FROM alpine:3.20 AS forecast-fetcher
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/forecast-fetcher /app/forecast-fetcher
CMD ["/app/forecast-fetcher"]

# Narodmon Sender
FROM alpine:3.20 AS narodmon-sender
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/narodmon-sender /app/narodmon-sender
CMD ["/app/narodmon-sender"]
