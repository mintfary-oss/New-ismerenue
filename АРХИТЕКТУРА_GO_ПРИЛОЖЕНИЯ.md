# АРХИТЕКТУРА GO-ПРИЛОЖЕНИЯ
## AQI Platform — Полное техническое задание на разработку
### 100% оригинальный код на Go 1.23+ | Self-hosted | Docker | Linux

---

## 1. КОНЦЕПЦИЯ

**Название проекта:** `aqi-platform`  
**Суть:** Единый бинарный файл на Go, упакованный в Docker, разворачивающийся одной командой на любом Linux-сервере без интернета.

**Три режима работы:**
1. `aqi-platform server` — основной сервер (API + dashboard + widget)
2. `aqi-platform agent` — агент сбора данных с датчиков (можно на Raspberry Pi)
3. `aqi-platform migrate` — миграции БД (запускается автоматически при старте)

---

## 2. СТРУКТУРА ПРОЕКТА (ПОЛНАЯ)

```
aqi-platform/
│
├── cmd/
│   └── aqi-platform/
│       └── main.go                    # Точка входа, cobra CLI
│
├── internal/
│   │
│   ├── config/
│   │   ├── config.go                  # Структуры конфигурации
│   │   └── loader.go                  # Viper loader (ENV + yaml)
│   │
│   ├── server/
│   │   ├── server.go                  # HTTP сервер + graceful shutdown
│   │   └── router.go                  # Chi роутер, все маршруты
│   │
│   ├── middleware/
│   │   ├── auth.go                    # JWT validation middleware
│   │   ├── rbac.go                    # Role-based access control
│   │   ├── ratelimit.go               # Rate limiting (token bucket)
│   │   ├── security.go                # Security headers (CSP, HSTS, etc.)
│   │   ├── cors.go                    # CORS policy
│   │   ├── logger.go                  # Structured request logging
│   │   └── recovery.go                # Panic recovery
│   │
│   ├── handler/
│   │   ├── auth.go                    # POST /api/v1/auth/*
│   │   ├── user.go                    # CRUD /api/v1/users/*
│   │   ├── sensor.go                  # CRUD /api/v1/sensors/*
│   │   ├── measurement.go             # GET  /api/v1/measurements/*
│   │   ├── forecast.go                # GET  /api/v1/forecast/*
│   │   ├── token.go                   # CRUD /api/v1/tokens/*
│   │   ├── ingest.go                  # POST /api/v1/ingest/*
│   │   ├── widget.go                  # GET  /widget/* (публичный)
│   │   ├── health.go                  # GET  /health, /ready
│   │   └── feedback.go                # POST /api/v1/feedback
│   │
│   ├── service/
│   │   ├── auth_service.go            # Логика авторизации
│   │   ├── user_service.go            # Управление пользователями
│   │   ├── sensor_service.go          # Управление датчиками
│   │   ├── measurement_service.go     # Работа с измерениями
│   │   ├── forecast_service.go        # Оркестрация прогнозов
│   │   ├── token_service.go           # API-токены
│   │   └── report_service.go          # Генерация отчётов
│   │
│   ├── repository/
│   │   ├── postgres.go                # Пул соединений pgx
│   │   ├── redis.go                   # Redis клиент
│   │   ├── queries/                   # SQL файлы (читает sqlc)
│   │   │   ├── users.sql
│   │   │   ├── sensors.sql
│   │   │   ├── measurements.sql
│   │   │   ├── forecasts.sql
│   │   │   ├── tokens.sql
│   │   │   └── audit.sql
│   │   └── generated/                 # Авто-генерация sqlc (не редактировать!)
│   │       ├── db.go
│   │       ├── models.go
│   │       └── *.go
│   │
│   ├── domain/
│   │   ├── user.go                    # User entity + roles
│   │   ├── sensor.go                  # Sensor entity
│   │   ├── measurement.go             # Measurement entity
│   │   ├── forecast.go                # Forecast entity
│   │   ├── aqi.go                     # AQI calculation (российская методология)
│   │   └── errors.go                  # Domain errors
│   │
│   ├── forecast/
│   │   ├── engine.go                  # Прогнозный движок
│   │   ├── interpolation.go           # Пространственная интерполяция IDW
│   │   ├── timeseries.go              # Анализ временных рядов (EWMA)
│   │   ├── wind.go                    # Ветровые поля (расчёт)
│   │   └── cams_client.go             # ОПЦИОНАЛЬНО: клиент Copernicus CAMS API
│   │
│   └── ingest/
│       ├── mqtt_agent.go              # MQTT subscriber (датчики)
│       ├── email_agent.go             # IMAP email-мониторинг
│       ├── http_ingest.go             # REST POST для сторонних систем
│       └── validator.go               # Валидация входящих данных
│
├── web/
│   ├── dashboard/                     # Закрытая платформа (TypeScript + Vite)
│   │   ├── src/
│   │   │   ├── main.ts
│   │   │   ├── auth.ts
│   │   │   ├── map.ts                 # MapLibre GL JS карта
│   │   │   ├── charts.ts              # Chart.js графики
│   │   │   ├── forecast.ts
│   │   │   └── settings.ts
│   │   ├── index.html
│   │   ├── package.json
│   │   └── vite.config.ts
│   │
│   └── widget/                        # Публичный виджет (TypeScript + Vite)
│       ├── src/
│       │   ├── main.ts
│       │   ├── map.ts
│       │   ├── timeline.ts            # Анимированный таймлайн
│       │   ├── wind.ts                # Анимация ветра
│       │   ├── aqi-colors.ts          # Цветовая шкала AQI
│       │   └── accessibility.ts       # ГОСТ Р 52872-2019
│       ├── index.html
│       ├── package.json
│       └── vite.config.ts
│
├── migrations/
│   ├── 000001_init_schema.up.sql
│   ├── 000001_init_schema.down.sql
│   ├── 000002_create_hypertable.up.sql
│   └── ...
│
├── docker/
│   ├── Dockerfile                     # Multi-stage build
│   ├── docker-compose.yml             # Продакшн
│   ├── docker-compose.dev.yml         # Разработка (с hot reload)
│   └── nginx/
│       ├── nginx.conf
│       └── ssl/                       # TLS сертификаты (или Let's Encrypt)
│
├── scripts/
│   ├── install.sh                     # Одна команда установки
│   ├── uninstall.sh
│   └── update.sh
│
├── api/
│   └── openapi.yaml                   # OpenAPI 3.1 (на русском языке)
│
├── sqlc.yaml                          # Конфигурация sqlc
├── config.example.yaml
├── .env.example
├── go.mod
├── go.sum
├── Makefile
├── .golangci.yml                      # Linter конфигурация
└── README.md
```

