#!/usr/bin/env bash
# =============================================================================
# AQI Platform — установщик
#
# Для полной автоматической установки используйте корневой install.sh:
#   curl -fsSL https://raw.githubusercontent.com/mintfary-oss/New-ismerenue/main/install.sh | sudo bash
#
# Или запустите из корня репозитория:
#   sudo bash install.sh
#
# Этот скрипт предназначен для ручной установки если репозиторий уже скачан.
# =============================================================================

set -euo pipefail

# Определяем корень репозитория относительно этого скрипта
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Если корневой install.sh существует — запускаем его
if [[ -f "$REPO_ROOT/install.sh" ]]; then
  echo "[INFO] Запускаю основной установщик: $REPO_ROOT/install.sh"
  # Передаём SOURCE_DIR чтобы не клонировать повторно — репо уже скачано
  export SOURCE_DIR="$REPO_ROOT"
  exec bash "$REPO_ROOT/install.sh" "$@"
else
  echo "[ERROR] Не найден корневой install.sh. Скачайте полный репозиторий:"
  echo "  git clone https://github.com/mintfary-oss/New-ismerenue.git"
  exit 1
fi
