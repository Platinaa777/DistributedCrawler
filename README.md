# Distributed Crawler

Распределённая платформа для веб-краулинга на Go. Система предоставляет gRPC API и HTTP gateway, выполняет пайплайн сканирования через набор воркеров, хранит метаданные в PostgreSQL, страницы и экспортные артефакты в MinIO, а для доставки задач использует RabbitMQ, Kafka или gRPC in-memory broker.

Подробное руководство по эксплуатации находится в [docs/operator-manual.md](docs/operator-manual.md). Примеры парсинга и образцы запросов лежат в каталоге [`docs/example*`](docs/).

## Что Делает Система

- `grpc-server` принимает запросы на создание crawl-job, сохраняет jobs/tasks/configs, обслуживает gRPC и HTTP API, запускает outbox publisher и schedule worker внутри одного процесса.
- `fetch-worker` потребляет crawl-задачи, проверяет scope и robots-правила, загружает страницы, сохраняет HTML в MinIO и публикует задачи на парсинг.
- `parser-worker` загружает сохранённый HTML, извлекает записи по DSL проекта, находит follow-up ссылки, создаёт новые задачи и сохраняет результаты извлечения.
- `export-worker` опрашивает завершённые job, агрегирует результаты задач и формирует JSON/CSV-экспорт в MinIO.
- `scheduler-worker` также доступен как отдельный процесс, если нужно вынести расписание за пределы API-процесса.
- `memory_broker` даёт лёгкий встроенный брокер для разработки и тестов, когда RabbitMQ/Kafka не нужны.

## Архитектура

```mermaid
flowchart LR
    User[Пользователь или UI] --> API[Координатор / grpc-server]
    API --> PG[(PostgreSQL)]
    API --> OUTBOX[Outbox publisher]
    OUTBOX --> MQ[Брокер сообщений]
    MQ --> FETCH[Fetch worker]
    FETCH --> MINIO[(Хранилище страниц MinIO)]
    FETCH --> MQ
    MQ --> PARSER[Parser worker]
    PARSER --> PG
    PARSER --> MINIO
    PARSER --> MQ
    PG --> EXPORT[Export worker]
    MINIO --> EXPORT
    EXPORT --> MINIO
```

### Роли Компонентов

```text
Координатор / coordinator service (grpc-server)
- Точка входа для UI и API-клиентов.
- Валидирует запросы и сохраняет crawl jobs и метаданные задач в PostgreSQL.
- Публикует gRPC на :8083 и HTTP gateway / Swagger на :8084.
- Создаёт администратора по умолчанию при старте.
- Запускает outbox publisher, который превращает события из БД в сообщения брокера.
- Запускает цикл schedule worker для cron-задач.

Fetch worker
- Читает crawl-задачи из одной или нескольких crawl-очередей.
- Применяет scope-правила, rate limiting и опциональную проверку robots.txt.
- Загружает HTML через настроенную реализацию fetcher.
- Загружает содержимое страницы и снапшоты в MinIO.
- Отправляет задачи в parsing queue.

Parser worker
- Читает parsing-задачи из parsing queue.
- Загружает ранее сохранённый HTML из MinIO.
- Извлекает поля и элементы по extraction DSL.
- Сохраняет результаты извлечения и статус задачи.
- Находит пагинацию и дочерние ссылки, затем ставит follow-up crawl-задачи через outbox flow.

Exporter worker
- Опрашивает job, у которых crawl-часть завершена.
- Загружает результаты извлечения по задачам из MinIO.
- Формирует итоговые JSON и CSV файлы экспорта.
- Обновляет export status и ключи объектов в записи job.
```

### Пайплайн

```mermaid
sequenceDiagram
    participant C as Клиент или UI
    participant A as grpc-server
    participant D as PostgreSQL
    participant O as Outbox publisher
    participant B as Broker
    participant F as fetch-worker
    participant M as MinIO
    participant P as parser-worker
    participant E as export-worker

    C->>A: Create crawl job
    A->>D: Save job, seed tasks, save outbox events
    O->>D: Poll unprocessed outbox events
    O->>B: Publish crawl task
    B->>F: Deliver crawl task
    F->>M: Save fetched page
    F->>B: Publish parsing task
    B->>P: Deliver parsing task
    P->>M: Read HTML and save extraction result
    P->>D: Update task and create follow-up tasks
    O->>B: Publish discovered follow-up crawl tasks
    E->>D: Poll export-eligible jobs
    E->>M: Read task results and write JSON/CSV exports
```

## Способы Запуска Проекта

### 1. Основной launcher, один регион

`deploy/scripts/default_run.sh` — основной launcher для полного стека.