---

## 3. БАЗА ДАННЫХ (СХЕМА)

### 3.1. Таблицы PostgreSQL

```sql
-- Пользователи
CREATE TABLE users (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT        NOT NULL UNIQUE,
    username    TEXT        NOT NULL UNIQUE,
    password    TEXT        NOT NULL,      -- Argon2id hash
    role        TEXT        NOT NULL CHECK (role IN ('admin','analyst','viewer')),
    is_active   BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Датчики
CREATE TABLE sensors (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    address     TEXT        NOT NULL,
    lat         DOUBLE PRECISION NOT NULL,
    lng         DOUBLE PRECISION NOT NULL,
    type        TEXT        NOT NULL CHECK (type IN ('gas','dust','combo')),
    is_active   BOOLEAN     NOT NULL DEFAULT true,
    last_seen   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Измерения (TimescaleDB hypertable)
CREATE TABLE measurements (
    time        TIMESTAMPTZ      NOT NULL,
    sensor_id   UUID             NOT NULL REFERENCES sensors(id),
    no2         DOUBLE PRECISION,
    o3          DOUBLE PRECISION,
    co          DOUBLE PRECISION,
    h2s         DOUBLE PRECISION,
    so2         DOUBLE PRECISION,
    pm25        DOUBLE PRECISION,
    temperature DOUBLE PRECISION,
    humidity    DOUBLE PRECISION,
    pressure    DOUBLE PRECISION,
    wind_speed  DOUBLE PRECISION,
    wind_dir    DOUBLE PRECISION
);
SELECT create_hypertable('measurements', 'time');
CREATE INDEX ON measurements (sensor_id, time DESC);

-- Прогнозы (TimescaleDB hypertable)
CREATE TABLE forecasts (
    time            TIMESTAMPTZ      NOT NULL,
    point_id        TEXT             NOT NULL,  -- "lat_lng" или named point
    lat             DOUBLE PRECISION NOT NULL,
    lng             DOUBLE PRECISION NOT NULL,
    horizon_hours   INTEGER          NOT NULL,
    aqi             INTEGER          NOT NULL,
    aqi_category    TEXT             NOT NULL,  -- 'good','moderate','unhealthy',...
    no2_forecast    DOUBLE PRECISION,
    pm25_forecast   DOUBLE PRECISION,
    so2_forecast    DOUBLE PRECISION,
    model_version   TEXT             NOT NULL,
    created_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);
SELECT create_hypertable('forecasts', 'time');

-- API токены
CREATE TABLE api_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    token_hash  TEXT        NOT NULL UNIQUE,  -- HMAC-SHA256 hash
    last_used   TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Аудит-лог
CREATE TABLE audit_log (
    id          BIGSERIAL   PRIMARY KEY,
    user_id     UUID        REFERENCES users(id),
    action      TEXT        NOT NULL,
    resource    TEXT        NOT NULL,
    resource_id TEXT,
    ip_address  INET,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Загруженные данные (email/файл)
CREATE TABLE uploaded_data (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    filename    TEXT        NOT NULL,
    source      TEXT        NOT NULL,  -- 'email','api','manual'
    status      TEXT        NOT NULL CHECK (status IN ('pending','valid','invalid','processed')),
    rows_total  INTEGER     DEFAULT 0,
    rows_valid  INTEGER     DEFAULT 0,
    error_msg   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Обратная связь
CREATE TABLE feedback (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT,
    subject     TEXT        NOT NULL,
    message     TEXT        NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'new',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 3.2. Retention Policy (автоматически через TimescaleDB)
```sql
-- Хранить сырые измерения 60 месяцев (требование ТЗ)
SELECT add_retention_policy('measurements', INTERVAL '60 months');

