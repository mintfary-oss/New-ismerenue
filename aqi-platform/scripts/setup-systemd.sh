#!/usr/bin/env bash
# =============================================================================
# setup-systemd.sh — установка systemd сервисов для AQI Platform
#
# Создаёт:
#   aqi-platform.service  — запуск/остановка Docker Compose
#   aqi-watchdog.service  — сторожевой процесс
#   aqi-watchdog.timer    — запуск watchdog каждые 30 секунд
#   aqi-healthcheck.timer — проверка здоровья каждые 5 минут (лог)
#
# Использование:
#   sudo bash scripts/setup-systemd.sh
# =============================================================================

set -euo pipefail

[[ $EUID -ne 0 ]] && { echo "Запустите от root: sudo bash $0"; exit 1; }

INSTALL_DIR="${INSTALL_DIR:-/opt/aqi-platform}"
SOURCE_DIR="${SOURCE_DIR:-/opt/aqi-source}"

echo "Устанавливаю systemd сервисы..."

# ── 1. Основной сервис платформы ──────────────────────────────────────────────
cat > /etc/systemd/system/aqi-platform.service << EOF
[Unit]
Description=AQI Platform — мониторинг качества атмосферного воздуха
Documentation=https://github.com/mintfary-oss/New-ismerenue
Requires=docker.service
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=${SOURCE_DIR}/aqi-platform

# Исправление прав Redis volume перед запуском.
# После аварийной остановки сервера appendonlydir может получить права root:root,
# что блокирует старт Redis (процесс работает под UID 999).
ExecStartPre=/bin/bash -c "docker run --rm -v aqi-platform_redis_data:/data alpine sh -c 'chmod 777 /data && chown -R 999:999 /data' 2>/dev/null || true"

# Запуск всей платформы
# ВАЖНО: docker compose pull убран — образ aqi-platform:local собирается локально
# и недоступен в DockerHub. Pull падал бы при каждой загрузке сервера.
ExecStart=/usr/bin/docker compose \\
  -f ${SOURCE_DIR}/aqi-platform/docker/docker-compose.yml \\
  --env-file ${INSTALL_DIR}/.env \\
  --project-name aqi-platform \\
  up -d --remove-orphans

# Остановка
ExecStop=/usr/bin/docker compose \\
  -f ${SOURCE_DIR}/aqi-platform/docker/docker-compose.yml \\
  --env-file ${INSTALL_DIR}/.env \\
  --project-name aqi-platform \\
  down

# Перезапуск при сбое
Restart=on-failure
RestartSec=30s
TimeoutStartSec=300
TimeoutStopSec=120

[Install]
WantedBy=multi-user.target
EOF

# ── 2. Watchdog сервис ────────────────────────────────────────────────────────
cat > /etc/systemd/system/aqi-watchdog.service << EOF
[Unit]
Description=AQI Platform Watchdog — сторожевой процесс
After=aqi-platform.service
Requires=aqi-platform.service

[Service]
Type=simple
ExecStart=/bin/bash ${SOURCE_DIR}/aqi-platform/scripts/watchdog.sh
Restart=always
RestartSec=10s
StandardOutput=append:/var/log/aqi-watchdog.log
StandardError=append:/var/log/aqi-watchdog.log

# Переменные окружения
Environment="COMPOSE_FILE=${SOURCE_DIR}/aqi-platform/docker/docker-compose.yml"
Environment="ENV_FILE=${INSTALL_DIR}/.env"
Environment="PROJECT_DIR=${SOURCE_DIR}/aqi-platform"
Environment="LOG_FILE=/var/log/aqi-watchdog.log"
Environment="CHECK_INTERVAL=30"
Environment="MAX_RESTART_ATTEMPTS=3"
Environment="RESTART_COOLDOWN=120"

[Install]
WantedBy=multi-user.target
EOF

# ── 3. Healthcheck таймер (проверка каждые 5 минут → лог) ────────────────────
cat > /etc/systemd/system/aqi-healthcheck.service << EOF
[Unit]
Description=AQI Platform Health Check
After=aqi-platform.service

[Service]
Type=oneshot
ExecStart=/bin/bash ${SOURCE_DIR}/aqi-platform/scripts/healthcheck.sh
StandardOutput=append:/var/log/aqi-healthcheck.log
StandardError=append:/var/log/aqi-healthcheck.log
Environment="COMPOSE_FILE=${SOURCE_DIR}/aqi-platform/docker/docker-compose.yml"
Environment="ENV_FILE=${INSTALL_DIR}/.env"
EOF

cat > /etc/systemd/system/aqi-healthcheck.timer << EOF
[Unit]
Description=AQI Platform Health Check (каждые 5 минут)
After=aqi-platform.service

[Timer]
OnBootSec=2min
OnUnitActiveSec=5min
AccuracySec=30s

[Install]
WantedBy=timers.target
EOF

# ── 4. Ротация логов watchdog ──────────────────────────────────────────────────
cat > /etc/logrotate.d/aqi-platform << EOF
/var/log/aqi-watchdog.log
/var/log/aqi-healthcheck.log
/var/log/aqi-backup.log {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
}
EOF

# ── Активация ────────────────────────────────────────────────────────────────
systemctl daemon-reload

systemctl enable aqi-platform.service
systemctl enable aqi-watchdog.service
systemctl enable aqi-healthcheck.timer

echo ""
echo "Systemd сервисы установлены:"
echo "  aqi-platform.service   — автозапуск Docker Compose при загрузке"
echo "  aqi-watchdog.service   — сторожевой процесс (перезапуск при сбоях)"
echo "  aqi-healthcheck.timer  — проверка здоровья каждые 5 минут"
echo ""
echo "Управление:"
echo "  sudo systemctl start aqi-platform     # запуск"
echo "  sudo systemctl stop aqi-platform      # остановка"
echo "  sudo systemctl restart aqi-platform   # перезапуск"
echo "  sudo systemctl status aqi-watchdog    # статус watchdog"
echo "  sudo journalctl -fu aqi-watchdog      # логи watchdog"
echo "  tail -f /var/log/aqi-watchdog.log     # лог watchdog"
echo ""
echo "Запустить прямо сейчас?"
read -r -p "  [y/N]: " answer
if [[ "${answer,,}" == "y" ]]; then
    systemctl start aqi-platform
    systemctl start aqi-watchdog
    systemctl start aqi-healthcheck.timer
    echo "Запущено!"
fi
