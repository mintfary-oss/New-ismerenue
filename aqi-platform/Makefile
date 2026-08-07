# AQI Platform Makefile
# Использование: make <target>
#   make build    — собрать бинарник
#   make run      — запустить сервер локально
#   make docker   — собрать Docker-образ
#   make up       — запустить через Docker Compose
#   make down     — остановить Docker Compose
#   make test     — запустить тесты
#   make lint     — проверить код линтером
#   make migrate  — применить миграции

BINARY     := aqi-platform
MODULE     := github.com/mintfary/aqi-platform
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS    := -w -s \
              -X main.version=$(VERSION) \
              -X main.buildTime=$(BUILD_TIME) \
              -X main.gitCommit=$(GIT_COMMIT)

DOCKER_IMAGE  := ghcr.io/mintfary/aqi-platform
COMPOSE_FILE  := docker/docker-compose.yml
COMPOSE_FLAGS := -f $(COMPOSE_FILE) --env-file .env

.PHONY: all build run clean docker docker-push up down logs test lint fmt vet migrate sqlc help

all: lint test build

# ── Сборка ─────────────────────────────────────────────────────────────────
build:
	@echo "→ Сборка $(BINARY) $(VERSION)"
	CGO_ENABLED=0 go build \
		-ldflags="$(LDFLAGS)" \
		-trimpath \
		-o bin/$(BINARY) \
		./cmd/$(BINARY)/

build-linux-amd64:
	@echo "→ Кросс-компиляция linux/amd64"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="$(LDFLAGS)" \
		-trimpath \
		-o bin/$(BINARY)-linux-amd64 \
		./cmd/$(BINARY)/

build-linux-arm64:
	@echo "→ Кросс-компиляция linux/arm64"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
		-ldflags="$(LDFLAGS)" \
		-trimpath \
		-o bin/$(BINARY)-linux-arm64 \
		./cmd/$(BINARY)/

run: build
	@echo "→ Запуск сервера"
	./bin/$(BINARY) server --config config.yaml

clean:
	@rm -rf bin/
	@go clean -testcache

# ── Docker ─────────────────────────────────────────────────────────────────
docker:
	@echo "→ Сборка Docker-образа $(DOCKER_IMAGE):$(VERSION)"
	docker build \
		--file docker/Dockerfile \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--tag $(DOCKER_IMAGE):$(VERSION) \
		--tag $(DOCKER_IMAGE):latest \
		.

docker-push:
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

# ── Docker Compose ─────────────────────────────────────────────────────────
up:
	@echo "→ Запуск через Docker Compose"
	docker compose $(COMPOSE_FLAGS) up -d

down:
	docker compose $(COMPOSE_FLAGS) down

logs:
	docker compose $(COMPOSE_FLAGS) logs -f app

restart:
	docker compose $(COMPOSE_FLAGS) restart app

status:
	docker compose $(COMPOSE_FLAGS) ps

# ── База данных ────────────────────────────────────────────────────────────
migrate:
	@echo "→ Применение миграций"
	./bin/$(BINARY) migrate --direction up

migrate-down:
	./bin/$(BINARY) migrate --direction down

# ── Генерация кода ─────────────────────────────────────────────────────────
sqlc:
	@echo "→ Генерация кода из SQL (sqlc)"
	sqlc generate

# ── Качество кода ──────────────────────────────────────────────────────────
test:
	@echo "→ Запуск тестов"
	go test -v -race -count=1 -timeout=60s ./...

test-coverage:
	go test -v -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "→ Отчёт: coverage.html"

lint:
	@echo "→ Линтер golangci-lint"
	golangci-lint run ./...

fmt:
	@echo "→ Форматирование кода"
	gofmt -w -s .
	goimports -w .

vet:
	go vet ./...

# ── Зависимости ────────────────────────────────────────────────────────────
deps:
	go mod download
	go mod verify
	go mod tidy

# ── Помощь ─────────────────────────────────────────────────────────────────
help:
	@echo "AQI Platform — доступные команды:"
	@echo ""
	@echo "  make build       — собрать бинарник"
	@echo "  make run         — запустить локально"
	@echo "  make docker      — собрать Docker-образ"
	@echo "  make up          — запустить Docker Compose"
	@echo "  make down        — остановить Docker Compose"
	@echo "  make logs        — логи приложения"
	@echo "  make test        — запустить тесты"
	@echo "  make lint        — запустить линтер"
	@echo "  make migrate     — применить миграции"
	@echo "  make sqlc        — сгенерировать SQL-код"
	@echo "  make help        — эта справка"
