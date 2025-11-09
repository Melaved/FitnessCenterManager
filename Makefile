SHELL := /bin/sh

# Variables
IMAGE ?= fitness-center-manager:local

.PHONY: run build test tidy fmt vet docker-build docker-up docker-down docker-logs docker-restart

run:
	go run ./cmd/web

build:
	go build -o bin/server ./cmd/web

test:
	go test ./...

tidy:
	go mod tidy

vet:
	go vet ./...

fmt:
	go fmt ./...

docker-build:
	docker build -t $(IMAGE) .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f web

docker-restart:
	docker compose restart web

