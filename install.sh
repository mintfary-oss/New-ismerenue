#!/usr/bin/env bash
# =============================================================================
# AQI Platform — установщик одной командой
# Поддерживаемые ОС: Debian 10+, Ubuntu 20.04+, Rocky/AlmaLinux 8+
# Архитектуры: amd64, arm64
#
# Использование (одна команда):
#   curl -fsSL https://raw.githubusercontent.com/mintfary-oss/New-ismerenue/main/install.sh | sudo bash
#
#   — или, скачать и запустить —
#   wget -O install.sh https://raw.githubusercontent.com/mintfary-oss/New-ismerenue/main/install.sh
#   sudo bash install.sh
#
# Без каких-либо облачных хранилищ образов.
# Все образы собираются локально из исходного кода.
# =============================================================================

set -euo pipefail

# ── Цвета вывода ─────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'

info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }
step()    { echo -e "\n${BOLD}━━━ $* ━━━${NC}"; }

# ── Параметры (можно переопределить через переменные окружения) ───────────────
REPO_URL="${REPO_URL:-https://github.com/mintfary-oss/New-ismerenue.git}"
INSTALL_DIR="${INSTALL_DIR:-/opt/aqi-platform}"
SOURCE_DIR="${SOURCE_DIR:-/opt/aqi-source}"

# ── Шапка ─────────────────────────────────────────────────────────────────────
clear 2>/dev/null || true
echo ""
echo -e "${BOLD}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BOLD}║        AQI Platform — Автоматическая установка              ║${NC}"
echo -e "${BOLD}║        Платформа качества атмосферного воздуха               ║${NC}"
echo -e "${BOLD}║        Версия: 1.0  │  Сборка: локальная из исходников       ║${NC}"
echo -e "${BOLD}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  Репозиторий: ${BLUE}${REPO_URL}${NC}"
echo -e "  Установка в: ${BLUE}${INSTALL_DIR}${NC}"
echo ""

# ── Базовые проверки ──────────────────────────────────────────────────────────
[[ $EUID -ne 0 ]] && error "Запустите от имени root:\n  sudo bash install.sh\n  — или —\n  curl -fsSL ${REPO_URL%%.git}/raw/main/install.sh | sudo bash"

ARCH=$(uname -m)
case $ARCH in
  x86_64)  ARCH_TAG="amd64" ;;
  aarch64) ARCH_TAG="arm64" ;;
  *) error "Архитектура $ARCH не поддерживается (нужна amd64 или arm64)" ;;
esac
info "Архитектура: $ARCH ($ARCH_TAG)"

OS_ID=$(grep -oP '(?<=^ID=).+' /etc/os-release 2>/dev/null | tr -d '"' || echo "unknown")
OS_VER=$(grep -oP '(?<=^VERSION_ID=).+' /etc/os-release 2>/dev/null | tr -d '"' || echo "unknown")
info "ОС: $OS_ID $OS_VER"

# ── Шаг 1: Установка системных пакетов ───────────────────────────────────────
step "Шаг 1/7: Системные пакеты"

install_packages() {
  case $OS_ID in
    ubuntu|debian)
      export DEBIAN_FRONTEND=noninteractive
      apt-get update -qq
      apt-get install -y -qq git curl wget openssl ca-certificates gnupg ufw 2>/dev/null || true
      ;;
    centos|rhel|rocky|almalinux|fedora)
      yum install -y git curl wget openssl ca-certificates gnupg2 firewalld 2>/dev/null || \
      dnf install -y git curl wget openssl ca-certificates gnupg2 firewalld 2>/dev/null || true
      ;;
    *)
      warn "Неизвестный дистрибутив. Убедитесь что git, curl, wget, openssl установлены."
      ;;
  esac
}

install_packages
success "Системные пакеты установлены"

# ── Шаг 2: Установка Docker ───────────────────────────────────────────────────
step "Шаг 2/7: Docker"

if command -v docker &>/dev/null; then
  DOCKER_VER=$(docker --version 2>/dev/null | grep -oP '\d+\.\d+\.\d+' | head -1 || echo "?")
  success "Docker $DOCKER_VER уже установлен"
else
  info "Устанавливаю Docker (официальный скрипт)..."
  curl -fsSL https://get.docker.com | bash
  systemctl enable docker
  systemctl start docker
  success "Docker установлен"
