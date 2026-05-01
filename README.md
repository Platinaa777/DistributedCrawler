# Distributed Crawler

Distributed Crawler — распределённая платформа для веб-краулинга на Go с gRPC API, HTTP gateway, Angular UI, набором воркеров и инфраструктурой для Docker Compose и Kubernetes.

Система создаёт crawl jobs, загружает страницы, сохраняет HTML/артефакты в MinIO, извлекает данные по DSL проекта, строит follow-up задачи, ведёт пользователей и роли, отдаёт превью страниц и формирует JSON/CSV экспорт.

Главный эксплуатационный runbook находится в [docs/operator-manual.md](docs/operator-manual.md). Синтаксис extraction DSL описан в [docs/parsing-syntax-spec.md](docs/parsing-syntax-spec.md), а multi-node Kubernetes affinity для региональных fetch-worker — в [docs/multiregion-affinity-spec.md](docs/multiregion-affinity-spec.md).

## Возможности

- gRPC API и HTTP gateway со Swagger UI.
- Angular Admin UI для jobs, tasks, workers, users и создания crawl jobs.
- Авторизация через JWT, refresh tokens, роли пользователей и seed admin.
- Preview API для загрузки страницы перед настройкой extraction spec.
- Fetch pipeline с scope rules, robots.txt, retry policy, rate limiting и разными fetcher backend (`http`, `browser`, `selenium`).
- Parser pipeline с extraction DSL, пагинацией, дочерними ссылками и сохранением результатов.
- Export worker, который собирает результаты завершённых jobs в JSON/CSV.
- RabbitMQ, Kafka или gRPC in-memory broker.
- PostgreSQL для метаданных, MinIO для страниц/экспортов, Redis для rate limiting и кэша.
- Observability через OpenTelemetry Collector, Jaeger, Prometheus, Grafana и OpenSearch.
- Docker Compose, Minikube/Helm и multi-region запуск fetch-worker.

## Структура проекта

| Путь | Назначение |
|---|---|
| `cmd/` | Точки входа: `grpc_server`, `fetch_worker`, `parser_worker`, `export_worker`, `scheduler_worker`, `memory_broker`, `migrate` |
| `internal/api/` | gRPC/HTTP application handlers: auth, jobs, preview, users, workers |
| `internal/app/` | Сборка API/worker приложений и wiring зависимостей |
| `internal/application/` | Use cases и application services |
| `internal/domain/` | Доменные модели, value objects, events и repo interfaces |
| `internal/infra/` | PostgreSQL, RabbitMQ/Kafka/memory messaging, MinIO, Redis, fetchers, logger |
| `internal/interfaces/http/` | Дополнительные HTTP handlers, DTO и middleware |
| `internal/worker/` | Fetch, parser, export, scheduler, outbox publisher, worker monitoring |
| `internal/config/` | Env/dotenv конфигурация |
| `api/v1/` | Protobuf definitions и Swagger assets |
| `pkg/v1/` | Сгенерированный Go-код protobuf/gRPC/gateway/validation |
| `ui/` | Angular frontend |
| `docker/` | Dockerfiles компонентов |
| `deploy/scripts/` | Launcher-скрипты для local, Docker и Kubernetes |
| `deploy/helm/` | Helm charts приложения и инфраструктуры |
| `docs/` | Operator manual, DSL spec, examples, multi-region guide |
| `observability/`, `grafana/` | Конфиги OTel Collector, Prometheus, Grafana dashboards/provisioning |

## Компоненты

```mermaid
flowchart LR
    UI[Angular UI] --> API[gRPC server / HTTP gateway]
    Client[API client] --> API
    API --> PG[(PostgreSQL)]
    API --> OUTBOX[Outbox publisher]
    OUTBOX --> MQ[Broker]
    MQ --> FETCH[Fetch worker]
    FETCH --> MINIO[(MinIO)]
    FETCH --> MQ
    MQ --> PARSER[Parser worker]
    PARSER --> PG
    PARSER --> MINIO
    PARSER --> MQ
    PG --> EXPORT[Export worker]
    MINIO --> EXPORT
    EXPORT --> MINIO
    WORKERS[Workers] --> API
```

