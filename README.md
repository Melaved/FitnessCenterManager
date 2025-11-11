[![ci](https://github.com/Melaved/FitnessCenterManager/actions/workflows/ci.yml/badge.svg)](https://github.com/Melaved/FitnessCenterManager/actions/workflows/ci.yml)
# FitnessCenterManager

CMS (Content Management System) для управления фитнес‑центром на Go (Fiber + PostgreSQL): клиенты, абонементы, зоны/тренеры, дашборд, загрузка фото зон. Проект ориентирован на локальный запуск для учебных/пилотных целей.

## Возможности
- Дашборд со сводной статистикой
- CRUD по клиентам, зонам, тренерам и абонементам
- Загрузка фото для зон (ограничение размера, проверка MIME, ETag/Cache‑Control)
- Шаблоны на Bootstrap 5
- Конфигурация через YAML, секрета отдельно

## Технологии
- Go 1.23
- Fiber v2 + html/template (github.com/gofiber/template/html)
- PostgreSQL (github.com/lib/pq)
- YAML (gopkg.in/yaml.v3)

## Быстрый старт

1) Склонируйте репозиторий и установите Go 1.23.

2) Поднимите PostgreSQL через Docker Compose:

   - Файл `docker-compose.yml` поднимает только БД.
   - Создайте файл `secrets/app_user_password.txt` с паролем пользователя БД (строка без перевода строки предпочтительна).

   Команда запуска:
   - `docker compose up -d`

   По умолчанию БД поднимется с именем `fitness_center`. Публикация порта в compose закомментирована; либо раскомментируйте её, либо подключайтесь из приложения по сети docker (по умолчанию приложение стучится на `127.0.0.1:5432`).

3) Настройте конфиги:

   - `config.yaml` — основной конфиг (хост, порт, пользователь, имя БД, пути к шаблонам/статике/загрузкам, лимиты соединений).
   - `config.secret.yaml` — секреты (пароль БД). Этот файл в `.gitignore` и в репозитории может присутствовать только локально.

   Пример `config.secret.yaml`:
   
   ```yaml
   database:
     password: "ваш_пароль"
   ```

4) Инициализируйте схему БД:

   - В репозитории присутствует `schema.sql` (дамп). Выполните его в вашей БД (psql/GUI). В перспективе рекомендуется перейти на миграции (goose/migrate).

5) Запустите приложение:

- `go run ./cmd/web`
- По умолчанию сервер слушает `:3000` → откройте `http://localhost:3000/`

## Контейнеризация

- Сборка Docker-образа:
  - `docker build -t fitness-center-manager .`

- Запуск через docker-compose (БД + веб):
  - Убедитесь, что файл `secrets/app_user_password.txt` существует и содержит пароль БД.
  - При необходимости отредактируйте `config.docker.example.yaml` (в нём `database.host: db`, пользователь по умолчанию `postgres`).
  - Запуск: `docker compose up -d`
  - Веб будет доступен на `http://localhost:3000/`.
  - В `docker-compose.yml` настроены healthchecks для `db` (pg_isready) и `web` (HTTP GET `/`).

Примечания:
- Конфиги монтируются в контейнер (`config.docker.example.yaml` → `/app/config.yaml`; `config.secret.yaml` → `/app/config.secret.yaml`).
- Загрузки (`web/uploads`) монтируются томом с хоста, чтобы сохранялись между перезапусками.
- Инициализация схемы: положите SQL в папку `init/` (монтируется в `/docker-entrypoint-initdb.d`) — он выполнится при первом старте кластера Postgres. Либо примените дамп `schema.sql` вручную через `psql`.

## CI

В репозитории настроен GitHub Actions (`.github/workflows/ci.yml`):
- Сборка Go-проекта и `go vet` на push/PR в `main/master`.
- Отдельная job для сборки Docker-образа (без публикации).

### Публикация образа (опционально)

Workflow поддерживает публикацию образа при пуше в `main`, если заданы секреты репозитория:
- `REGISTRY` — адрес реестра (например, `ghcr.io` или `docker.io`).
- `REGISTRY_USERNAME` — имя пользователя реестра.
- `REGISTRY_PASSWORD` — токен/пароль.
- `IMAGE_NAME` — имя образа (например, `OWNER/fitness-center-manager`).

При наличии этих секретов job `docker-publish` соберёт и запушит тег `:latest`.

## Makefile (шпаргалка)

