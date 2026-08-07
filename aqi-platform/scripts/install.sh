#!/usr/bin/env bash
# =============================================================================
# AQI Platform — автоматический установщик
# Поддерживаемые ОС: Debian 12, Ubuntu 22.04+, любой Linux с Docker
# Архитектуры: amd64, arm64
#
# Использование:
#   curl -fsSL https://your-domain.ru/install.sh | sudo bash
#   — или —
#   sudo bash install.sh
# =============================================================================

set -euo pipefail

# ── Цвета для вывода ──────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'

info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# ── Конфигурация ──────────────────────────────────────────────────────────
INSTALL_DIR="${INSTALL_DIR:-/opt/aqi-platform}"
DOCKER_COMPOSE_URL="https://github.com/mintfary/aqi-platform/releases/latest/download/docker-compose.yml"
NGINX_CONF_URL="https://github.com/mintfary/aqi-platform/releases/latest/download/nginx.conf"
APP_VERSION="${APP_VERSION:-latest}"

# ── Заголовок ─────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BOLD}║      AQI Platform — Установщик                       ║${NC}"
echo -e "${BOLD}║      Платформа качества атмосферного воздуха          ║${NC}"
echo -e "${BOLD}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# ── Проверки ──────────────────────────────────────────────────────────────
[[ $EUID -ne 0 ]] && error "Запустите от имени root: sudo bash install.sh"

# Определяем архитектуру
ARCH=$(uname -m)
case $ARCH in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  *)       error "Архитектура $ARCH не поддерживается. Поддерживаются: amd64, arm64" ;;
esac
info "Архитектура: $ARCH"

# Определяем ОС
OS_ID=$(grep -oP '(?<=^ID=).+' /etc/os-release | tr -d '"' 2>/dev/null || echo "unknown")
OS_VER=$(grep -oP '(?<=^VERSION_ID=).+' /etc/os-release | tr -d '"' 2>/dev/null || echo "unknown")
info "ОС: $OS_ID $OS_VER"

# ── Установка Docker ──────────────────────────────────────────────────────
install_docker() {
  if command -v docker &>/dev/null; then
    DOCKER_VER=$(docker --version | grep -oP '\d+\.\d+\.\d+' | head -1)
    success "Docker $DOCKER_VER уже установлен"
    return
  fi

  info "Устанавливаю Docker..."
  curl -fsSL https://get.docker.com | bash
  systemctl enable docker
  systemctl start docker
  success "Docker установлен"
}

# ── Проверка Docker Compose ────────────────────────────────────────────────
check_compose() {
  if docker compose version &>/dev/null; then
    success "Docker Compose плагин доступен"
    return
  fi

  info "Устанавливаю Docker Compose плагин..."
  case $OS_ID in
    ubuntu|debian)
      apt-get install -y docker-compose-plugin >/dev/null 2>&1 || true ;;
    centos|rhel|fedora|rocky|almalinux)
      yum install -y docker-compose-plugin >/dev/null 2>&1 || \
      dnf install -y docker-compose-plugin >/dev/null 2>&1 || true ;;
  esac

  docker compose version &>/dev/null || error "Не удалось установить Docker Compose плагин"
  success "Docker Compose плагин установлен"
}

# ── Генерация секретов ─────────────────────────────────────────────────────
generate_secrets() {
  info "Генерирую криптографически случайные секреты..."

  DB_PASSWORD=$(openssl rand -base64 32 | tr -d '/=+' | cut -c1-32)
  REDIS_PASSWORD=$(openssl rand -base64 24 | tr -d '/=+' | cut -c1-24)
  # JWT секрет: минимум 32 символа по требованию конфигурации
  JWT_SECRET=$(openssl rand -base64 64 | tr -d '/=+' | cut -c1-64)
  # Grafana admin password
  GRAFANA_PASSWORD=$(openssl rand -base64 20 | tr -d '/=+' | cut -c1-20)

  success "Секреты сгенерированы"
}