| Компонент | Точка входа | Роль |
|---|---|---|
| `grpc-server` | `cmd/grpc_server` | gRPC `:8083`, HTTP gateway `:8084`, Swagger, auth, jobs, preview, users, workers, outbox publisher и встроенный scheduler loop |
| `fetch-worker` | `cmd/fetch_worker` | Забирает crawl tasks, проверяет scope/robots/rate limits, скачивает страницы и публикует parsing tasks |
| `parser-worker` | `cmd/parser_worker` | Читает HTML из MinIO, извлекает records, создаёт follow-up tasks и обновляет статусы |
| `export-worker` | `cmd/export_worker` | Собирает результаты завершённых jobs и пишет JSON/CSV exports в MinIO |
| `scheduler-worker` | `cmd/scheduler_worker` | Отдельный scheduler процесс, если расписание нужно вынести из API |
| `memory-broker` | `cmd/memory_broker` | gRPC in-memory broker для разработки и тестов |
| `ui` | `ui/`, `docker/ui` | Angular SPA, отдаётся через nginx container |

## Быстрый запуск

Верхнеуровневые launcher-скрипты находятся в `deploy/scripts/` и описаны подробнее в [operator manual](docs/operator-manual.md#5-быстрый-запуск-с-помощью-launcher-скриптов).

```bash
# Локальные процессы + infra из docker-compose.yaml
./deploy/scripts/default_run.sh

# Полный стек в Docker Compose
./deploy/scripts/default_run.sh --mode docker

# Minikube + Helm
./deploy/scripts/default_run.sh --mode k8s --port-forward

# Multi-region fetch-worker pools
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west --mode k8s --port-forward
```

Основные флаги launcher-скриптов:

| Флаг | Режимы | Значение |
|---|---|---|
| `--mode local\|docker\|k8s` | все | Режим запуска |
| `--config <path>` | local | dotenv-файл API, по умолчанию `.env` |
| `--worker-config <path>` | local | dotenv-файл воркеров, по умолчанию `.worker.env` |
| `--build` | local | Собрать бинарники перед запуском |
| `--no-build` | docker/k8s | Пропустить сборку образов |
| `--app-only` | docker | Поднять только app-контейнеры поверх уже запущенной infra |
| `--tag <tag>` | docker/k8s | Тег Docker-образов |
| `--registry <name>` | docker/k8s | Префикс/registry образов |
| `--port-forward` | k8s | Открыть локальные port-forward после деплоя |
| `--full-observability` | k8s | Включить Prometheus, Grafana, OpenSearch и связанные сервисы |
| `--messaging-broker <kind>` | k8s | `rabbitmq`, `kafka` или `grpc_memory` |
| `--regions <csv>` | multi-region | Список регионов для отдельных fetch-worker deployments |
| `-- ...` | все | Проброс дополнительных аргументов в нижележащий скрипт |

## Низкоуровневый запуск

Инфраструктура для локальной разработки:

```bash
docker compose -f docker-compose.yaml up -d
```

Локальные процессы приложения:

```bash
./deploy/scripts/local/start-all.sh
./deploy/scripts/local/stop-all.sh
```

Make targets:

```bash
make run-grpc-server
make run-fetcher
make run-parser
make run-export
make docker-deploy
make k8s-deploy
```

Прямые команды бинарников:

```bash
go run ./cmd/grpc_server/main.go --config-path=.env
go run ./cmd/fetch_worker/main.go --worker-config-path=.worker.env
go run ./cmd/parser_worker/main.go --worker-config-path=.worker.env
go run ./cmd/export_worker/main.go --worker-config-path=.worker.env
go run ./cmd/memory_broker/main.go --addr :9095 --capacity 2000
```

Миграции:

```bash
make local-migration-status
make local-migration-up
make local-migration-down

go run ./cmd/migrate/main.go \
  --dsn "postgres://crawler:pwd@localhost:54322/crawler?sslmode=disable" \
  status
```

## Конфигурация

По умолчанию локальные процессы читают dotenv-файлы `.env` и `.worker.env`. В Docker/Kubernetes настройки прокидываются через env; при необходимости используйте `CONFIG_SOURCE=env`.

Ключевые переменные:

| Группа | Переменные |
|---|---|
| База и логирование | `PG_DSN`, `LOG_LEVEL`, `LOG_ENV` |
| API | `GRPC_HOST`, `GRPC_PORT`, `HTTP_HOST`, `HTTP_PORT`, `HTTP_CORS_ALLOWED_ORIGINS` |
| Auth | `JWT_SECRET`, `ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL`, `JWT_ISSUER`, `JWT_AUDIENCE`, `DEFAULT_USER_EMAIL`, `DEFAULT_USER_PWD` |
| Messaging | `MESSAGING_BROKER` |
| RabbitMQ | `RABBITMQ_URL`, `RABBITMQ_CRAWL_QUEUE_NAME`, `RABBITMQ_CRAWL_QUEUE_NAMES`, `RABBITMQ_PARSING_QUEUE_NAME` |
| Kafka | `KAFKA_BROKERS`, `KAFKA_CONSUMER_GROUP`, `KAFKA_CRAWL_TOPIC_NAME`, `KAFKA_PARSING_TOPIC_NAME` |
| gRPC memory broker | `MEMORY_BROKER_ADDR`, `MEMORY_BROKER_CAPACITY` |
| Workers | `MINIO_ENDPOINT`, `MINIO_USER`, `MINIO_PWD`, `MINIO_BUCKET_NAME`, `REDIS_ADDRESS`, `REDIS_PWD`, `REDIS_DB`, `LIMITER_TYPE`, `FETCHER_TYPE`, `WORKER_REGION` |
| Browser/Selenium fetch | `CHROME_REMOTE_URL` и связанные настройки fetcher backend |
| Queue secrets | `QUEUE_SECRETS_FILE_PATH`, `QUEUE_SECRETS_WATCH_ENABLED`, `QUEUE_SECRETS_RELOAD_INTERVAL` |
| Observability | `OTEL_ENABLED`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_INSECURE`, `OTEL_TRACE_SAMPLE_RATE`, `OTEL_METRICS_INTERVAL_SECONDS`, `OPENSEARCH_ENABLED`, `OPENSEARCH_ENDPOINT`, `OPENSEARCH_INDEX` |

Полная таблица переменных окружения и требования к окружению поддерживаются в [docs/operator-manual.md](docs/operator-manual.md#4-переменные-окружения).

## API и UI

Основные сервисы protobuf API:

- `CrawlerService`: jobs, tasks, analytics, export links.
- `PreviewService`: загрузка preview перед созданием job.
- `AuthService`: register, login, refresh, logout.
- `UserService`: список пользователей и смена ролей.
- `WorkerService`: worker heartbeat stream, worker list, drain, force kill.

Дополнительный HTTP endpoint:

- `GET /api/v1/crawl-queues` — список доступных crawl queues для UI и routing.

Типичные локальные адреса:

| Сервис | Адрес |
|---|---|
| gRPC API | `localhost:8083` |
| HTTP gateway | `http://localhost:8084` |
| Swagger UI | `http://localhost:8084/swagger-ui` |
| Admin UI | `http://localhost:18080` |
| PostgreSQL | `localhost:54322` |
| RabbitMQ UI | `http://localhost:15672` |
| MinIO console | `http://localhost:9001` |
| RedisInsight | `http://localhost:5540` |
| Grafana | `http://localhost:3000` |
| Prometheus | `http://localhost:9090` |
| Jaeger | `http://localhost:16686` |
| OpenSearch Dashboards | `http://localhost:5601` |

## Multi-Region Очереди

`multi_region_run.sh` создаёт отдельный пул `fetch-worker` на каждый регион и выставляет `WORKER_REGION=<region>`. Fetch-worker читает default crawl queue и региональную очередь вида:

```text
<RABBITMQ_CRAWL_QUEUE_NAME>_<region>
```

Скрипт также формирует `RABBITMQ_CRAWL_QUEUE_NAMES`, чтобы API и UI знали весь список очередей. Распределение задач между очередями задаётся через `queue_weights` в конфигурации crawl job; если веса не заданы, маршрутизация идёт равномерно по доступным очередям.

Для Kubernetes node affinity по регионам используйте [multi-region affinity spec](docs/multiregion-affinity-spec.md).

## Сборка, Тесты, Генерация

```bash
make build
make test
make test-coverage

make .bin-deps
make generate
go generate ./...
```

`make generate` обновляет protobuf/gRPC/gateway/validation код и embedded Swagger assets через `buf` и `statik`.

## Документация

- [docs/operator-manual.md](docs/operator-manual.md): эксплуатация, требования, env, Docker Compose, Helm, endpoints, миграции и teardown.
- [docs/parsing-syntax-spec.md](docs/parsing-syntax-spec.md): DSL для extraction spec.
- [docs/operator-messages.md](docs/operator-messages.md): сообщения и команды оператора.
- [docs/multiregion-affinity-spec.md](docs/multiregion-affinity-spec.md): спека по regional node affinity в Kubernetes.
- [docs/example*](docs/): примеры запросов, HTML fixtures и ожидаемые ответы.
- [api/v1/swagger/api.swagger.json](api/v1/swagger/api.swagger.json): сгенерированная OpenAPI/Swagger схема.
- [queue-secrets.json.example](queue-secrets.json.example): пример queue secrets файла.
