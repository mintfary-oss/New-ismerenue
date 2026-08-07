# GitHub Actions Secrets

Для работы CI/CD необходимо настроить следующие секреты в репозитории:
**Settings → Secrets and variables → Actions → New repository secret**

## Обязательные (для деплоя)

| Секрет | Описание | Пример |
|--------|----------|--------|
| `DEPLOY_HOST` | IP или домен сервера | `192.168.1.100` или `monitor.kemerovo.ru` |
| `DEPLOY_USER` | SSH-пользователь | `deploy` |
| `DEPLOY_SSH_KEY` | Приватный SSH-ключ (содержимое `~/.ssh/id_ed25519`) | `-----BEGIN OPENSSH PRIVATE KEY-----...` |
| `DEPLOY_PORT` | SSH-порт (опционально, default: 22) | `22` |

## Автоматические (не нужно настраивать)

| Переменная | Описание |
|-----------|----------|
| `GITHUB_TOKEN` | Автоматически выдаётся GitHub Actions (для push в GHCR) |

## Настройка SSH-ключа на сервере

```bash
# Генерируем ключ для деплоя
ssh-keygen -t ed25519 -C "deploy@aqi-platform" -f ~/.ssh/aqi_deploy

# Добавляем публичный ключ на сервер
ssh-copy-id -i ~/.ssh/aqi_deploy.pub deploy@YOUR_SERVER

# Содержимое приватного ключа — в GitHub Secret DEPLOY_SSH_KEY
cat ~/.ssh/aqi_deploy
```

## Настройка Environment Protection

В GitHub: **Settings → Environments → production → Required reviewers**

Добавьте себя как обязательного ревьюера — тогда деплой на продакшн
будет требовать ручного подтверждения перед выполнением.
