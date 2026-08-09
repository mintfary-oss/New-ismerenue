#!/usr/bin/env bash
# =============================================================================
# watchdog.sh — сторожевой процесс AQI Platform
#
# Проверяет здоровье всех сервисов каждые N секунд.
# При обнаружении проблемы:
#   1. Перезапускает упавший контейнер
#   2. Пишет в лог с меткой времени
#   3. Отправляет email-уведомление (если настроен SMTP)
#   4. Создаёт alert-файл в /tmp/aqi_alert_<service>.flag
#
# Использование:
#   # Запустить в фоне
#   sudo bash scripts/watchdog.sh &
#
#   # Или через systemd timer (см. ниже — watchdog.timer)
#   sudo systemctl start aqi-watchdog.timer
#
#   # Проверка статуса
#   tail -f /var/log/aqi-watchdog.log
# =============================================================================

set -euo pipefail

# ── Конфигурация ──────────────────────────────────────────────────────────────
COMPOSE_FILE="${COMPOSE_FILE:-/opt/aqi-platform/docker/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-/opt/aqi-platform/.env}"
PROJECT_DIR="${PROJECT_DIR:-/opt/aqi-platform}"
LOG_FILE="${LOG_FILE:-/var/log/aqi-watchdog.log}"
CHECK_INTERVAL="${CHECK_INTERVAL:-30}"        # секунд между проверками
MAX_RESTART_ATTEMPTS="${MAX_RESTART_ATTEMPTS:-3}"  # максимум перезапусков подряд
RESTART_COOLDOWN="${RESTART_COOLDOWN:-120}"   # секунд ожидания после MAX попыток

# ── Критичные сервисы (в порядке зависимостей) ───────────────────────────────
CRITICAL_SERVICES=("postgres" "redis" "app" "nginx")
# Некритичные — логируем, но не перезапускаем агрессивно
OPTIONAL_SERVICES=("prometheus" "grafana" "backup")

# Глобальные переменные для счётчиков перезапусков
declare -A RESTART_COUNT
declare -A LAST_RESTART_TIME

# ── Цвета для лога ────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# ── Утилиты ───────────────────────────────────────────────────────────────────
log() {
    local level="$1"
    local msg="$2"
    local ts
    ts=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${ts}] [${level}] ${msg}" | tee -a "${LOG_FILE}"
}

info()  { log "INFO " "$1"; }
warn()  { log "WARN " "$1"; }
error() { log "ERROR" "$1"; }

# Отправка email-уведомления (если SMTP настроен в .env)
notify_email() {
    local subject="$1"
    local body="$2"

    # Загружаем SMTP настройки из .env
    if [[ -f "${ENV_FILE}" ]]; then
        # shellcheck source=/dev/null
        set +u  # .env может не содержать нужных переменных
        SMTP_HOST="${SMTP_HOST:-$(grep '^SMTP_HOST=' "${ENV_FILE}" | cut -d= -f2)}"
        SMTP_USER="${SMTP_USER:-$(grep '^SMTP_USER=' "${ENV_FILE}" | cut -d= -f2)}"
        SMTP_PASS="${SMTP_PASS:-$(grep '^SMTP_PASS=' "${ENV_FILE}" | cut -d= -f2)}"
        ALERT_RECIPIENTS="${ALERT_RECIPIENTS:-$(grep '^ALERT_RECIPIENTS=' "${ENV_FILE}" | cut -d= -f2)}"
        set -u
    fi

    if [[ -z "${SMTP_HOST:-}" || -z "${ALERT_RECIPIENTS:-}" ]]; then
        return 0  # SMTP не настроен — пропускаем
    fi

    # Отправляем через curl (должен быть установлен)
    if command -v curl &>/dev/null; then
        curl -s --ssl-reqd \
          --url "smtp://${SMTP_HOST}:587" \
          --user "${SMTP_USER}:${SMTP_PASS}" \
          --mail-from "${SMTP_USER}" \
          --mail-rcpt "${ALERT_RECIPIENTS//,/ --mail-rcpt }" \
          --upload-file - << EOF
Subject: [AQI Platform] ${subject}
Content-Type: text/plain; charset=UTF-8

${body}

---
Время: $(date '+%Y-%m-%d %H:%M:%S')
Хост: $(hostname)
EOF
    fi
}

# Команда docker compose
compose_cmd() {
    docker compose \
      -f "${COMPOSE_FILE}" \
      --env-file "${ENV_FILE}" \
      --project-directory "${PROJECT_DIR}" \
      "$@"
}

# ── Проверка состояния контейнера ─────────────────────────────────────────────
check_service() {
    local service="$1"
    local status

    # Получаем состояние контейнера
    status=$(docker inspect \
      --format='{{.State.Health.Status}}-{{.State.Status}}' \
      "aqi_${service}" 2>/dev/null || echo "missing")

    case "${status}" in
        "healthy-running")
            return 0  # всё хорошо
            ;;
        "starting-running")
            return 2  # запускается, ждём
            ;;
        *"-running")
            # Нет healthcheck, но контейнер работает
            return 0
            ;;
        "unhealthy-running"|*"-unhealthy")
            return 1  # нездоров
            ;;
        "missing"|*"-exited"|*"-dead")
            return 1  # не работает
            ;;
        *)
            return 1  # неизвестный статус
            ;;
    esac
}

