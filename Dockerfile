FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o /telegram-bot ./cmd/bot

FROM alpine:latest
# Add ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /telegram-bot .

# Copy .env file (if exists - will be overridden by volume mount)
COPY .env* ./

# Run the binary
CMD ["./telegram-bot"]