-- Агрегация: 1-часовые бакеты для быстрых запросов
CREATE MATERIALIZED VIEW measurements_1h
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    sensor_id,
    AVG(no2)   AS avg_no2,
    AVG(pm25)  AS avg_pm25,
    AVG(co)    AS avg_co,
    MAX(no2)   AS max_no2,
    MAX(pm25)  AS max_pm25
FROM measurements
GROUP BY bucket, sensor_id;
```

---

## 4. API ЭНДПОИНТЫ (ПОЛНАЯ КАРТА)

```
PUBLIC (без авторизации):
GET  /health                           — статус сервиса
GET  /ready                            — готовность (БД подключена)
GET  /widget                           — HTML публичного виджета
GET  /widget/data                      — JSON данные для виджета
GET  /widget/forecast                  — прогноз для виджета
GET  /widget/weather                   — метеоданные для виджета
GET  /api/v1/docs                      — OpenAPI документация (SwaggerUI)

AUTH:
POST /api/v1/auth/login                — авторизация (email+password)
POST /api/v1/auth/refresh              — обновление access token
POST /api/v1/auth/logout               — инвалидация refresh token
POST /api/v1/auth/forgot-password      — запрос сброса пароля
POST /api/v1/auth/reset-password       — сброс пароля по токену

PROTECTED (JWT обязателен):
— Пользователи (только Admin):
GET    /api/v1/users                   — список пользователей
POST   /api/v1/users                   — создание пользователя
GET    /api/v1/users/{id}              — получение пользователя
PATCH  /api/v1/users/{id}              — обновление пользователя
DELETE /api/v1/users/{id}              — удаление пользователя

