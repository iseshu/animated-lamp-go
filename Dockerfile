# --- Stage 1: Build the Go binary ---
FROM golang:1.22-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /bin/screenshotbot \
    ./cmd/bot

# --- Stage 2: Final Ultra-Lightweight Container (<80MB total) ---
FROM alpine:3.19

RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata

WORKDIR /app

RUN mkdir -p /app/data

COPY --from=builder /bin/screenshotbot /app/screenshotbot

ENV DB_PATH=/app/data/bot.db
ENV MAX_WORKERS=2
ENV MAX_PROCESSES_PER_USER=1

VOLUME ["/app/data"]

ENTRYPOINT ["/app/screenshotbot"]
