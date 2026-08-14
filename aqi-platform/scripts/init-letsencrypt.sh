#!/usr/bin/env bash
# =============================================================================
# init-letsencrypt.sh — первичный выпуск сертификата Let's Encrypt
#
# Запустить один раз после первого деплоя:
#   bash /opt/aqi-source/aqi-platform/scripts/init-letsencrypt.sh
#
# Что делает:
#   1. Убеждается что платформа запущена (docker compose up -d)
#   2. Получает реальный сертификат от Let's Encrypt через webroot-challenge
#   3. Перезапускает nginx с реальным сертификатом
#
# При следующих запусках docker compose — nginx-init автоматически создаёт
# самоподписанный сертификат если Let's Encrypt ещё недоступен,
# поэтому nginx всегда стартует без ошибок.
# =============================================================================

set -euo pipefail

DOMAIN="217-198-12-184.sslip.io"
EMAIL="${LETSENCRYPT_EMAIL:-admin@${DOMAIN}}"
STAGING="${LETSENCRYPT_STAGING:-0}"   # 1 = тест без лимитов LE, 0 = боевой

COMPOSE_FILE="${SOURCE_DIR:-/opt/aqi-source}/aqi-platform/docker/docker-compose.yml"
ENV_FILE="${INSTALL_DIR:-/opt/aqi-platform}/.env"
PROJECT="aqi-platform"

BOLD='\033[1m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'

log()  { echo -e "${BOLD}[SSL]${NC} $*"; }
ok()   { echo -e "${GREEN}[OK]${NC}  $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
die()  { echo -e "${RED}[ERR]${NC}  $*" >&2; exit 1; }

[[ $EUID -ne 0 ]] && die "Запустите от root: sudo bash $0"
command -v docker >/dev/null || die "Docker не установлен"

# ── 1. Запускаем платформу если ещё не запущена ───────────────────────────
log "Запускаю платформу (если не запущена)..."
docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" \
  --project-name "${PROJECT}" up -d
sleep 5
ok "Платформа запущена"

# ── 2. Получаем реальный сертификат ──────────────────────────────────────
log "Получаю сертификат от Let's Encrypt для ${DOMAIN}..."

STAGING_FLAG=""
[[ "${STAGING}" == "1" ]] && {
  STAGING_FLAG="--staging"
  warn "Тестовый режим (staging) — сертификат не будет доверенным браузерами"
}

docker run --rm \
  -v "${PROJECT}_certbot_data:/etc/letsencrypt" \
  -v "${PROJECT}_certbot_webroot:/var/www/certbot" \
  certbot/certbot certonly \
    --webroot \
    --webroot-path=/var/www/certbot \
    ${STAGING_FLAG} \
    --email "${EMAIL}" \
    --agree-tos \
    --no-eff-email \
    --force-renewal \
    -d "${DOMAIN}"

ok "Сертификат получен"

# ── 3. Перезапуск nginx с реальным сертификатом ───────────────────────────
log "Перезапускаю nginx с Let's Encrypt сертификатом..."
docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" \
  --project-name "${PROJECT}" restart nginx
sleep 3

ok "Готово!"
echo ""
echo -e "${BOLD}Сайт доступен по адресу:${NC} https://${DOMAIN}"
echo ""
echo "Сертификат действителен 90 дней и обновляется автоматически каждые 12 часов."
echo "Повторно запускать этот скрипт не нужно."