- `make run` — запустить приложение локально (`go run ./cmd/web`).
- `make build` — собрать бинарник в `bin/server`.
- `make test` — запустить тесты `go test ./...`.
- `make tidy` / `make vet` / `make fmt` — обслуживание зависимостей и кода.
- `make docker-build` — собрать Docker‑образ (имя по умолчанию `fitness-center-manager:local`, задаётся переменной `IMAGE`).
- `make docker-up` / `make docker-down` / `make docker-logs` — управление `docker compose`.

Пример: `make docker-build IMAGE=ghcr.io/owner/fitness-center-manager:latest`.


## Роуты
- `GET /` — дашборд
- `GET /about` — инфо
- `GET /clients` / `POST /clients` / `GET|PUT|DELETE /clients/:id`
- `GET /subscriptions`
- `GET /trainers` / `GET /trainings` / `GET /equipment`
- Зоны:
  - `GET /zones` — список (HTML)
  - `GET /api/zones/:id` — одна зона (JSON)
  - `POST /zones` — создать (JSON)
  - `PUT /zones/:id` — обновить (JSON)
  - `DELETE /zones/:id` — удалить (JSON)
  - `POST /zones/:id/upload-photo` — загрузить фото (multipart form‑data: `photo`)
  - `DELETE /zones/:id/photo` — очистить фото
  - `GET /zones/:id/photo` — выдача фото (с ETag/Cache‑Control)

## Конфигурация

`config.yaml` (важные поля):
- `database.host/port/user/dbname/sslmode` — параметры подключения. Пароль читается из `config.secret.yaml`.
- `database.max_open_conns/max_idle_conns/conn_max_lifetime_minutes/conn_max_idle_minutes/connect_timeout_seconds` — пул соединений и таймауты пинга.
- `server.port` — порт приложения (например, `:3000`).
- `server.template_path/static_path/upload_path` — пути к шаблонам/статическим/загрузкам.

Примечания к DSN:
- Для TCP используется форма `postgres://user:pass@host:port/dbname?sslmode=...`.
- Если `host` начинается с `/` (Unix‑socket), DSN будет построен в форме `postgres://user:pass@/dbname?host=/var/run/postgresql&port=5432&sslmode=disable`.

## Безопасность и приватность
- В хэндлерах введён таймаут контекста для всех SQL‑вызовов (withDBTimeout, 5s), чтобы защищаться от «зависших» запросов.
- Рекомендуется добавить CSRF‑защиту для форм (если планируете приём данных из браузера вне доверенной среды).

## Разработка
- Код хэндлеров использует `QueryContext/QueryRowContext/ExecContext` с `context.WithTimeout` (см. `internal/handlers/dbctx.go:1`).
- Подключение к БД и пул соединений настраиваются через `internal/database/db.go` и `internal/config/config.go`.

## Ограничения / известные особенности
- Схема БД и названия столбцов/таблиц в примере — на кириллице.

## Типичные проблемы
- "connection refused" к Postgres: проверьте, что контейнер БД запущен, порт доступен, и что `config.yaml` совпадает с настройками.
- Ошибки шаблонов: проверьте `server.template_path`.
- Большие изображения не грузятся: лимит 5 МБ и проверка типа файла (JPEG/PNG/WebP).

## Лицензия
Проект предоставлен как есть для учебных целей.
## Problem Details (RFC 7807)

В проекте включён единый формат ошибок RFC 7807 (application/problem+json). Поле `type` формируется как URI и может быть кликабельным, если задан `server.problem_base_url` в `config.yaml`.

- База URI: значение `server.problem_base_url` (например, `https://fitness-center-manager.dev/problem`)
- Итоговый вид: `{base}/{code}`. Если база не задана — используется `urn:fitness-center-manager:problem:{code}`

Коды ошибок (`{code}`), которые сейчас используются:

- invalid-id — некорректный идентификатор
- invalid-form — неверные данные формы/тела запроса
- missing-required-fields — не заполнены обязательные поля
- invalid-date — неверный формат даты
- invalid-date-range — дата окончания раньше даты начала
- not-found — ресурс не найден
- database-error — внутренняя ошибка БД
- conflict — конфликт операции (например, удаление невозможно из-за связей)
- file-too-large — загружаемый файл слишком большой
- invalid-image-type — недопустимый тип изображения (ожидаются JPEG/PNG/WebP)
- invalid-status — недопустимый статус
- validation-error — общее нарушение валидации (HTTP 400)
- unauthorized — требуется аутентификация (HTTP 401)
- forbidden — нет прав (HTTP 403)
- request-entity-too-large — тело запроса слишком большое (HTTP 413)
- internal-error — внутренняя ошибка сервера (HTTP 5xx)

Шаблоны статических страниц для публикации документации по кодам находятся в каталоге `web/static/problem/`.