— Датчики (Admin + Analyst):
GET    /api/v1/sensors                 — список датчиков
POST   /api/v1/sensors                 — добавление датчика (Admin)
GET    /api/v1/sensors/{id}            — данные датчика
PATCH  /api/v1/sensors/{id}            — обновление (Admin)
DELETE /api/v1/sensors/{id}            — удаление (Admin)
GET    /api/v1/sensors/{id}/status     — онлайн/оффлайн, последнее значение

— Измерения (все роли):
GET    /api/v1/measurements            — ?sensor_id=&from=&to=&period=1h|1d
GET    /api/v1/measurements/latest     — последние значения всех датчиков
GET    /api/v1/measurements/aggregate  — агрегированные данные для графиков

— Прогнозы (все роли):
GET    /api/v1/forecast/points         — список контрольных точек
GET    /api/v1/forecast/current        — текущий прогноз всех точек
GET    /api/v1/forecast/{point_id}     — прогноз конкретной точки
GET    /api/v1/forecast/city-average   — средний AQI по городу
GET    /api/v1/forecast/district/{id}  — средний AQI по району

— API токены (Admin + свой аккаунт):
GET    /api/v1/tokens                  — список токенов
POST   /api/v1/tokens                  — создание токена
DELETE /api/v1/tokens/{id}             — удаление токена

— Загрузка данных (Admin + Analyst):
POST   /api/v1/ingest/data             — загрузка CSV/Excel (multipart)
GET    /api/v1/ingest/history          — история загрузок
GET    /api/v1/ingest/validation-rules — правила валидации
PUT    /api/v1/ingest/validation-rules — обновление правил (Admin)

— Отчёты (Admin + Analyst):
GET    /api/v1/reports                 — список сформированных отчётов
POST   /api/v1/reports/generate        — запрос генерации отчёта
GET    /api/v1/reports/{id}/download   — скачать отчёт (PDF/Excel/Word)

— Обратная связь (все роли):
POST   /api/v1/feedback                — отправка обращения
GET    /api/v1/feedback                — список обращений (Admin)

— Статистика (Admin):
GET    /api/v1/stats/availability      — статистика доступности за период
GET    /api/v1/stats/data-coverage     — % полноты данных
```

---

## 5. DOCKER COMPOSE (docker-compose.yml)

```yaml
version: "3.9"

services:
  app:
    image: aqi-platform:latest
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      - AQI_DB_HOST=postgres
      - AQI_DB_PORT=5432
      - AQI_DB_NAME=aqi
      - AQI_DB_USER=aqi
      - AQI_DB_PASS=${DB_PASSWORD}
      - AQI_REDIS_ADDR=redis:6379
      - AQI_JWT_SECRET=${JWT_SECRET}
      - AQI_SERVER_PORT=8080
    volumes:
      - app_data:/data
    networks:
      - aqi_net
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: timescale/timescaledb:latest-pg17
    restart: unless-stopped
    environment:
      - POSTGRES_DB=aqi
      - POSTGRES_USER=aqi
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - aqi_net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U aqi -d aqi"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: redis-server --requirepass ${REDIS_PASSWORD} --maxmemory 128mb
    volumes:
      - redis_data:/data
    networks:
      - aqi_net
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    depends_on:
      - app
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./docker/nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - nginx_certs:/etc/nginx/ssl
    networks:
      - aqi_net

volumes:
  postgres_data:
  redis_data:
  app_data:
  nginx_certs:

networks:
  aqi_net:
    driver: bridge
```

---

## 6. DOCKERFILE (Multi-stage)

```dockerfile
# Stage 1: Build Go binary
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Build frontend (TypeScript)
FROM node:22-alpine AS frontend-builder
WORKDIR /web

COPY web/dashboard/package*.json ./dashboard/
COPY web/widget/package*.json ./widget/
RUN cd dashboard && npm ci && cd ../widget && npm ci