```bash
# локальные фоновые процессы
./deploy/scripts/default_run.sh

# docker compose
./deploy/scripts/default_run.sh --mode docker

# minikube + helm
./deploy/scripts/default_run.sh --mode k8s
```

#### Аргументы

| Флаг | Режимы | По умолчанию | Значение |
|---|---|---:|---|
| `--mode <local\|docker\|k8s>` | все | `local` | Выбор режима запуска |
| `--config <path>` | local | `.env` | Файл конфигурации API |
| `--worker-config <path>` | local | `.worker.env` | Файл конфигурации воркеров |
| `--build` | local | выкл | Сначала собрать бинарники и запускать их вместо `go run` |
| `--redis-limiter` | local | выкл | Экспортировать `LIMITER_TYPE=redis` перед стартом |
| `--no-build` | docker, k8s | выкл | Пропустить сборку образов |
| `--app-only` | docker | выкл | Поднять только app-контейнеры, переиспользовать уже запущенную infra |
| `--tag <tag>` | docker, k8s | `latest` | Тег образа |
| `--registry <name>` | docker, k8s | `distributed-crawler` | Префикс образа |
| `--no-bucket` | k8s | выкл | Не создавать MinIO bucket |
| `--port-forward` | k8s | выкл | Запустить `kubectl port-forward` после деплоя |
| `--full-observability` | k8s | выкл | Включить стек Prometheus/Grafana/OpenSearch |
| `--jwt-secret <value>` | k8s | dev-значение | JWT secret для API |
| `--pg-password <pwd>` | k8s | dev-значение | Пароль PostgreSQL |
| `--default-user-password <pwd>` | k8s | dev-значение | Пароль seed-админа |
| `--messaging-broker <kind>` | k8s | `rabbitmq` | `rabbitmq`, `kafka` или `grpc_memory` |
| `-- ...` | все | нет | Передать оставшиеся аргументы нижележащим скриптам |

Примеры:

```bash
./deploy/scripts/default_run.sh --mode docker --no-build
./deploy/scripts/default_run.sh --mode k8s --port-forward --full-observability
./deploy/scripts/default_run.sh --mode k8s -- --app-set grpcServer.replicaCount=2
```

### 2. Основной launcher, несколько регионов fetch-worker

`deploy/scripts/multi_region_run.sh` поднимает отдельный пул `fetch-worker` на каждый регион. `parser-worker` при этом остаются общими.

```bash
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west --mode local
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west --mode k8s --port-forward
```

#### Аргументы

| Флаг | Режимы | По умолчанию | Значение |
|---|---|---:|---|
| `--regions <csv>` | все | обязателен | Список регионов, например `us-east,eu-west` |
| `--mode <local\|docker\|k8s>` | все | `docker` | Выбор режима запуска |
| `--config <path>` | local | `.env` | Файл конфигурации API |
| `--worker-config <path>` | local | `.worker.env` | Конфигурация non-fetch воркеров |
| `--build` | local | выкл | Собрать бинарники перед запуском |
| `--no-build` | docker, k8s | выкл | Пропустить сборку образов |
| `--tag <tag>` | docker, k8s | `latest` | Тег образа |
| `--registry <name>` | docker | `distributed-crawler` | Префикс образа |
| `--no-bucket` | k8s | выкл | Не создавать MinIO bucket |
| `--port-forward` | k8s | выкл | Запустить `kubectl port-forward` после деплоя |
| `--full-observability` | k8s | выкл | Включить полный observability stack |
| `--jwt-secret <value>` | k8s | dev-значение | JWT secret для API |
| `--pg-password <pwd>` | k8s | dev-значение | Пароль PostgreSQL |
| `--default-user-password <pwd>` | k8s | dev-значение | Пароль seed-админа |
| `--messaging-broker <kind>` | k8s | `rabbitmq` | `rabbitmq`, `kafka` или `grpc_memory` |
| `-- ...` | все | нет | Передать оставшиеся аргументы нижележащим скриптам |

Примечания:

- Скрипт экспортирует `RABBITMQ_CRAWL_QUEUE_NAMES`, чтобы API и UI видели полный список региональных очередей.
- Каждый региональный fetch-worker получает `WORKER_REGION=<region>`.
- Маршрутизация очередей конфигурируется через настройки; runtime discovery региональных broker endpoint из базы нет.

## Низкоуровневые Режимы Запуска

### Локальные процессы

Сначала поднять инфраструктуру:

```bash
docker compose -f docker-compose.yaml up -d
```

Потом поднять компоненты приложения:

```bash
./deploy/scripts/local/start-all.sh
./deploy/scripts/local/stop-all.sh
```

Сборка бинарников:

```bash
./deploy/scripts/local/build.sh
./deploy/scripts/local/build.sh grpc_server fetch_worker
```

Прямые `make`-таргеты:

