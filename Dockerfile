FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o /telegram-bot ./cmd/bot

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /telegram-bot .

# Копируем конфигурационные файлы
COPY --from=builder /app/configs ./configs/

# Копируем .env файл (будет переопределен volume mount при необходимости)
COPY .env* ./

CMD ["./telegram-bot"]