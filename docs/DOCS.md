# Distributed Crawler — подробный обзор проекта

## 1) Назначение и общий смысл
Проект — распределённая система веб‑краулинга и парсинга страниц. Основная идея: отделить **fetch** (скачивание страницы) от **parse** (извлечение данных), хранить сырые HTML в MinIO/S3, результаты парсинга — в S3‑объектах и метаданных в PostgreSQL. Дополнительно есть:
- outbox‑механизм для надёжной постановки задач в RabbitMQ;
- планировщик (cron‑schedule) для периодических запусков;
- экспорт результатов на уровне job (JSON/CSV);
- превью/санитизация HTML для UI‑инспектора;
- мониторинг воркеров через gRPC stream.

## 2) Архитектура и поток данных (backend)
**Основной поток (2‑стадийный pipeline):**
1. **Job/Task создаются** через API → сохраняются в PostgreSQL → для seed‑URL создаются задачи (`crawl_tasks`) + события в outbox (`crawl_task_outbox`).
2. **OutboxPublisher** (живёт внутри gRPC сервера) публикует события в **RabbitMQ** (queue `crawl_queue`).
3. **FetchWorker** читает `crawl_queue` → проверяет scope/robots.txt/ratelimit → скачивает страницу → сохраняет HTML в MinIO → пишет `body_hash`, `minio_object_key` в Postgres → публикует `ParsingTaskMessage` в `parsing_queue`.
4. **ParserWorker** читает `parsing_queue` → берёт HTML из MinIO → извлекает данные по DSL (ExtractionSpec) → сохраняет результат JSON в MinIO (prefix `results/`) → пишет ссылку на результат в `crawl_tasks` →
   - выполняет **pagination extraction** по пользовательским селекторам,
   - выполняет **автоматическое обнаружение ссылок**,
   - новые ссылки → новые задачи + outbox‑события.
5. **ExportWorker** периодически ищет завершённые jobs → собирает результаты из MinIO → выгружает `exports/jobs/{job_id}/report.json` и `report.csv` → отмечает job как экспортированный.

**Планировщик:**
- **SchedulerWorker** периодически сканирует конфиги с `schedule.cron`, создаёт job и задачи (через outbox), учитывает `last_run_at/next_run_at`.

**Превью для UI:**
- **PreviewService** скачивает HTML, санитизирует, сохраняет в MinIO (prefix `previews/`), возвращает presigned URL.

**Мониторинг воркеров:**
- Воркеры держат gRPC stream и шлют heartbeats в WorkerService, сервер хранит статус/uptime в памяти.

## 3) Технологический стек
### Backend
- **Go 1.25.x**
- **gRPC + grpc-gateway** (REST поверх gRPC)
- **PostgreSQL** (migrations через goose)
- **RabbitMQ** (очереди `crawl_queue`, `parsing_queue`)
- **MinIO (S3‑compatible)** для HTML, результатов и экспортов
- **Redis** (ratelimit, robots.txt cache)
- **JWT auth** (access/refresh tokens, RBAC роли)
- **goquery** (DOM парсинг, CSS‑селекторы)
- **chromedp** (используется в браузерном fetcher)
- **zap** (логирование)
- **buf / protoc / statik** (генерация gRPC/Swagger)

### Frontend
- **Angular 19**
- **PrimeNG + PrimeIcons + Prime UIX themes**
- **TailwindCSS**
- **RxJS / Zone.js**

### Infra
- Dockerfiles для каждого сервиса
- Helm chart для Kubernetes

## 4) Репозиторий: ключевые папки
- `cmd/` — entrypoints:
  - `grpc_server/` — запуск API (gRPC + HTTP gateway) + outbox publisher.
  - `fetch_worker/`, `parser_worker/`, `export_worker/`, `scheduler_worker/` — воркеры.
  - `grpc_service/`, `http_server/`, `http_client/` — дополнительные утилиты/черновые бинарники.
- `internal/` — clean architecture:
  - `api/` — gRPC handlers (auth, crawl_job, preview, user, worker).
  - `application/` — сервисный слой.
  - `domain/` — модели, valueobjects, сервисы, события, репозитории.
  - `infra/` — Postgres, RabbitMQ, Redis, MinIO, fetchers, sanitizer.
  - `worker/` — реализация воркеров.
  - `workerhealth/` — in‑memory registry состояния воркеров.
- `api/v1/` — protobuf + swagger.
- `ui/` — Angular приложение.
- `docker/` — Dockerfile’ы под каждый компонент.
- `deploy/helm/` — Helm chart.
- `docs/` — проектные спецификации и заметки.
- `statik/` — встроенные swagger‑артефакты.