```bash
make run-grpc-server
make run-fetcher
make run-parser
make run-export
```

### Docker Compose

Полный стек:

```bash
make docker-deploy
make docker-deploy ARGS="--pg-password mypwd --jwt-secret supersecret"
make docker-deploy ARGS="--messaging-broker kafka --no-build"
```

Один компонент:

```bash
make docker-deploy-component COMPONENT=grpc-server
make docker-deploy-component COMPONENT=fetch-worker
```

Аргументы Docker launcher из `deploy/scripts/docker/launch.sh`:

| Флаг | По умолчанию |
|---|---:|
| `--registry <name>` | `distributed-crawler` |
| `--tag <tag>` | `latest` |
| `--components <csv>` / `--component <name>` | `grpc-server,fetch-worker,parser-worker,export-worker,ui` |
| `--pg-user <name>` | `crawler` |
| `--pg-password <pwd>` | `some-pwd-123` |
| `--pg-database <name>` | `crawler` |
| `--pg-port <port>` | `54322` |
| `--rabbitmq-user <name>` | `guest` |
| `--rabbitmq-password <pwd>` | `guest` |
| `--minio-user <name>` | `minioadmin` |
| `--minio-password <pwd>` | `minioadmin` |
| `--minio-bucket <name>` | `pages` |
| `--redis-password <pwd>` | `some_redis_pwd_123` |
| `--grafana-user <name>` | `admin` |
| `--grafana-password <pwd>` | `changeme-grafana-password` |
| `--jwt-secret <value>` | небезопасное dev-значение |
| `--default-user-email <email>` | `admin@example.com` |
| `--default-user-password <pwd>` | `12345678` |
| `--messaging-broker <kind>` | `rabbitmq` |
| `--cors-origin <origin>` | `http://localhost:4200` |
| `--queue-secrets-file <path>` | `queue-secrets.json.example` |
| `--app-only` | выкл |
| `--no-build` | выкл |
| `--env KEY=VALUE` | можно повторять |
| `--compose-arg <arg>` | можно повторять |

### Kubernetes через Helm

Полный стек:

```bash
make k8s-deploy
make k8s-deploy ARGS="--full-observability --port-forward"
```

Один компонент:

```bash
make k8s-deploy-component COMPONENT=grpc-server
make k8s-deploy-component COMPONENT=fetch-worker
```

Основные аргументы `deploy/scripts/k8s/launch-minikube.sh`:

| Флаг | По умолчанию |
|---|---:|
| `--release-name <name>` | `crawler` |
| `--namespace <name>` | `crawler` |
| `--infra-release-name <name>` | `infra` |
| `--infra-namespace <name>` | `infra` |
| `--values-env <dev\|prod>` | `dev` |
| `--registry <name>` | `distributed-crawler` |
| `--tag <tag>` | `latest` |
| `--queue-secrets-file <path>` | `queue-secrets.json.example` |
| `--pg-user`, `--pg-password`, `--pg-database` | dev-значения |
| `--rabbitmq-user`, `--rabbitmq-password` | dev-значения |
| `--minio-user`, `--minio-password`, `--minio-bucket` | dev-значения |
| `--redis-password` | dev-значение |
| `--grafana-user`, `--grafana-password` | dev-значения |
| `--jwt-secret` | небезопасное dev-значение |
| `--default-user-email`, `--default-user-password` | admin defaults |
| `--messaging-broker <kind>` | `rabbitmq` |
| `--cors-origin <origin>` | `http://localhost:4200` |
| `--skip-minikube-start` | выкл |
| `--no-build` | выкл |
| `--no-bucket` | выкл |
| `--port-forward` | выкл |
| `--port-forward-services <csv>` | все поддерживаемые сервисы |
| `--lite` | вкл |
| `--full-observability` | выкл |
| `--minikube-driver <name>` | `docker` |
| `--minikube-cpus <n>` | `6` |
| `--minikube-memory <mb>` | `8192` |
| `--minikube-disk-size <size>` | не задан |
| `--app-values-file`, `--infra-values-file` | можно повторять |
| `--app-set`, `--app-set-string`, `--app-set-file` | можно повторять |
| `--infra-set`, `--infra-set-string`, `--infra-set-file` | можно повторять |

## Прямые Флаги Бинарников

Ниже реальные command-line flags, которые понимают бинарники.

| Бинарник | Флаги |
|---|---|
| `cmd/grpc_server` | `--config-path=.env` |
| `cmd/fetch_worker` | `--worker-config-path=.worker.env` |
| `cmd/parser_worker` | `--worker-config-path=.worker.env` |
| `cmd/export_worker` | `--worker-config-path=.worker.env` |
| `cmd/scheduler_worker` | `--worker-config-path=.worker.env` |
| `cmd/memory_broker` | `--addr=:9095`, `--capacity=1000` |
| `cmd/migrate` | `--dsn`, `--dir`, затем команда `up`, `up-by-one`, `down`, `reset`, `status`, `version`, `create <name>` |