# ── Перезапуск сервиса с отслеживанием попыток ───────────────────────────────
restart_service() {
    local service="$1"
    local now
    now=$(date +%s)

    # Инициализируем счётчик
    RESTART_COUNT["${service}"]=${RESTART_COUNT["${service}"]:-0}
    LAST_RESTART_TIME["${service}"]=${LAST_RESTART_TIME["${service}"]:-0}

    local count=${RESTART_COUNT["${service}"]}
    local last=${LAST_RESTART_TIME["${service}"]}
    local elapsed=$(( now - last ))

    # Если прошло достаточно времени — сбрасываем счётчик
    if (( elapsed > RESTART_COOLDOWN * 2 )); then
        RESTART_COUNT["${service}"]=0
        count=0
    fi

    if (( count >= MAX_RESTART_ATTEMPTS )); then
        warn "Сервис ${service}: достигнут лимит ${MAX_RESTART_ATTEMPTS} перезапусков за ${RESTART_COOLDOWN}с"
        warn "Ожидаем ${RESTART_COOLDOWN}с перед следующей попыткой..."
        sleep "${RESTART_COOLDOWN}"
        RESTART_COUNT["${service}"]=0
        return 0
    fi

    error "Сервис ${service} не работает! Попытка перезапуска $((count + 1))/${MAX_RESTART_ATTEMPTS}..."

    # Создаём flag-файл для внешнего мониторинга
    touch "/tmp/aqi_alert_${service}.flag"

    # Перезапускаем контейнер
    if compose_cmd restart "${service}" 2>&1 | tee -a "${LOG_FILE}"; then
        RESTART_COUNT["${service}"]=$(( count + 1 ))
        LAST_RESTART_TIME["${service}"]=${now}
        info "Сервис ${service} перезапущен (попытка $((count + 1)))"

        # Уведомление
        notify_email \
          "⚠️ Перезапуск сервиса ${service}" \
          "Сервис ${service} был перезапущен автоматически. Попытка: $((count + 1))/${MAX_RESTART_ATTEMPTS}."
    else
        error "Не удалось перезапустить сервис ${service}!"
        notify_email \
          "🔴 КРИТИЧЕСКАЯ ОШИБКА: ${service}" \
          "Не удалось перезапустить сервис ${service}! Требуется ручное вмешательство."
    fi
}

# ── HTTP-проверка здоровья ─────────────────────────────────────────────────────
# -k (--insecure) нужен если BASE_URL использует HTTPS с самоподписанным сертом.
# -L следует редиректам (HTTP → HTTPS).
check_http_health() {
    local url="$1"
    local timeout="${2:-5}"
    curl -skLf --max-time "${timeout}" "${url}" > /dev/null 2>&1
}

# ── Проверка диска ────────────────────────────────────────────────────────────
check_disk_space() {
    local threshold=90  # % использования
    local usage
    usage=$(df / | tail -1 | awk '{gsub(/%/, "", $5); print $5}')

    if (( usage >= threshold )); then
        warn "Диск заполнен на ${usage}%! Рекомендуется освободить место."
        notify_email \
          "⚠️ Диск заполнен на ${usage}%" \
          "На сервере осталось менее $((100 - usage))% свободного места. Очистите старые логи или бэкапы."
    fi
}

# ── Главный цикл ──────────────────────────────────────────────────────────────
main() {
    info "========================================"
    info "AQI Platform Watchdog запущен"
    info "Check interval: ${CHECK_INTERVAL}s"
    info "Max restarts: ${MAX_RESTART_ATTEMPTS}"
    info "Критичные сервисы: ${CRITICAL_SERVICES[*]}"
    info "========================================"

    local iteration=0

    while true; do
        iteration=$(( iteration + 1 ))

        # ── Проверяем критичные сервисы ──────────────────────────────────────
        for service in "${CRITICAL_SERVICES[@]}"; do
            if ! check_service "${service}"; then
                restart_service "${service}"
            else
                # Удаляем flag если сервис восстановился
                rm -f "/tmp/aqi_alert_${service}.flag"
            fi
        done

        # ── Проверяем некритичные сервисы (только лог) ───────────────────────
        for service in "${OPTIONAL_SERVICES[@]}"; do
            if ! check_service "${service}"; then
                warn "Сервис ${service} нездоров (некритично, перезапуск Docker справится)"
            fi
        done

        # ── HTTP-проверка (раз в 5 итераций) ─────────────────────────────────
        if (( iteration % 5 == 0 )); then
            local base_url
            base_url=$(grep '^BASE_URL=' "${ENV_FILE}" 2>/dev/null | cut -d= -f2 || echo "http://localhost")

            if ! check_http_health "${base_url}/health"; then
                error "HTTP health check провален: ${base_url}/health недоступен!"
                restart_service "app"
                restart_service "nginx"
            fi
        fi

        # ── Проверка диска (раз в 20 итераций) ───────────────────────────────
        if (( iteration % 20 == 0 )); then
            check_disk_space
        fi

        # ── Краткий отчёт о статусе (раз в 60 итераций) ──────────────────────
        if (( iteration % 60 == 0 )); then
            info "--- Статус сервисов ---"
            for service in "${CRITICAL_SERVICES[@]}" "${OPTIONAL_SERVICES[@]}"; do
                local status
                status=$(docker inspect \
                  --format='{{.State.Health.Status}}/{{.State.Status}}' \
                  "aqi_${service}" 2>/dev/null || echo "missing")
                info "  ${service}: ${status}"
            done
        fi

        sleep "${CHECK_INTERVAL}"
    done
}

# ── Точка входа ───────────────────────────────────────────────────────────────
# Создаём лог-файл если не существует
mkdir -p "$(dirname "${LOG_FILE}")"
touch "${LOG_FILE}"

# Ротация лога (не более 10 МБ)
if [[ -f "${LOG_FILE}" ]] && (( $(stat -c%s "${LOG_FILE}" 2>/dev/null || echo 0) > 10485760 )); then
    mv "${LOG_FILE}" "${LOG_FILE}.old"
    touch "${LOG_FILE}"
    info "Лог ротирован"
fi

main "$@"