## 5) Backend: сервисы и процессы
### API (gRPC + HTTP gateway)
Entry: `cmd/grpc_server/main.go` → `internal/app/api_app.go`.
- Поднимает gRPC сервер + HTTP gateway.
- Включает interceptors: логирование, validate, JWT auth, RBAC.
- CORS настроен под UI (localhost:4200).
- Запускает OutboxPublisher как фонового воркера.

### Workers
Entry: `internal/app/worker_app.go` + `cmd/*_worker/main.go`.
- **FetchWorker** — `internal/worker/fetch_worker.go`.
- **ParserWorker** — `internal/worker/parser_worker.go`.
- **ExportWorker** — `internal/worker/export_worker.go`.
- **SchedulerWorker** — `internal/worker/scheduler_worker.go`.
- **OutboxPublisher** — `internal/worker/outbox_publisher.go` (не отдельный процесс).

### Worker Monitoring
- **WorkerMonitor** (client) — `internal/worker/monitoring.go`.
- **Worker registry** (server) — `internal/workerhealth/registry.go`.

## 6) API (proto → REST)
Определено в `api/v1/service.proto` + `api/v1/models.proto`.

### Jobs
- `GET /api/v1/jobs` — список job’ов (cursor pagination + фильтры).
- `POST /api/v1/jobs` — создать job (CrawlJobConfig).
- `GET /api/v1/jobs/{id}` — получить job.
- `GET /api/v1/jobs/{job_id}/export-url` — presigned URL экспорта (json/csv).

### Tasks
- `GET /api/v1/tasks/{id}` — задача.
- `GET /api/v1/jobs/{job_id}/tasks` — задачи job’а.
- `GET /api/v1/tasks/{task_id}/file-url` — presigned URL (HTML или result JSON).

### Preview
- `POST /api/v1/previews` — создать HTML preview.
- `GET /api/v1/previews/{id}` — получить preview + presigned URL.