Примеры:

```bash
go run ./cmd/grpc_server/main.go --config-path=.env
go run ./cmd/fetch_worker/main.go --worker-config-path=.worker.env
go run ./cmd/memory_broker/main.go --addr :9095 --capacity 2000
go run ./cmd/migrate/main.go --dsn "postgres://crawler:pwd@localhost:54322/crawler?sslmode=disable" status
```

## Ключевые Переменные Окружения

Конфигурация читается из dotenv-файлов, если не выставлен `CONFIG_SOURCE=env`. Именно так настройки прокидываются в Docker и Kubernetes.

### Базовые

| Переменная | Назначение |
|---|---|
| `PG_DSN` | DSN подключения к PostgreSQL |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` |
| `LOG_ENV` | `development` или `production` |
| `MESSAGING_BROKER` | `rabbitmq`, `kafka`, `grpc_memory` |

### API

| Переменная | Назначение |
|---|---|
| `GRPC_HOST`, `GRPC_PORT` | адрес привязки gRPC |
| `HTTP_HOST`, `HTTP_PORT` | адрес привязки HTTP gateway |
| `JWT_SECRET` | секрет подписи JWT |
| `ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL` | время жизни токенов |
| `JWT_ISSUER`, `JWT_AUDIENCE` | метаданные JWT |
| `DEFAULT_USER_EMAIL`, `DEFAULT_USER_PWD` | seed-учётка администратора |
| `HTTP_CORS_ALLOWED_ORIGINS` | CORS allowlist |

### Messaging

| Брокер | Переменные |
|---|---|
| RabbitMQ | `RABBITMQ_URL`, `RABBITMQ_CRAWL_QUEUE_NAME`, `RABBITMQ_CRAWL_QUEUE_NAMES`, `RABBITMQ_PARSING_QUEUE_NAME` |
| Kafka | `KAFKA_BROKERS`, `KAFKA_CONSUMER_GROUP`, `KAFKA_CRAWL_TOPIC_NAME`, `KAFKA_PARSING_TOPIC_NAME` |
| gRPC memory | `MEMORY_BROKER_ADDR`, `MEMORY_BROKER_CAPACITY` |

### Workers

| Переменная | Назначение |
|---|---|
| `MINIO_ENDPOINT`, `MINIO_USER`, `MINIO_PWD`, `MINIO_BUCKET_NAME`, `MINIO_PUBLIC_BASE_URL` | объектное хранилище |
| `REDIS_ADDRESS`, `REDIS_PWD`, `REDIS_DB` | rate limiting и кэш |
| `LIMITER_TYPE` | `redis` или `inmemory` |
| `WORKER_REGION` | региональный суффикс для fetch-worker |
| `FETCHER_TYPE` | `http`, `browser` или `selenium` |
| `CHROME_REMOTE_URL` | endpoint удалённого Chrome при browser fetcher |
| `QUEUE_SECRETS_FILE_PATH` | файл с credentials для queue endpoint |
| `QUEUE_SECRETS_WATCH_ENABLED` | включить периодический reload |
| `QUEUE_SECRETS_RELOAD_INTERVAL` | интервал reload |

### Observability

| Переменная | Назначение |
|---|---|
| `OTEL_ENABLED` | включить OpenTelemetry |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | адрес collector |
| `OTEL_EXPORTER_OTLP_INSECURE` | отключить TLS для OTLP |
| `OTEL_TRACE_SAMPLE_RATE` | доля trace sampling |
| `OTEL_METRICS_INTERVAL_SECONDS` | интервал экспорта метрик |
| `OPENSEARCH_ENABLED` | включить экспорт логов |
| `OPENSEARCH_ENDPOINT`, `OPENSEARCH_INDEX` | target OpenSearch |

## Локальные Endpoint

Типичные адреса после локального или Docker-запуска:

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

## Миграции, Сборка, Тесты, Генерация

```bash
make build

make local-migration-status
make local-migration-up
make local-migration-down
make local-migration-create NAME=add_something

make test
make test-coverage

make .bin-deps
make generate
go generate ./...
```

## Полезные Ссылки По Проекту

- [docs/operator-manual.md](docs/operator-manual.md): руководство по развертыванию и эксплуатации
- [docs/parsing-syntax-spec.md](docs/parsing-syntax-spec.md): extraction DSL
- [api/v1/swagger/api.swagger.json](api/v1/swagger/api.swagger.json): сгенерированная схема API
- [queue-secrets.json.example](queue-secrets.json.example): пример файла с queue secrets
