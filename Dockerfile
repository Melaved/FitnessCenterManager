# syntax=docker/dockerfile:1

FROM golang:1.23 AS builder
WORKDIR /app

# Сначала зависимости для кеша
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Копируем остальной код и собираем бинарник
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /out/server ./cmd/web

FROM alpine:3.20
WORKDIR /app

COPY --from=builder /out/server /app/server
COPY web/ /app/web/

# Конфиг монтируется из хоста (см. docker-compose), но для локальной сборки можно скопировать дефолт
COPY config.yaml /app/config.yaml

EXPOSE 3000
ENTRYPOINT ["/app/server"]