fi

# Проверяем Docker Compose плагин
if ! docker compose version &>/dev/null; then
  info "Устанавливаю Docker Compose плагин..."
  case $OS_ID in
    ubuntu|debian)
      apt-get install -y docker-compose-plugin 2>/dev/null || true ;;
    centos|rhel|rocky|almalinux|fedora)
      yum install -y docker-compose-plugin 2>/dev/null || \
      dnf install -y docker-compose-plugin 2>/dev/null || true ;;
  esac
  docker compose version &>/dev/null || error "Не удалось установить Docker Compose плагин"
fi
COMPOSE_VER=$(docker compose version --short 2>/dev/null || echo "?")
success "Docker Compose $COMPOSE_VER доступен"

# ── Шаг 3: Клонирование репозитория ──────────────────────────────────────────
step "Шаг 3/7: Получение исходного кода"

if [[ -d "$SOURCE_DIR/.git" ]]; then
  info "Репозиторий уже существует, обновляю..."
  git -C "$SOURCE_DIR" pull --ff-only 2>/dev/null || {
    warn "Не удалось обновить, использую существующую версию"
  }
  success "Исходный код актуален: $SOURCE_DIR"
else
  info "Клонирую репозиторий в $SOURCE_DIR ..."
  git clone --depth 1 "$REPO_URL" "$SOURCE_DIR"
  success "Исходный код скачан: $SOURCE_DIR"
fi

# ── Шаг 4: Подготовка рабочей директории ─────────────────────────────────────
step "Шаг 4/7: Конфигурация"

mkdir -p \
  "$INSTALL_DIR/docker/nginx/ssl" \
  "$INSTALL_DIR/docker/grafana" \
  "$INSTALL_DIR/scripts" \
  "$INSTALL_DIR/data" \
  "$INSTALL_DIR/logs"

# Копируем конфигурацию Docker из репозитория
cp -r "$SOURCE_DIR/aqi-platform/docker/." "$INSTALL_DIR/docker/"
# Копируем скрипты управления
cp    "$SOURCE_DIR/aqi-platform/scripts/"*.sh "$INSTALL_DIR/scripts/"
chmod +x "$INSTALL_DIR/scripts/"*.sh

success "Конфигурация скопирована в $INSTALL_DIR"

# Генерируем секреты (cryptographically random)
info "Генерирую секреты..."
DB_PASSWORD=$(openssl rand -base64 32 | tr -d '/=+' | cut -c1-32)
REDIS_PASSWORD=$(openssl rand -base64 24 | tr -d '/=+' | cut -c1-24)
JWT_SECRET=$(openssl rand -base64 64 | tr -d '/=+' | cut -c1-64)
GRAFANA_PASSWORD=$(openssl rand -base64 20 | tr -d '/=+' | cut -c1-20)

SERVER_IP=$(hostname -I | awk '{print $1}')

# Создаём .env файл
cat > "$INSTALL_DIR/.env" <<EOF
# AQI Platform — конфигурация
# Создан: $(date '+%Y-%m-%d %H:%M:%S')
# ВАЖНО: не передавайте этот файл посторонним

# Сервер
APP_VERSION=local
BASE_URL=http://${SERVER_IP}
HTTP_PORT=80
HTTPS_PORT=443

