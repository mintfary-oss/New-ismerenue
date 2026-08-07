#!/usr/bin/env bash
# =============================================================================
# backup.sh — резервное копирование базы данных AQI Platform
#
# Выполняет pg_dump и сохраняет сжатый дамп в /backups/.
# Хранит последние BACKUP_KEEP_DAYS дней (по умолчанию 30).
#
# Использование:
#   # Вручную:
#   bash scripts/backup.sh
#
#   # В Docker Compose (см. docker/docker-compose.yml — сервис backup)
#   docker compose exec backup /scripts/backup.sh
#
#   # Через cron (хост):
#   0 2 * * * /opt/aqi-platform/scripts/backup.sh >> /var/log/aqi-backup.log 2>&1
# =============================================================================

set -euo pipefail

# ── Конфигурация ──────────────────────────────────────────────────────────────

DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-aqi}"
DB_USER="${DB_USER:-aqi}"
PGPASSWORD="${DB_PASSWORD:?Необходимо задать переменную DB_PASSWORD}"
export PGPASSWORD

BACKUP_DIR="${BACKUP_DIR:-/backups}"
BACKUP_KEEP_DAYS="${BACKUP_KEEP_DAYS:-30}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${BACKUP_DIR}/aqi_${TIMESTAMP}.sql.gz"

# ── Основной скрипт ───────────────────────────────────────────────────────────

echo "[$(date '+%Y-%m-%d %H:%M:%S')] AQI Platform — запуск резервного копирования"
echo "  Источник: ${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
echo "  Файл:     ${BACKUP_FILE}"

# Создаём директорию для бэкапов если не существует
mkdir -p "${BACKUP_DIR}"

# pg_dump + gzip: транзакционный дамп в custom формате, затем gzip
if pg_dump \
    --host="${DB_HOST}" \
    --port="${DB_PORT}" \
    --username="${DB_USER}" \
    --dbname="${DB_NAME}" \
    --format=plain \
    --no-password \
    --verbose \
    2>/tmp/pg_dump_stderr.log \
  | gzip -9 > "${BACKUP_FILE}"; then

    SIZE=$(du -sh "${BACKUP_FILE}" | cut -f1)
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Бэкап создан: ${BACKUP_FILE} (${SIZE})"
else
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ОШИБКА pg_dump:" >&2
    cat /tmp/pg_dump_stderr.log >&2
    # Удаляем битый файл
    rm -f "${BACKUP_FILE}"
    exit 1
fi

# ── Удаление старых бэкапов ───────────────────────────────────────────────────

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Удаление бэкапов старше ${BACKUP_KEEP_DAYS} дней..."

DELETED=$(find "${BACKUP_DIR}" \
    -name "aqi_*.sql.gz" \
    -type f \
    -mtime "+${BACKUP_KEEP_DAYS}" \
    -print \
    -delete \
  | wc -l)

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Удалено старых бэкапов: ${DELETED}"

# ── Итог ──────────────────────────────────────────────────────────────────────

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Резервное копирование завершено успешно"

# Список последних бэкапов
echo ""
echo "Актуальные бэкапы в ${BACKUP_DIR}:"
ls -lhrt "${BACKUP_DIR}"/aqi_*.sql.gz 2>/dev/null | tail -10 || echo "  (нет файлов)"
