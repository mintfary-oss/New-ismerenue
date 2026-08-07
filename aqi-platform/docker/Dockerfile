# ──────────────────────────────────────────────────────────────
# Stage 1: сборка Go-бинарника
# ──────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS go-builder

# Устанавливаем git (нужен для go mod download с VCS)
RUN apk add --no-cache git ca-certificates tzdata

# Используем китайский/российский прокси для Go-модулей (proxy.golang.org недоступен из РФ)
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
ENV GONOSUMDB=*

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
# Stage 2: минимальный runtime-образ на базе Alpine
# Alpine — стандартный Docker Hub образ, не требует внешних реестров.
# Создаём непривилегированного пользователя для безопасности.
# Размер финального образа: ~15–20 МБ
# ──────────────────────────────────────────────────────────────
FROM alpine:3.20

# Устанавливаем минимальные зависимости + CA-сертификаты для HTTPS-вызовов
RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

# Метаданные образа
LABEL org.opencontainers.image.title="AQI Platform" \
      org.opencontainers.image.description="Платформа прогнозирования качества атмосферного воздуха" \
      org.opencontainers.image.licenses="MIT"

WORKDIR /app

# Копируем бинарник и статические файлы
COPY --from=go-builder /build/aqi-platform .
COPY --from=go-builder /build/migrations ./migrations

# Устанавливаем корректные права
RUN chown -R appuser:appgroup /app

# Порт приложения
EXPOSE 8080

# Запуск от непривилегированного пользователя
USER appuser:appgroup

# Команда по умолчанию — запуск сервера
ENTRYPOINT ["/app/aqi-platform"]
CMD ["server"]