# Переменные сборки (заполняются установщиком)
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(git -C "$SOURCE_DIR" rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Базы данных
DB_PASSWORD=${DB_PASSWORD}
REDIS_PASSWORD=${REDIS_PASSWORD}

# Аутентификация
JWT_SECRET=${JWT_SECRET}

# Grafana
GRAFANA_USER=admin
GRAFANA_PASSWORD=${GRAFANA_PASSWORD}

# Бэкапы
BACKUP_KEEP_DAYS=30

# Email (заполните для получения данных с датчиков по IMAP)
IMAP_HOST=
IMAP_USER=
IMAP_PASSWORD=

# SMTP (заполните для отправки уведомлений)
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=

# AQI-алерты
ALERT_ENABLED=false
ALERT_THRESHOLD=101
ALERT_COOLDOWN=4h
EOF
chmod 600 "$INSTALL_DIR/.env"
success "Конфигурация .env создана (права 600)"

# TLS сертификат (самоподписанный для начала работы)
if [[ ! -f "$INSTALL_DIR/docker/nginx/ssl/cert.pem" ]]; then
  info "Создаю самоподписанный TLS-сертификат..."
  openssl req -x509 -nodes -newkey rsa:4096 \
    -keyout "$INSTALL_DIR/docker/nginx/ssl/key.pem" \
    -out    "$INSTALL_DIR/docker/nginx/ssl/cert.pem" \
    -days   365 \
    -subj   "/CN=${SERVER_IP}/O=AQI Platform/C=RU" \
    2>/dev/null
  chmod 600 "$INSTALL_DIR/docker/nginx/ssl/key.pem"
  success "TLS-сертификат создан (действителен 365 дней)"
fi

# ── Шаг 5: Сборка образов из исходников ──────────────────────────────────────
step "Шаг 5/7: Сборка приложения (займёт 3–10 минут)"
info "Собираю Go-бэкенд и React-фронтенд из исходного кода..."
info "Интернет нужен только для загрузки golang:alpine, node:alpine, postgres, redis, nginx с Docker Hub"

# Сборка: Go API + React SPA
docker compose \
  -f "$INSTALL_DIR/docker/docker-compose.yml" \
  --env-file "$INSTALL_DIR/.env" \
  --project-directory "$INSTALL_DIR" \
  build --no-cache 2>&1 | while IFS= read -r line; do
    # Показываем только значимые строки (не спам от npm/go)
    case "$line" in
      *"Step "*|*"COPY"*|*"RUN"*|*"FROM"*|*" ---> "*|*"Successfully"*|*"Error"*|*"error"*)
        echo "    $line" ;;
    esac
  done

success "Образы собраны локально"

# ── Шаг 6: Запуск платформы ───────────────────────────────────────────────────
step "Шаг 6/7: Запуск"

docker compose \
  -f "$INSTALL_DIR/docker/docker-compose.yml" \
  --env-file "$INSTALL_DIR/.env" \
  --project-directory "$INSTALL_DIR" \
  up -d --remove-orphans

success "Контейнеры запущены"

# ── Шаг 7: Автозапуск и watchdog ─────────────────────────────────────────────
step "Шаг 7/7: Автозапуск"

if command -v systemctl &>/dev/null; then
  # Основной сервис — запуск платформы при старте сервера
  cat > /etc/systemd/system/aqi-platform.service <<SYSTEMD_EOF
[Unit]
Description=AQI Platform
Documentation=https://github.com/mintfary-oss/New-ismerenue
Requires=docker.service
After=docker.service network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=${INSTALL_DIR}
ExecStart=/usr/bin/docker compose -f ${INSTALL_DIR}/docker/docker-compose.yml --env-file ${INSTALL_DIR}/.env --project-directory ${INSTALL_DIR} up -d --remove-orphans
ExecStop=/usr/bin/docker compose -f ${INSTALL_DIR}/docker/docker-compose.yml --env-file ${INSTALL_DIR}/.env --project-directory ${INSTALL_DIR} down
Restart=on-failure
RestartSec=30s
TimeoutStartSec=300

[Install]
WantedBy=multi-user.target
SYSTEMD_EOF

  # Watchdog — перезапускает упавшие контейнеры каждые 30 сек
  cat > /etc/systemd/system/aqi-watchdog.service <<SYSTEMD_EOF
[Unit]
Description=AQI Platform Watchdog
After=aqi-platform.service
Requires=aqi-platform.service

[Service]
Type=simple
ExecStart=/bin/bash ${INSTALL_DIR}/scripts/watchdog.sh
Restart=always
RestartSec=10s
StandardOutput=append:/var/log/aqi-watchdog.log
StandardError=append:/var/log/aqi-watchdog.log
Environment="COMPOSE_FILE=${INSTALL_DIR}/docker/docker-compose.yml"
Environment="ENV_FILE=${INSTALL_DIR}/.env"
Environment="PROJECT_DIR=${INSTALL_DIR}"
Environment="LOG_FILE=/var/log/aqi-watchdog.log"
Environment="CHECK_INTERVAL=30"

