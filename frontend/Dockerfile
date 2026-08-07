# Многоэтапная сборка React SPA
# Stage 1: сборка
FROM node:22-slim AS builder

WORKDIR /app

# Устанавливаем зависимости отдельно для кэширования слоя
COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts

# Копируем исходники и собираем
COPY . .
RUN npm run build


# Stage 2: финальный образ — просто статические файлы для Nginx
FROM alpine:3.20 AS spa

WORKDIR /spa

COPY --from=builder /app/dist .

# Контейнер не запускает сервер — его цель передать файлы через volume.
# Используется как init-container в docker-compose.
CMD ["true"]
