#!/usr/bin/env bash
# =============================================================================
# restore.sh — восстановление базы данных AQI Platform из резервной копии
#
# Использование:
#   bash scripts/restore.sh /backups/aqi_20260801_020000.sql.gz
# =============================================================================

set -euo pipefail

BACKUP_FILE="${1:?Использование: $0 <файл_бэкапа.sql.gz>}"

DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-aqi}"
DB_USER="${DB_USER:-aqi}"
PGPASSWORD="${DB_PASSWORD:?Необходимо задать переменную DB_PASSWORD}"
export PGPASSWORD

if [ ! -f "${BACKUP_FILE}" ]; then
    echo "Ошибка: файл ${BACKUP_FILE} не найден" >&2
    exit 1
fi

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Восстановление из: ${BACKUP_FILE}"
echo "  Цель: ${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
echo ""
echo "ВНИМАНИЕ: Текущие данные в базе будут перезаписаны!"
read -r -p "Продолжить? (введите 'yes' для подтверждения): " CONFIRM

if [ "${CONFIRM}" != "yes" ]; then
    echo "Отменено."
    exit 0
fi

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Восстановление..."

gunzip -c "${BACKUP_FILE}" | psql \
    --host="${DB_HOST}" \
    --port="${DB_PORT}" \
    --username="${DB_USER}" \
    --dbname="${DB_NAME}" \
    --no-password

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Восстановление завершено успешно"