[Install]
WantedBy=multi-user.target
SYSTEMD_EOF

  # Ротация логов
  cat > /etc/logrotate.d/aqi-platform <<'LOGROTATE_EOF'
/var/log/aqi-watchdog.log {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
}
LOGROTATE_EOF

  systemctl daemon-reload
  systemctl enable aqi-platform.service
  systemctl enable aqi-watchdog.service
  systemctl start  aqi-watchdog.service

  success "Автозапуск настроен (systemd)"
else
  warn "systemd не найден — автозапуск не настроен"
fi

# Удобный псевдоним "aqi"
if ! grep -q 'alias aqi=' /root/.bashrc 2>/dev/null; then
  cat >> /root/.bashrc <<'ALIAS_EOF'

# AQI Platform — управление
alias aqi='docker compose -f /opt/aqi-platform/docker/docker-compose.yml --env-file /opt/aqi-platform/.env --project-directory /opt/aqi-platform'
ALIAS_EOF
fi

# ── Ожидание готовности платформы ────────────────────────────────────────────
echo ""
info "Ожидаю готовности сервисов (до 90 секунд)..."
READY=false
for i in $(seq 1 18); do
  sleep 5
  if curl -sf "http://localhost/health" >/dev/null 2>&1 || \
     curl -sf "http://${SERVER_IP}/health" >/dev/null 2>&1; then
    READY=true
    break
  fi
  echo -n "."
done
echo ""

# ── Итог ──────────────────────────────────────────────────────────────────────
echo ""
if [[ "$READY" == "true" ]]; then
  echo -e "${BOLD}${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
  echo -e "${BOLD}${GREEN}║   ✓  AQI Platform успешно установлена и работает!           ║${NC}"
  echo -e "${BOLD}${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
else
  echo -e "${BOLD}${YELLOW}╔══════════════════════════════════════════════════════════════╗${NC}"
  echo -e "${BOLD}${YELLOW}║   ⚠  Установка завершена, сервисы ещё запускаются...        ║${NC}"
  echo -e "${BOLD}${YELLOW}║      Подождите 1–2 минуты, затем откройте браузер.           ║${NC}"
  echo -e "${BOLD}${YELLOW}╚══════════════════════════════════════════════════════════════╝${NC}"
fi

echo ""
echo -e "  ${BOLD}Доступ к платформе:${NC}"
echo -e "  ┌───────────────────────────────────────────────────────────┐"
echo -e "  │  Платформа:  http://${SERVER_IP}/                              │"
echo -e "  │  Grafana:    http://${SERVER_IP}/grafana/                       │"
echo -e "  │  Swagger UI: http://${SERVER_IP}/api/v1/docs                    │"
echo -e "  │  Виджет:     http://${SERVER_IP}/widget/                        │"
echo -e "  └───────────────────────────────────────────────────────────┘"
echo ""
echo -e "  ${BOLD}Учётные данные:${NC}"
echo -e "  ┌───────────────────────────────────────────────────────────┐"
echo -e "  │  Grafana логин:  admin                                    │"
echo -e "  │  Grafana пароль: ${GRAFANA_PASSWORD}  │"
echo -e "  └───────────────────────────────────────────────────────────┘"
echo -e "  ${RED}  Сохраните пароль! Он больше не будет показан.${NC}"
echo ""
echo -e "  ${BOLD}Управление (после перелогина или source ~/.bashrc):${NC}"
echo -e "  aqi ps              # статус контейнеров"
echo -e "  aqi logs -f app     # логи приложения"
echo -e "  aqi restart app     # перезапуск"
echo -e "  aqi stop / aqi start"
echo ""
echo -e "  ${BOLD}Конфигурация:${NC}  ${INSTALL_DIR}/.env"
echo -e "  ${BOLD}Исходный код:${NC}  ${SOURCE_DIR}"
echo -e "  ${BOLD}Логи watchdog:${NC} /var/log/aqi-watchdog.log"
echo ""
echo -e "  ${BOLD}Следующие шаги:${NC}"
echo -e "  1. Откройте http://${SERVER_IP}/ и создайте учётную запись администратора"
echo -e "  2. Добавьте датчики в разделе Настройки → Датчики"
echo -e "  3. Для HTTPS: sudo bash ${INSTALL_DIR}/scripts/setup-tls.sh"
echo ""
