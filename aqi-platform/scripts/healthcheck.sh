#!/usr/bin/env bash
# =============================================================================
# healthcheck.sh — быстрая проверка состояния всей платформы
#
# Использование:
#   bash scripts/healthcheck.sh          # вывод в терминал
#   bash scripts/healthcheck.sh --json   # вывод в JSON
#   bash scripts/healthcheck.sh --quiet  # только exit code (0=ok, 1=проблема)
# =============================================================================

set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-/opt/aqi-platform/docker/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-/opt/aqi-platform/.env}"
PROJECT_DIR="${PROJECT_DIR:-/opt/aqi-platform}"

# Цвета
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; BOLD='\033[1m'; NC='\033[0m'

MODE="${1:-}"
OVERALL_STATUS=0

# Получаем BASE_URL из .env
BASE_URL="http://localhost"
if [[ -f "${ENV_FILE}" ]]; then
    url=$(grep '^BASE_URL=' "${ENV_FILE}" 2>/dev/null | cut -d= -f2 | tr -d '"' || true)
    [[ -n "${url}" ]] && BASE_URL="${url}"
fi

# ── Проверка контейнера ────────────────────────────────────────────────────────
check_container() {
    local name="$1"
    local result
    result=$(docker inspect \
      --format='{{.State.Status}}|{{.State.Health.Status}}' \
      "aqi_${name}" 2>/dev/null || echo "missing|missing")

    local state health
    state=$(echo "${result}" | cut -d'|' -f1)
    health=$(echo "${result}" | cut -d'|' -f2)

    if [[ "${state}" == "running" ]]; then
        if [[ "${health}" == "healthy" || "${health}" == "" || "${health}" == "<nil>" ]]; then
            echo "ok"
        elif [[ "${health}" == "starting" ]]; then
            echo "starting"
        else
            echo "unhealthy"
        fi
    elif [[ "${state}" == "missing" ]]; then
        echo "missing"
    else
        echo "${state}"
    fi
}

# ── HTTP-проверка ─────────────────────────────────────────────────────────────
check_http() {
    local url="$1"
    local timeout="${2:-5}"
    if curl -sf --max-time "${timeout}" "${url}" > /dev/null 2>&1; then
        echo "ok"
    else
        echo "fail"
    fi
}

# ── Размер диска ──────────────────────────────────────────────────────────────
disk_usage() {
    df / | tail -1 | awk '{gsub(/%/, "", $5); print $5}'
}

# ── Docker volumes ────────────────────────────────────────────────────────────
volumes_size() {
    docker system df --format "{{.Type}}\t{{.Size}}" 2>/dev/null | \
      grep Volumes | awk '{print $2}' || echo "?"
}

# ── Сбор данных ───────────────────────────────────────────────────────────────
declare -A RESULTS

SERVICES=("postgres" "redis" "app" "nginx" "prometheus" "grafana" "backup")
for svc in "${SERVICES[@]}"; do
    RESULTS["${svc}"]=$(check_container "${svc}")
    [[ "${RESULTS["${svc}"]}" != "ok" ]] && OVERALL_STATUS=1
done

HTTP_HEALTH=$(check_http "${BASE_URL}/health")
HTTP_API=$(check_http "${BASE_URL}/api/v1/public/forecast/current")
DISK=$(disk_usage)
VOL_SIZE=$(volumes_size)

[[ "${HTTP_HEALTH}" != "ok" ]] && OVERALL_STATUS=1
(( DISK >= 90 )) && OVERALL_STATUS=1

# ── Вывод ─────────────────────────────────────────────────────────────────────
status_icon() {
    case "$1" in
        ok)        echo -e "${GREEN}✓${NC}" ;;
        starting)  echo -e "${YELLOW}⟳${NC}" ;;
        missing)   echo -e "${RED}✗${NC}" ;;
        fail)      echo -e "${RED}✗${NC}" ;;
        *)         echo -e "${RED}✗${NC}" ;;
    esac
}

status_color() {
    case "$1" in
        ok)        echo -e "${GREEN}$1${NC}" ;;
        starting)  echo -e "${YELLOW}$1${NC}" ;;
        *)         echo -e "${RED}$1${NC}" ;;
    esac
}

if [[ "${MODE}" == "--json" ]]; then
    # JSON-вывод для мониторинга
    python3 -c "
import json, sys
data = {
    'timestamp': '$(date -u +%Y-%m-%dT%H:%M:%SZ)',
    'overall': '$( [[ ${OVERALL_STATUS} -eq 0 ]] && echo ok || echo fail)',
    'services': {
$(for svc in "${SERVICES[@]}"; do
    echo "        '${svc}': '${RESULTS[${svc}]}',"
done)
    },
    'http': {
        'health': '${HTTP_HEALTH}',
        'forecast': '${HTTP_API}'
    },
    'disk_used_pct': ${DISK},
    'volumes_size': '${VOL_SIZE}'
}
print(json.dumps(data, indent=2))
"
elif [[ "${MODE}" == "--quiet" ]]; then
    exit ${OVERALL_STATUS}
else
    # Читаемый вывод
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  AQI Platform — Статус системы$(date '+  %d.%m.%Y %H:%M:%S')${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "  ${BOLD}Контейнеры:${NC}"
    for svc in "${SERVICES[@]}"; do
        icon=$(status_icon "${RESULTS[${svc}]}")
        status_txt=$(status_color "${RESULTS[${svc}]}")
        printf "    %s  %-12s %s\n" "${icon}" "${svc}" "${status_txt}"
    done
    echo ""
    echo -e "  ${BOLD}HTTP:${NC}"
    printf "    %s  %-20s %s\n" "$(status_icon "${HTTP_HEALTH}")" "/health endpoint" "$(status_color "${HTTP_HEALTH}")"
    printf "    %s  %-20s %s\n" "$(status_icon "${HTTP_API}")"    "/forecast/current" "$(status_color "${HTTP_API}")"
    echo ""
    echo -e "  ${BOLD}Ресурсы:${NC}"
    disk_color="${GREEN}"
    (( DISK >= 80 )) && disk_color="${YELLOW}"
    (( DISK >= 90 )) && disk_color="${RED}"
    printf "    %-22s %b%s%%%b\n" "Диск (/):" "${disk_color}" "${DISK}" "${NC}"
    printf "    %-22s %s\n" "Docker volumes:" "${VOL_SIZE}"
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    if [[ ${OVERALL_STATUS} -eq 0 ]]; then
        echo -e "  ${BOLD}${GREEN}  СИСТЕМА РАБОТАЕТ НОРМАЛЬНО  ${NC}"
    else
        echo -e "  ${BOLD}${RED}  ОБНАРУЖЕНЫ ПРОБЛЕМЫ — ТРЕБУЕТСЯ ВНИМАНИЕ  ${NC}"
    fi
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
fi

exit ${OVERALL_STATUS}