COPY web/dashboard ./dashboard
COPY web/widget ./widget
RUN cd dashboard && npm run build && cd ../widget && npm run build

# Back to Go build
FROM golang:1.23-alpine AS go-builder
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /web/dashboard/dist ./web/dashboard/dist
COPY --from=frontend-builder /web/widget/dist ./web/widget/dist

# Build: статически слинкованный бинарник
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -extldflags=-static \
    -X main.version=$(git describe --tags --always) \
    -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -trimpath \
    -o aqi-platform \
    ./cmd/aqi-platform

# Stage 2: Minimal runtime image
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=go-builder /build/aqi-platform .
COPY migrations ./migrations

EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/aqi-platform", "server"]
```

**Результирующий образ:** ~12–15 МБ (distroless + статический Go бинарник)

---

## 7. СКРИПТ АВТОУСТАНОВКИ (install.sh)

```bash
#!/usr/bin/env bash
# AQI Platform Installer
# Использование: curl -fsSL https://your-domain/install.sh | bash

set -euo pipefail

# Проверка root
if [[ $EUID -ne 0 ]]; then
    echo "Запустите от root: sudo bash install.sh"
    exit 1
fi

# Проверка Docker
if ! command -v docker &>/dev/null; then
    echo "Устанавливаю Docker..."
    curl -fsSL https://get.docker.com | bash
fi

# Проверка Docker Compose
if ! docker compose version &>/dev/null; then
    echo "Устанавливаю Docker Compose..."
    apt-get install -y docker-compose-plugin 2>/dev/null || \
    yum install -y docker-compose-plugin 2>/dev/null || true
fi

# Установочная директория
INSTALL_DIR="/opt/aqi-platform"
mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

# Генерация секретов
DB_PASSWORD=$(openssl rand -base64 32 | tr -d /=+ | cut -c1-32)
JWT_SECRET=$(openssl rand -base64 64 | tr -d /=+)
REDIS_PASSWORD=$(openssl rand -base64 24 | tr -d /=+)

# .env файл
cat > .env <<EOF
DB_PASSWORD=${DB_PASSWORD}
JWT_SECRET=${JWT_SECRET}
REDIS_PASSWORD=${REDIS_PASSWORD}
EOF
chmod 600 .env

# Скачать docker-compose.yml
curl -fsSL https://your-release-url/docker-compose.yml -o docker-compose.yml

# Запуск
docker compose pull
docker compose up -d

echo ""
echo "✅ AQI Platform установлена!"
echo "   Адрес: http://$(hostname -I | awk '{print $1}')"
echo "   Логин admin: admin@localhost"
echo "   Пароль: (задайте через интерфейс при первом входе)"
```

---

## 8. ПРОГНОЗНЫЙ ДВИЖОК (Go)

```go
// internal/forecast/engine.go

// ForecastEngine — модуль прогнозирования AQI
// Алгоритм: EWMA + IDW (Inverse Distance Weighting)
// Горизонт: 6+ часов, обновление каждые 20 минут

type ForecastEngine struct {
    repo     repository.MeasurementRepository
    points   []domain.ForecastPoint  // контрольные точки (≥10)
    model    *TimeSeriesModel
}

// TimeSeriesModel использует EWMA (Exponentially Weighted Moving Average)
// для краткосрочного прогноза + IDW для пространственной интерполяции
// 
// Никаких внешних ML библиотек — чистый Go:
// - EWMA: O(n) вычисление, 20-минутные интервалы
// - IDW: взвешенная интерполяция по 4 датчикам
// - Bias Correction: поправка по последним измерениям

// Run запускает цикл прогнозирования (каждые 20 минут)
func (e *ForecastEngine) Run(ctx context.Context) {
    ticker := time.NewTicker(20 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            e.computeAndSave(ctx)
        case <-ctx.Done():
            return
        }
    }
}
```

---

## 9. БЕЗОПАСНОСТЬ: КОД (ПРИМЕРЫ)

### Argon2id хеширование паролей
```go
// internal/service/auth_service.go
import "golang.org/x/crypto/argon2"