# ── Создание директории и файлов ──────────────────────────────────────────
setup_files() {
  info "Создаю директорию $INSTALL_DIR..."
  mkdir -p "$INSTALL_DIR"/{docker/nginx,data,logs}
  cd "$INSTALL_DIR"

  # .env файл с правами только для root
  cat > .env <<EOF
# AQI Platform — конфигурация (автогенерировано установщиком)
# ВНИМАНИЕ: не передавайте этот файл посторонним лицам

APP_VERSION=${APP_VERSION}
BASE_URL=http://$(hostname -I | awk '{print $1}')
HTTP_PORT=80
HTTPS_PORT=443

DB_PASSWORD=${DB_PASSWORD}
REDIS_PASSWORD=${REDIS_PASSWORD}
JWT_SECRET=${JWT_SECRET}

# Grafana
GRAFANA_USER=admin
GRAFANA_PASSWORD=${GRAFANA_PASSWORD}

# Заполните для email-интеграции:
IMAP_HOST=
IMAP_USER=
IMAP_PASSWORD=
EOF
  chmod 600 .env
  success "Файл .env создан с правами 600"

  # Скачиваем или копируем docker-compose.yml
  if [[ -f "/tmp/docker-compose.yml" ]]; then
    cp /tmp/docker-compose.yml docker/docker-compose.yml
  else
    info "Скачиваю docker-compose.yml..."
    # При локальной установке из репозитория — копируем
    if [[ -d "/workspace/New-ismerenue/aqi-platform/docker" ]]; then
      cp /workspace/New-ismerenue/aqi-platform/docker/docker-compose.yml docker/docker-compose.yml
      cp /workspace/New-ismerenue/aqi-platform/docker/nginx/nginx.conf docker/nginx/nginx.conf
    else
      # В продакшне — скачиваем с GitHub Releases
      curl -fsSL "$DOCKER_COMPOSE_URL" -o docker/docker-compose.yml || \
        warn "Не удалось скачать docker-compose.yml — создайте вручную"
      curl -fsSL "$NGINX_CONF_URL" -o docker/nginx/nginx.conf || \
        warn "Не удалось скачать nginx.conf — создайте вручную"
    fi
  fi

  # Самоподписанный TLS сертификат (если нет настоящего)
  if [[ ! -f "docker/nginx/ssl/cert.pem" ]]; then
    mkdir -p docker/nginx/ssl
    info "Генерирую самоподписанный TLS сертификат..."
    openssl req -x509 -nodes -newkey rsa:4096 \
      -keyout docker/nginx/ssl/key.pem \
      -out    docker/nginx/ssl/cert.pem \
      -days   365 \
      -subj   "/CN=$(hostname -I | awk '{print $1}')/O=AQI Platform" \
      2>/dev/null
    chmod 600 docker/nginx/ssl/key.pem
    success "Самоподписанный сертификат создан (замените на настоящий для продакшна)"
  fi
}

# ── Запуск платформы ──────────────────────────────────────────────────────
start_platform() {
  info "Скачиваю Docker-образы..."
  docker compose -f "$INSTALL_DIR/docker/docker-compose.yml" \
    --env-file "$INSTALL_DIR/.env" \
    --project-directory "$INSTALL_DIR" \
    pull

  info "Запускаю платформу..."
  docker compose -f "$INSTALL_DIR/docker/docker-compose.yml" \
    --env-file "$INSTALL_DIR/.env" \
    --project-directory "$INSTALL_DIR" \
    up -d

  success "Платформа запущена"
}

# ── Проверка здоровья ─────────────────────────────────────────────────────
health_check() {
  info "Ожидаю готовности сервисов (до 60 секунд)..."
  local ip
  ip=$(hostname -I | awk '{print $1}')
  local attempts=0

  while [[ $attempts -lt 12 ]]; do
    if curl -sf "http://$ip/health" >/dev/null 2>&1; then
      success "Платформа отвечает на запросы"
      return 0
    fi
    sleep 5
    ((attempts++)) || true
  done

  warn "Платформа не ответила за 60 секунд. Проверьте логи: docker compose logs"
}

