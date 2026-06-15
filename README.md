# Momo Store

Интернет-магазин пельменей: Go API и Vue.js SPA.

## Архитектура

```text
Browser
  |-- :80   -> frontend (nginx, Vue.js)
  `-- :8081 -> nginx-lb -> backend x 2
                         `-> order-data
```

- `frontend` доступен на порту `80`.
- `nginx-lb` доступен на порту `8081` и балансирует запросы между репликами API.
- `backend` доступен только во внутренней сети Docker.
- `order-data` хранит общий счётчик заказов и исключает дублирование ID между репликами.

Фронтенд и API находятся в разных сетях. Сеть API помечена как `internal`; балансировщик дополнительно подключён к внешней edge-сети.

## Запуск

Требуются Docker Engine 24+ и Docker Compose v2.

```bash
docker compose up -d --build --wait
```

По умолчанию для локального запуска используется публичный тестовый secret из `secrets/order_id_secret.example`. Для окружения, где ID заказов должны быть защищены, создайте secret вне репозитория:

```bash
mkdir -p "$HOME/.config/momo-store"
openssl rand -hex 32 > "$HOME/.config/momo-store/order_id_secret.txt"
ORDER_ID_SECRET_FILE="$HOME/.config/momo-store/order_id_secret.txt" \
  docker compose up -d --build --wait
```

Адреса:

- фронтенд: http://localhost/momo-store/
- API: http://localhost:8081
- healthcheck API: http://localhost:8081/health

Проверка:

```bash
curl --fail http://localhost/
curl --fail http://localhost:8081/health
curl --fail http://localhost:8081/products
docker compose ps
```

Остановка:

```bash
docker compose down
```

Удаление сохранённых данных:

```bash
docker compose down -v
```

## Development

Локальный override включает debug-логи, одну реплику API и увеличенные лимиты:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build --wait
```

## Масштабирование

По умолчанию запускаются две реплики API. Число реплик можно изменить:

```bash
docker compose up -d --scale backend=4 --wait
```

nginx повторно разрешает имя `backend` через Docker DNS. Общий volume и файловая блокировка обеспечивают корректную выдачу ID заказов при нескольких репликах.

## Конфигурация

Основные переменные окружения перечислены в `.env.example`:

| Переменная | Назначение | Значение по умолчанию |
|---|---|---|
| `VUE_APP_API_URL` | URL API, встраиваемый при сборке SPA | `http://localhost:8081` |
| `FRONTEND_PORT` | Порт фронтенда на хосте | `80` |
| `BACKEND_PORT` | Порт балансировщика на хосте | `8081` |
| `LOG_LEVEL` | Уровень логирования API | `info` |
| `IMAGE_TAG` | Тег локальных образов | `1.0.0` |
| `FRONTEND_IMAGE` | Имя образа фронтенда | `momo-store-frontend` |
| `BACKEND_IMAGE` | Имя образа бэкенда | `momo-store-backend` |
| `NODE_IMAGE` | Builder-образ фронтенда | `node:22-alpine3.23` |
| `GO_IMAGE` | Builder-образ бэкенда | `golang:1.26.4-alpine3.23` |
| `NGINX_IMAGE` | Непривилегированный runtime-образ nginx | `nginxinc/nginx-unprivileged:1.29.4-alpine` |
| `ORDER_ID_SECRET_FILE` | Локальный файл Docker Secret | `./secrets/order_id_secret.example` |

Build arguments можно передать напрямую:

```bash
docker build --build-arg GO_IMAGE=golang:1.26.4-alpine3.23 -t momo-store-backend:1.0.0 backend
docker build \
  --build-arg NODE_IMAGE=node:22-alpine3.23 \
  --build-arg NGINX_IMAGE=nginxinc/nginx-unprivileged:1.29.4-alpine \
  --build-arg VUE_APP_API_URL=http://localhost:8081 \
  -t momo-store-frontend:1.0.0 frontend
```

## Образы

Оба Dockerfile используют multi-stage build. Бэкенд запускается из `scratch`, а фронтенд и балансировщик — из обновлённого Alpine-образа nginx без root. Компиляторы, npm, исходники и build-кэш в финальные образы не попадают.

Фактические размеры после локальной сборки:

| Образ | Размер |
|---|---:|
| `momo-store-backend:1.0.0` | 4.2 MB |
| `momo-store-frontend:1.0.0` | 32.5 MB |

Проверка размеров:

```bash
docker image inspect \
  momo-store-backend:1.0.0 \
  momo-store-frontend:1.0.0 \
  --format '{{index .RepoTags 0}} {{.Size}}'
```

## Безопасность

- Все контейнеры работают от фиксированных непривилегированных UID.
- Корневые файловые системы доступны только для чтения.
- Все Linux capabilities сброшены.
- Включён `no-new-privileges`.
- Для сервисов заданы лимиты CPU и памяти.
- Открыты только порты `80` и `8081`.
- Рабочий secret хранится вне Git и монтируется в `/run/secrets/order_id_secret`; в репозитории есть только тестовый пример.
- Для данных используется отдельный named volume.
- Дополнительный CI job завершается ошибкой при `HIGH` или `CRITICAL` уязвимостях с доступным исправлением.

## CI/CD

Workflow `.github/workflows/deploy.yaml`:

1. Проверяет Compose-конфигурацию.
2. Собирает оба образа.
3. Сканирует их Trivy.
4. Запускает Compose и выполняет smoke-тесты.
5. Сохраняет исходные jobs и steps базового пайплайна без изменений.
