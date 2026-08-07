#!/usr/bin/env bash
# =============================================================================
# AQI Platform — настройка TLS/SSL через Let's Encrypt (certbot)
#
# Использование:
#   sudo bash scripts/setup-tls.sh --domain monitor.kemerovo.ru --email admin@example.com
#   sudo bash scripts/setup-tls.sh --domain monitor.kemerovo.ru --email admin@example.com --staging
#   sudo bash scripts/setup-tls.sh --self-signed   # для локального dev/тестирования
#
# Что делает скрипт:
#   1. Устанавливает certbot (если не установлен)
#   2. Временно запускает Nginx на HTTP для ACME-challenge
#   3. Получает сертификат Let's Encrypt
#   4. Настраивает cron для auto-renew
#   5. Перезапускает Nginx с HTTPS
# =============================================================================

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'

info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

INSTALL_DIR="${INSTALL_DIR:-/opt/aqi-platform}"
DOMAIN=""
EMAIL=""
STAGING=false
SELF_SIGNED=false

# ── Парсинг аргументов ────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case $1 in
    --domain)    DOMAIN="$2";    shift 2 ;;
    --email)     EMAIL="$2";     shift 2 ;;
    --staging)   STAGING=true;   shift   ;;
    --self-signed) SELF_SIGNED=true; shift ;;
    *) error "Неизвестный аргумент: $1" ;;
  esac
done

[[ $EUID -ne 0 ]] && error "Запустите от имени root: sudo bash setup-tls.sh"

# ── Режим: самоподписанный сертификат (dev/локальный тест) ─────────────────
if [[ "$SELF_SIGNED" == true ]]; then
  info "Генерируем самоподписанный сертификат (dev-режим)..."

  SSL_DIR="$INSTALL_DIR/ssl"
  mkdir -p "$SSL_DIR"

  openssl req -x509 -nodes -days 3650 \
    -newkey rsa:2048 \
    -keyout "$SSL_DIR/key.pem" \
    -out "$SSL_DIR/cert.pem" \
    -subj "/C=RU/ST=Kemerovo/L=Kemerovo/O=AQI Platform/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

  # Монтируем в Docker volume
  docker run --rm \
    -v aqi-platform_nginx_certs:/certs \
    -v "$SSL_DIR:/src:ro" \
    alpine:3.20 \
    sh -c "cp /src/cert.pem /certs/cert.pem && cp /src/key.pem /certs/key.pem && chmod 600 /certs/key.pem"

  success "Самоподписанный сертификат создан: $SSL_DIR/"
  success "Срок действия: 10 лет (dev-режим)"
  warn "Браузер покажет предупреждение — это нормально для dev"

  # Перезапускаем Nginx
  cd "$INSTALL_DIR"
  docker compose restart nginx

  success "Nginx перезапущен с TLS"
  exit 0
fi

# ── Режим: Let's Encrypt (продакшн) ──────────────────────────────────────
[[ -z "$DOMAIN" ]] && error "Укажите домен: --domain monitor.kemerovo.ru"
[[ -z "$EMAIL" ]]  && error "Укажите email: --email admin@example.com"

info "Настройка TLS для домена: $DOMAIN"
info "Email для уведомлений certbot: $EMAIL"

# ── Установка certbot ────────────────────────────────────────────────────
if ! command -v certbot &>/dev/null; then
  info "Устанавливаем certbot..."
  if command -v apt-get &>/dev/null; then
    apt-get update -qq
    apt-get install -y -q certbot
  elif command -v yum &>/dev/null; then
    yum install -y certbot
  else
    # Универсальный способ через snap
    snap install --classic certbot
    ln -sf /snap/bin/certbot /usr/bin/certbot
  fi
fi
success "certbot $(certbot --version 2>&1 | head -1)"

# ── Временный HTTP-сервер для ACME challenge ──────────────────────────────
info "Временно останавливаем Nginx для ACME challenge..."
cd "$INSTALL_DIR"
docker compose stop nginx 2>/dev/null || true

# Папка для ACME challenge
WEBROOT="$INSTALL_DIR/certbot-webroot"
mkdir -p "$WEBROOT"

# Запускаем минимальный HTTP-сервер на порту 80
python3 -m http.server 80 --directory "$WEBROOT" &
HTTP_PID=$!
trap "kill $HTTP_PID 2>/dev/null; true" EXIT

sleep 1
info "HTTP-сервер запущен (PID $HTTP_PID)"

# ── Получаем сертификат ───────────────────────────────────────────────────
CERTBOT_ARGS=(
  certonly
  --webroot
  --webroot-path="$WEBROOT"
  --domain "$DOMAIN"
  --email "$EMAIL"
  --agree-tos
  --non-interactive
  --keep-until-expiring
)