### Auth
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`

### Users (RBAC)
- `GET /api/v1/users` — список пользователей.
- `PATCH /api/v1/users/{id}/role` — изменение роли.

### Workers (monitoring/control)
- `rpc WorkerStream` — bidirectional stream (heartbeat/commands).
- `GET /api/v1/workers` — список воркеров.
- `POST /api/v1/workers/{worker_id}:drain` — graceful drain.
- `POST /api/v1/workers/{worker_id}:force-kill` — force kill.

## 7) Конфигурация и переменные окружения
Основные `.env` и `.worker.env` содержат:
- PG DSN + миграции
- RabbitMQ URL + queue names
- MinIO endpoint/credentials/bucket
- HTTP/GRPC host/port
- Redis address/pwd/db
- JWT секрет и TTL
- default admin (email/password)

## 8) Хранилище данных (PostgreSQL)
Миграции: `internal/infra/persistence/postgres/migrations/`.
Ключевые таблицы:
- `crawl_job_configs` — конфигурации (extraction_spec, scopes, seeds, retries, schedule, auth).
- `crawl_jobs` — запуски job’ов.
- `crawl_tasks` — отдельные URL‑задачи.
- `crawl_task_outbox` — transactional outbox.
- `previews` — метаданные preview.
- `users`, `refresh_tokens` — auth.
- дополнительные поля для export (`export_status`, `export_json_key`, `export_csv_key`, `exported_at`).
- уникальный индекс для дедупликации по `body_hash` внутри job.

## 9) MinIO / S3 объекты
**ContentStore** (`internal/infra/services/contentstore/minio_store.go`) хранит:
- HTML страниц: `pages/{job_id}/{task_id}.html`
- Результаты парсинга: `results/tasks/{task_id}.json`
- Экспорты: `exports/jobs/{job_id}/report.json` и `report.csv`
- Превью: `previews/{preview_id}.html`

## 10) Очереди и outbox
- RabbitMQ queue для fetch (`crawl_queue`) и parsing (`parsing_queue`).
- Новые задачи создаются через **outbox** → публикуются **OutboxPublisher**.
- Воркер Parser создаёт новые задачи (pagination + discovery) тоже через outbox.

## 11) Парсер и Extraction DSL
DSL описан в `PARSING.md` и в `api/v1/models.proto`:
- **FieldSpec**: name, type, required, extractor, transforms.
- **ExtractorSpec**: selector (CSS), attribute (text/html/href/src/etc.), multiple, index.
- **TransformSpec**: trim/lower/upper/normalize_url/unique/limit/parse_int/parse_float/parse_price/html_to_text/collapse_ws/sha256.
- **MetricSpec**: len/count/word_count/field_present/count_external_links.
- **PaginationSpec**: selector+attribute для извлечения next‑page ссылок.

ParserWorker (`internal/worker/parser_worker.go`) реализует:
- извлечение данных (goquery),
- вычисление метрик,
- дедупликацию по `body_hash`,
- сохранение результата в S3 + ссылок в БД,
- pagination extraction,
- link discovery + outbox enqueue,
- robots.txt check + scope validation.

## 12) Fetcher и правила доступа
FetchWorker (`internal/worker/fetch_worker.go`) использует:
- **scope validation** (domains, depth, deny patterns),
- **robots.txt** (кэш в Redis),
- **rate limiting** (Redis),
- retry policy.

Fetcher реализован в `internal/infra/services/fetcher/`:
- HTTP fetcher
- Browser fetcher (chromedp)

## 13) Экспорт результатов
ExportWorker (`internal/worker/export_worker.go`):
- ищет jobs с `export_status=NOT_STARTED` и `completed_at != null`,
- собирает результаты из MinIO,
- генерирует JSON и CSV,
- сохраняет `export_*` поля в `crawl_jobs`.

## 14) Планировщик
SchedulerWorker (`internal/worker/scheduler_worker.go`):
- читает cron‑выражения из `CrawlJobConfig.schedule`,
- создаёт jobs и задачи, выставляет `next_run_at/last_run_at`.

## 15) Превью HTML
PreviewService (API + сервисный слой):
- загружает HTML, санитизирует, сохраняет в S3,
- отдаёт presigned URL для UI инспектора.

## 16) Auth и RBAC
- JWT access/refresh tokens.
- Пользователи + роли (`READ`, `READ_WRITE`, `ADMINISTRATOR`).
- Interceptors: JWT validation + RBAC.
- Auto‑создание default admin при старте API.

## 17) Worker Monitoring
- Воркеры шлют heartbeats каждые ~4 секунды.
- Registry определяет INACTIVE при отсутствии heartbeat.
- Есть команды Drain / ForceKill.

## 18) Frontend (Angular)
Роуты: `ui/src/app/app.routes.ts`:
- `/auth/login`, `/auth/register`
- `/jobs` (список)
- `/jobs/create` и `/jobs/simple-create`
- `/jobs/:id` (детали job + tasks)
- `/workers` (мониторинг воркеров)
- `/users` (управление пользователями)

Структура UI:
- `ui/src/app/features/jobs` — list
- `ui/src/app/features/job-details` — job detail + tasks
- `ui/src/app/features/job-create` — визуальный билдер конфигурации
- `ui/src/app/features/auth` — login/register
- `ui/src/app/features/workers` — мониторинг
- `ui/src/app/features/users` — управление ролями

## 19) Docker и Kubernetes
### Docker
- `docker/grpc_server/Dockerfile`
- `docker/fetch_worker/Dockerfile`
- `docker/parser_worker/Dockerfile`
- `docker/export_worker/Dockerfile`

Сборка через `make docker-build APP=...`.

### Helm
- `deploy/helm/distributed-crawler/` — Helm chart.
- templates для каждого сервиса + ConfigMap/Secret/PDB.
- `values.yaml`, `values-dev.yaml`, `values-prod.yaml`.

## 20) Сборка, генерация, тесты
Makefile ключевые цели:
- `make generate` — buf + swagger statik.
- `make run-grpc-server` — API + outbox publisher.
- `make run-fetcher`, `make run-parser`, `make run-export`.
- `make local-migration-up` — goose migrations.
- `make test` / `make test-coverage`.

## 21) Документация в проекте
Ключевые файлы:
- `README.md` — обзор, use cases.
- `PIPELINE_ARCHITECTURE.md` — детальная схема fetch/parse с MinIO.
- `PARSING.md` — детальная спецификация DSL.
- `docs/` — дополнительные спецификации: export system, preview, pagination, auth, dedup, worker healthchecks и т.д.

Примечание: часть `docs/*.txt|md` — это **технические задания/спеки** (описание желаемого поведения), а не факт текущей реализации. Реализация подтверждается кодом в `internal/`.

---

Если нужно, могу дополнительно:
- составить диаграмму потоков (mermaid),
- сделать карту зависимостей сервисов,
- добавить краткое “How to run locally” на основе текущего `.env`.