# ── Вывод итогов ──────────────────────────────────────────────────────────
print_summary() {
  local ip
  ip=$(hostname -I | awk '{print $1}')

  echo ""
  echo -e "${BOLD}${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
  echo -e "${BOLD}${GREEN}║   ✓  AQI Platform успешно установлена!               ║${NC}"
  echo -e "${BOLD}${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
  echo ""
  echo -e "  ${BOLD}Адрес платформы:${NC}  http://$ip"
  echo -e "  ${BOLD}Grafana (мониторинг):${NC}  http://$ip/grafana/  (login: admin / ${GRAFANA_PASSWORD})"
  echo -e "  ${BOLD}Установочная директория:${NC} $INSTALL_DIR"
  echo -e "  ${BOLD}Логи:${NC}  docker compose -f $INSTALL_DIR/docker/docker-compose.yml logs -f"
  echo ""
  echo -e "  ${YELLOW}Следующие шаги:${NC}"
  echo -e "  1. Войдите на http://$ip и смените пароль администратора"
  echo -e "  2. Добавьте датчики в разделе Настройки → Датчики"
  echo -e "  3. Откройте Grafana на http://$ip/grafana/ для мониторинга метрик"
  echo -e "  4. Получите TLS сертификат (Let's Encrypt) для HTTPS"
  echo ""
  echo -e "  ${BOLD}Конфигурация хранится в:${NC} $INSTALL_DIR/.env"
  echo -e "  ${RED}ВАЖНО: не передавайте .env файл посторонним лицам!${NC}"
  echo ""
}

# ── Установка systemd сервисов (watchdog + автозапуск) ────────────────────
setup_systemd() {
  if ! command -v systemctl &>/dev/null; then
    warn "systemd не найден — автозапуск не настроен"
    return
  fi

  info "Устанавливаю systemd сервисы..."

  # Основной сервис
  cat > /etc/systemd/system/aqi-platform.service << EOF
[Unit]
Description=AQI Platform
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
EOF

  # Watchdog сервис
  cat > /etc/systemd/system/aqi-watchdog.service << EOF
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
EOF

  # Healthcheck таймер
  cat > /etc/systemd/system/aqi-healthcheck.service << EOF
[Unit]
Description=AQI Platform Health Check
After=aqi-platform.service

[Service]
Type=oneshot
ExecStart=/bin/bash ${INSTALL_DIR}/scripts/healthcheck.sh
StandardOutput=append:/var/log/aqi-healthcheck.log
StandardError=append:/var/log/aqi-healthcheck.log
Environment="ENV_FILE=${INSTALL_DIR}/.env"
EOF

  cat > /etc/systemd/system/aqi-healthcheck.timer << EOF
[Unit]
Description=AQI Platform Health Check Timer
After=aqi-platform.service

[Timer]
OnBootSec=2min
OnUnitActiveSec=5min

[Install]
WantedBy=timers.target
EOF

  # Ротация логов
  cat > /etc/logrotate.d/aqi-platform << EOF
/var/log/aqi-watchdog.log /var/log/aqi-healthcheck.log {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
}
EOF

  # Копируем скрипты
  cp "$(dirname "$0")/watchdog.sh"    "$INSTALL_DIR/scripts/watchdog.sh"
  cp "$(dirname "$0")/healthcheck.sh" "$INSTALL_DIR/scripts/healthcheck.sh"
  chmod +x "$INSTALL_DIR/scripts/watchdog.sh" "$INSTALL_DIR/scripts/healthcheck.sh"

  systemctl daemon-reload
  systemctl enable aqi-platform.service
  systemctl enable aqi-watchdog.service
  systemctl enable aqi-healthcheck.timer

  success "Systemd сервисы установлены (watchdog + автозапуск)"
}

# ── Основной поток выполнения ─────────────────────────────────────────────
main() {
  install_docker
  check_compose
  generate_secrets
  setup_files
  start_platform
  health_check
  setup_systemd
  print_summary
}

main "$@"