if [[ "$STAGING" == true ]]; then
  CERTBOT_ARGS+=(--staging)
  warn "Используем staging-сервер Let's Encrypt (тест, не выдаёт доверенные сертификаты)"
fi

info "Запрашиваем сертификат у Let's Encrypt..."
certbot "${CERTBOT_ARGS[@]}"

# Останавливаем временный сервер
kill $HTTP_PID 2>/dev/null || true
trap - EXIT

CERT_DIR="/etc/letsencrypt/live/$DOMAIN"

if [[ ! -f "$CERT_DIR/fullchain.pem" ]]; then
  error "Сертификат не получен. Проверьте что домен $DOMAIN указывает на этот сервер."
fi

success "Сертификат получен: $CERT_DIR"

# ── Копируем сертификат в Docker volume ──────────────────────────────────
info "Копируем сертификат в Docker volume nginx_certs..."
docker run --rm \
  -v aqi-platform_nginx_certs:/certs \
  -v "$CERT_DIR:/src:ro" \
  alpine:3.20 \
  sh -c "cp /src/fullchain.pem /certs/cert.pem && cp /src/privkey.pem /certs/key.pem && chmod 600 /certs/key.pem"

success "Сертификат скопирован в volume"

# ── Запускаем Nginx с HTTPS ────────────────────────────────────────────────
cd "$INSTALL_DIR"
docker compose start nginx
sleep 3

# Проверяем
if curl -sf --max-time 5 "https://$DOMAIN/health" &>/dev/null; then
  success "HTTPS работает: https://$DOMAIN"
elif curl -sf --insecure --max-time 5 "https://$DOMAIN/health" &>/dev/null; then
  warn "HTTPS работает но сертификат staging (не доверенный браузером)"
else
  warn "Не удалось проверить HTTPS — проверьте DNS и firewall"
fi

# ── Настройка авто-обновления (cron) ─────────────────────────────────────
setup_autorenew() {
  local RENEW_SCRIPT="$INSTALL_DIR/scripts/renew-tls.sh"

  cat > "$RENEW_SCRIPT" << 'RENEW_EOF'
#!/usr/bin/env bash
# Авто-обновление сертификата Let's Encrypt
set -euo pipefail

DOMAIN="__DOMAIN__"
INSTALL_DIR="__INSTALL_DIR__"
LOG="$INSTALL_DIR/logs/tls-renew.log"

mkdir -p "$(dirname $LOG)"

{
  echo "=== $(date '+%Y-%m-%d %H:%M:%S') ==="

  # Обновляем сертификат (certbot проверяет, осталось ли < 30 дней)
  certbot renew --quiet --non-interactive

  # Если сертификат обновился — копируем в volume
  CERT_DIR="/etc/letsencrypt/live/$DOMAIN"
  if [[ -f "$CERT_DIR/fullchain.pem" ]]; then
    docker run --rm \
      -v aqi-platform_nginx_certs:/certs \
      -v "$CERT_DIR:/src:ro" \
      alpine:3.20 \
      sh -c "cp /src/fullchain.pem /certs/cert.pem && cp /src/privkey.pem /certs/key.pem && chmod 600 /certs/key.pem"

    # Перезагружаем Nginx без остановки (graceful reload)
    cd "$INSTALL_DIR"
    docker compose exec -T nginx nginx -s reload
    echo "OK: сертификат обновлён и Nginx перезагружен"
  else
    echo "OK: сертификат ещё действителен (обновление не требуется)"
  fi

} >> "$LOG" 2>&1
RENEW_EOF

  sed -i "s|__DOMAIN__|$DOMAIN|g" "$RENEW_SCRIPT"
  sed -i "s|__INSTALL_DIR__|$INSTALL_DIR|g" "$RENEW_SCRIPT"
  chmod +x "$RENEW_SCRIPT"

  # Добавляем в cron: каждый день в 3:30 ночи
  CRON_LINE="30 3 * * * root $RENEW_SCRIPT"
  CRON_FILE="/etc/cron.d/aqi-tls-renew"

  echo "$CRON_LINE" > "$CRON_FILE"
  chmod 644 "$CRON_FILE"

  success "Авто-обновление настроено: /etc/cron.d/aqi-tls-renew (каждый день в 3:30)"
  success "Скрипт обновления: $RENEW_SCRIPT"
}

setup_autorenew

echo ""
echo -e "${BOLD}═══════════════════════════════════════════════════${NC}"
success "TLS настроен успешно!"
echo -e "  Домен:         ${BOLD}https://$DOMAIN${NC}"
echo -e "  Сертификат:    ${GREEN}Let's Encrypt${NC} (действует 90 дней)"
echo -e "  Авто-обновление: ${GREEN}cron 3:30 ежедневно${NC}"
echo -e "${BOLD}═══════════════════════════════════════════════════${NC}"
echo ""
