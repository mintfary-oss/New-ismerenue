# ──────────────────────────────────────────────────────────────
# Stage 1: сборка Go-бинарника
# ──────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS go-builder

# Устанавливаем git (нужен для go mod download с VCS)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Кешируем зависимости отдельным слоем
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Копируем весь исходный код
COPY . .

# Собираем статически слинкованный бинарник
# CGO_ENABLED=0 — без C-зависимостей
# -ldflags="-w -s" — без debug-символов (уменьшает размер)
# -trimpath — убирает локальные пути из бинарника
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

RUN CGO_ENABLED=0 GOOS=linux \
    go build \
    -ldflags="-w -s \
      -X main.version=${VERSION} \
      -X main.buildTime=${BUILD_TIME} \
      -X main.gitCommit=${GIT_COMMIT}" \
    -trimpath \
    -o /build/aqi-platform \
    ./cmd/aqi-platform/

# ──────────────────────────────────────────────────────────────
# Stage 2: минимальный runtime-образ
# Используем distroless/static — нет shell, нет пакетных менеджеров
# Размер финального образа: ~12–15 МБ
# ──────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

# Метаданные образа
LABEL org.opencontainers.image.title="AQI Platform" \
      org.opencontainers.image.description="Платформа прогнозирования качества атмосферного воздуха" \
      org.opencontainers.image.source="https://github.com/mintfary/aqi-platform" \
      org.opencontainers.image.licenses="MIT"

WORKDIR /app

# Копируем бинарник и статические файлы
COPY --from=go-builder /build/aqi-platform .
COPY --from=go-builder /build/migrations ./migrations
COPY --from=go-builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Порт приложения
EXPOSE 8080

# Запуск от непривилегированного пользователя (nonroot UID=65532)
USER nonroot:nonroot

# Команда по умолчанию — запуск сервера
ENTRYPOINT ["/app/aqi-platform"]
CMD ["server"]