const (
    argonTime    = 3         // итерации
    argonMemory  = 64 * 1024 // 64 MB (OWASP 2025)
    argonThreads = 4
    argonKeyLen  = 32
)

func HashPassword(password string) (string, error) {
    salt := make([]byte, 16)
    if _, err := rand.Read(salt); err != nil {
        return "", err
    }
    hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
    // формат: $argon2id$v=19$m=65536,t=3,p=4$<salt_b64>$<hash_b64>
    return encodeArgon2Hash(hash, salt), nil
}
```

### RBAC middleware
```go
// internal/middleware/rbac.go

type Role string

const (
    RoleAdmin   Role = "admin"
    RoleAnalyst Role = "analyst"
    RoleViewer  Role = "viewer"
)

func RequireRole(roles ...Role) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := ClaimsFromContext(r.Context())
            if claims == nil {
                writeError(w, http.StatusUnauthorized, "authentication required")
                return
            }
            for _, role := range roles {
                if Role(claims.Role) == role {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            writeError(w, http.StatusForbidden, "insufficient permissions")
        })
    }
}
```

### Rate Limiter (token bucket)
```go
// internal/middleware/ratelimit.go
// 100 req/min для авторизованных, 10 req/min для /auth/login

import "golang.org/x/time/rate"

func RateLimit(rps rate.Limit, burst int) func(http.Handler) http.Handler {
    limiter := rate.NewLimiter(rps, burst)
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                w.Header().Set("Retry-After", "60")
                writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 10. ПЛАН РАЗРАБОТКИ (СПРИНТЫ)

### Спринт 1 (1–2 недели): Фундамент
- [ ] `go.mod` + все зависимости
- [ ] Структура директорий
- [ ] Config (Viper + ENV)
- [ ] PostgreSQL + TimescaleDB подключение (pgx/v5)
- [ ] golang-migrate миграции
- [ ] sqlc генерация
- [ ] Базовый HTTP сервер (Chi)
- [ ] Health/Ready эндпоинты
- [ ] Docker Compose + Dockerfile
- [ ] Makefile (build, test, lint, docker)

### Спринт 2 (1–2 недели): Аутентификация и RBAC
- [ ] User domain model
- [ ] Argon2id хеширование
- [ ] JWT (access + refresh tokens)
- [ ] Refresh token rotation
- [ ] Blacklist в Redis
- [ ] Все /auth/* эндпоинты
- [ ] RBAC middleware
- [ ] Rate limiting middleware
- [ ] Security headers middleware
- [ ] Audit log

### Спринт 3 (1–2 недели): Датчики и измерения
- [ ] Sensor CRUD API
- [ ] Measurements API (чтение + агрегация)
- [ ] MQTT agent (приём с датчиков)
- [ ] Email IMAP agent (Ecology@kemerovo.ru)
- [ ] CSV/Excel ingest endpoint
- [ ] Валидация данных по правилам

### Спринт 4 (2 недели): Прогнозный движок
- [ ] Domain: AQI calculation (российская методология + международная)
- [ ] EWMA time series модель
- [ ] IDW spatial interpolation
- [ ] Forecast points management
- [ ] Forecast API
- [ ] Фоновый scheduler (каждые 20 мин)
- [ ] Bias correction по измерениям

### Спринт 5 (1–2 недели): Frontend Dashboard (TypeScript)
- [ ] Vite + TypeScript проект
- [ ] Страница авторизации
- [ ] Dashboard: карта MapLibre + маркеры AQI
- [ ] Графики Chart.js (выбор показателей)
- [ ] Таблица прогнозов по точкам
- [ ] Настройки: пользователи, токены
- [ ] Форма обратной связи

### Спринт 6 (1–2 недели): Публичный виджет (TypeScript)
- [ ] Самодостаточный iframe-виджет
- [ ] MapLibre GL JS + маркеры AQI (≥10 точек)
- [ ] Таймлайн анимация (D3 или vanilla)
- [ ] Анимация ветровых потоков
- [ ] Полигоны районов/жилых/промышленных зон
- [ ] Клик на точку → прогноз
- [ ] Метеоданные панель
- [ ] 3-уровневая иерархия
- [ ] ГОСТ Р 52872-2019 (доступность)

### Спринт 7 (1 неделя): Отчёты и финальная полировка
- [ ] Генерация PDF/Excel/Word отчётов
- [ ] API-токены полный CRUD
- [ ] OpenAPI документация (русский язык)
- [ ] install.sh скрипт
- [ ] README.md полная документация
- [ ] Тесты (coverage >70%)
- [ ] golangci-lint без ошибок

---

## 11. go.mod ЗАВИСИМОСТИ

```go
module github.com/your-org/aqi-platform

go 1.23

require (
    // HTTP
    github.com/go-chi/chi/v5           v5.2.x  // роутер
    github.com/go-chi/cors             v1.2.x  // CORS
    github.com/go-chi/httprate         v0.14.x // rate limiting
    
    // Database
    github.com/jackc/pgx/v5            v5.7.x  // PostgreSQL driver
    github.com/golang-migrate/migrate/v4 v4.18.x // миграции
    
    // Auth
    github.com/golang-jwt/jwt/v5       v5.2.x  // JWT
    golang.org/x/crypto                v0.30.x // Argon2id
    
    // Config
    github.com/spf13/viper             v1.20.x
    
    // Redis
    github.com/redis/go-redis/v9       v9.7.x
    
    // Validation
    github.com/go-playground/validator/v10 v10.25.x
    
    // Email
    github.com/emersion/go-message     v0.18.x // IMAP/MIME
    
    // Time
    golang.org/x/time                  v0.10.x // rate.Limiter
    
    // UUID
    github.com/google/uuid             v1.6.x
    
    // CLI
    github.com/spf13/cobra             v1.9.x
    
    // Logging (stdlib slog — без зависимостей)
    // Reporting
    github.com/xuri/excelize/v2        v2.9.x  // Excel
)
```

**Намеренно исключены:**
- ❌ GORM — используем sqlc
- ❌ Gin/Fiber — используем Chi
- ❌ logrus/zap — используем stdlib slog (Go 1.21+)
- ❌ testcontainers — используем docker для тестов
- ❌ любые AI/ML библиотеки — прогноз на чистом Go (EWMA + IDW)

---

## 12. ЛИЦЕНЗИИ ИСПОЛЬЗУЕМЫХ КОМПОНЕНТОВ

| Компонент | Лицензия | Коммерческое использование |
|-----------|----------|---------------------------|
| Go stdlib | BSD 3-Clause | ✅ Свободно |
| Chi v5 | MIT | ✅ Свободно |
| pgx/v5 | MIT | ✅ Свободно |
| golang-jwt/v5 | MIT | ✅ Свободно |
| golang-migrate | MIT | ✅ Свободно |
| Viper | MIT | ✅ Свободно |
| go-redis/v9 | BSD 2-Clause | ✅ Свободно |
| validator/v10 | MIT | ✅ Свободно |
| MapLibre GL JS | BSD 2-Clause | ✅ Свободно |
| Chart.js | MIT | ✅ Свободно |
| PostgreSQL | PostgreSQL License | ✅ Свободно |
| TimescaleDB | Apache 2.0 | ✅ Свободно (self-hosted) |
| Redis | BSD 3-Clause | ✅ Свободно |
| Nginx | BSD 2-Clause | ✅ Свободно |

**Всё ПО: Apache 2.0 / MIT / BSD → можно использовать коммерчески, нет копилефта (GPL), нет ограничений.**

---

*Этот документ — полное ТЗ на разработку. Код будет на 100% оригинальным, написанным с нуля на основе изученных паттернов и архитектурных решений, без копирования чужого кода.*
