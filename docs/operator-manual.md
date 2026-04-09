# Руководство оператора — Distributed Crawler

## Оглавление

1. [Общие сведения](#1-общие-сведения)
2. [Требования к окружению](#2-требования-к-окружению)
3. [Описание компонентов системы](#3-описание-компонентов-системы)
4. [Переменные окружения](#4-переменные-окружения)
5. [Быстрый запуск с помощью launcher-скриптов](#5-быстрый-запуск-с-помощью-launcher-скриптов)
6. [Развертывание с помощью Docker Compose](#6-развертывание-с-помощью-docker-compose)
7. [Развертывание в Kubernetes с помощью Helm](#7-развертывание-в-kubernetes-с-помощью-helm)
8. [Адреса сервисов и интерфейсов](#8-адреса-сервисов-и-интерфейсов)
9. [Управление миграциями базы данных](#9-управление-миграциями-базы-данных)
10. [Остановка и удаление](#10-остановка-и-удаление)

---

## 1. Общие сведения

Distributed Crawler — распределённая платформа для сканирования веб-страниц,
построенная на языке Go. Система состоит из нескольких независимых сервисов,
взаимодействующих через брокер сообщений (RabbitMQ, Kafka или gRPC in-memory
broker). В качестве хранилища используются PostgreSQL (метаданные задач) и
MinIO (содержимое страниц). Redis применяется для ограничения скорости запросов
и кэширования.

Необходимые инструменты для развертывания расположены в директориях:

| Путь | Назначение |
|------|-----------|
| `docker-compose.yaml` | Инфраструктура для локальной разработки |
| `deploy/scripts/default_run.sh` | Launcher: запуск полного стека (один регион) в режимах local/docker/k8s |
| `deploy/scripts/multi_region_run.sh` | Launcher: запуск стека с несколькими региональными пулами fetch-воркеров |
| `deploy/scripts/local/` | Низкоуровневые скрипты запуска компонентов приложения локально |
| `deploy/scripts/docker/` | Низкоуровневые скрипты деплоя через Docker Compose |
| `deploy/scripts/k8s/` | Низкоуровневые скрипты сборки образов и деплоя в Kubernetes |
| `deploy/helm/distributed-crawler/` | Helm-чарт приложения |
| `deploy/helm/infra/` | Helm-чарт инфраструктурных сервисов |

---

## 2. Требования к окружению

### Для развертывания через Docker Compose

- Docker Engine 24+ и Docker Compose v2+
- Go 1.22+ (если приложение запускается через `go run` без сборки)
- Порты, свободные на хосте: `5432 / 54322`, `5672`, `15672`, `9000`, `9001`,
  `6379`, `4317`, `4318`, `9090`, `3000`, `9200`, `5601`, `8083`, `8084`

### Для развертывания через Kubernetes + Helm

- Kubernetes 1.27+ (например, minikube, k3s или управляемый кластер)
- Helm 3.12+
- kubectl, настроенный на целевой кластер
- Docker (для сборки образов)
- Доступ к реестру образов (для production-развертывания)

---

## 3. Описание компонентов системы

| Компонент | Бинарный файл | Назначение |
|-----------|--------------|-----------|
| `grpc-server` | `cmd/grpc_server` | Основной API-сервер: gRPC `:8083` + HTTP-шлюз `:8084` |
| `fetch-worker` | `cmd/fetch_worker` | Загружает страницы, сохраняет в MinIO, публикует в очередь разбора |
| `parser-worker` | `cmd/parser_worker` | Разбирает страницы, извлекает записи, обнаруживает ссылки |
| `export-worker` | `cmd/export_worker` | Генерирует файлы экспорта в MinIO по завершённым задачам |
| `scheduler-worker` | `cmd/scheduler_worker` | Создаёт новые циклы сканирования по расписанию |
| `ui` | Docker-образ nginx | Веб-интерфейс администратора (Angular) |

Инфраструктурные сервисы (управляются отдельно):

| Сервис | Назначение |
|--------|-----------|
| PostgreSQL 14 | Хранение метаданных заданий, задач, пользователей |
| RabbitMQ 4 | Брокер сообщений между компонентами |
| MinIO | Объектное хранилище для сохранения содержимого страниц |
| Redis 7 | Ограничение скорости запросов, кэш robots.txt |
| OpenTelemetry Collector | Сбор трассировок и метрик |
| Jaeger | UI для распределённой трассировки |
| Prometheus + Grafana | Метрики и дашборды |
| OpenSearch | Хранение и поиск по логам приложения |

---

## 4. Переменные окружения

### 4.1 Обязательные переменные для всех компонентов

| Переменная | Пример | Описание |
|-----------|--------|---------|
| `PG_DSN` | `postgres://user:pwd@host:5432/crawler?sslmode=disable` | Строка подключения к PostgreSQL |
| `LOG_LEVEL` | `info` | Уровень логирования (`debug`, `info`, `warn`, `error`) |
| `LOG_ENV` | `production` | Окружение для логгера (`development`, `production`) |

### 4.2 Переменные API-сервера (`grpc-server`)

| Переменная | Пример | Описание |
|-----------|--------|---------|
| `GRPC_HOST` | `0.0.0.0` | Адрес прослушивания gRPC |
| `GRPC_PORT` | `8083` | Порт gRPC |
| `HTTP_HOST` | `0.0.0.0` | Адрес прослушивания HTTP-шлюза |
| `HTTP_PORT` | `8084` | Порт HTTP-шлюза |
| `JWT_SECRET` | `min-32-chars-random-string` | Секрет для подписи JWT-токенов |
| `ACCESS_TOKEN_TTL` | `15m` | Время жизни access-токена |
| `REFRESH_TOKEN_TTL` | `720h` | Время жизни refresh-токена |
| `JWT_ISSUER` | `distributed-crawler` | Издатель JWT |
| `JWT_AUDIENCE` | `api` | Аудитория JWT |
| `DEFAULT_USER_EMAIL` | `admin@example.com` | Email администратора по умолчанию |
| `DEFAULT_USER_PWD` | `changeme` | Пароль администратора по умолчанию |

### 4.3 Переменные брокера сообщений

Выбор брокера задаётся переменной `MESSAGING_BROKER` (`rabbitmq` | `kafka` | `grpc_memory`).

**RabbitMQ (по умолчанию):**

| Переменная | Пример | Описание |
|-----------|--------|---------|
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | URL подключения |
| `RABBITMQ_CRAWL_QUEUE_NAME` | `crawl_queue` | Очередь задач сканирования |
| `RABBITMQ_PARSING_QUEUE_NAME` | `parsing_queue` | Очередь задач разбора |

**Kafka:**

| Переменная | Пример | Описание |
|-----------|--------|---------|
| `KAFKA_BROKERS` | `localhost:9091` | Адреса брокеров Kafka |
| `KAFKA_CONSUMER_GROUP` | `distributed-crawler` | Группа потребителей |
| `KAFKA_CRAWL_TOPIC_NAME` | `crawl_queue` | Топик задач сканирования |
| `KAFKA_PARSING_TOPIC_NAME` | `parsing_queue` | Топик задач разбора |

**gRPC in-memory broker:**

| Переменная | Пример | Описание |
|-----------|--------|---------|
| `MEMORY_BROKER_ADDR` | `:9095` | Адрес gRPC-брокера |
| `MEMORY_BROKER_CAPACITY` | `1000` | Ёмкость очередей |

### 4.4 Переменные воркеров (`fetch-worker`, `parser-worker` и др.)

| Переменная | Пример | Описание |
|-----------|--------|---------|
| `MINIO_ENDPOINT` | `localhost:9000` | Адрес MinIO |
| `MINIO_USER` | `minioadmin` | Пользователь MinIO |
| `MINIO_PWD` | `changeme` | Пароль MinIO |
| `MINIO_USE_SSL` | `false` | Использование SSL для MinIO |
| `MINIO_BUCKET_NAME` | `pages` | Имя бакета для страниц |
| `REDIS_ADDRESS` | `localhost:6379` | Адрес Redis |
| `REDIS_PWD` | `changeme` | Пароль Redis |
| `REDIS_DB` | `0` | Номер базы данных Redis |
| `LIMITER_TYPE` | `redis` | Тип ограничителя скорости (`redis` или `inmemory`) |
| `WORKER_REGION` | `default` | Регион воркера для маршрутизации очередей |

### 4.5 Переменные наблюдаемости (опционально)

| Переменная | Пример | Описание |
|-----------|--------|---------|
| `OTEL_ENABLED` | `true` | Включить OpenTelemetry |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | Адрес OTel Collector |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true` | Отключить TLS для OTel |
| `OTEL_TRACE_SAMPLE_RATE` | `1` | Частота сэмплирования трассировок (0–1) |
| `OTEL_METRICS_INTERVAL_SECONDS` | `15` | Интервал отправки метрик |
| `OPENSEARCH_ENABLED` | `true` | Включить отправку логов в OpenSearch |
| `OPENSEARCH_ENDPOINT` | `http://localhost:9200` | Адрес OpenSearch |
| `OPENSEARCH_INDEX` | `app-logs` | Имя индекса логов |

### 4.6 Переменные Grafana и секретов очередей (опционально)

| Переменная | Пример | Описание |
|-----------|--------|---------|
| `GRAFANA_USER` | `admin` | Логин Grafana |
| `GRAFANA_PWD` | `changeme` | Пароль Grafana |
| `QUEUE_SECRETS_FILE_PATH` | `./queue-secrets.json` | Путь к файлу секретов очередей |
| `QUEUE_SECRETS_WATCH_ENABLED` | `true` | Автоперечитывание файла секретов |
| `QUEUE_SECRETS_RELOAD_INTERVAL` | `60s` | Интервал перечитывания |

---

## 5. Быстрый запуск с помощью launcher-скриптов

В директории `deploy/scripts/` расположены два верхнеуровневых launcher-скрипта,
которые покрывают два основных сценария развертывания — однорегиональный и
многорегиональный. Скрипты принимают флаг `--mode` и делегируют работу
соответствующим низкоуровневым скриптам из `local/`, `docker/` или `k8s/`.
Дополнительные аргументы, переданные после `--`, пробрасываются в нижележащий
скрипт без изменений.

| Скрипт | Назначение |
|--------|-----------|
| `deploy/scripts/default_run.sh` | Полный стек, один регион (без `WORKER_REGION`) |
| `deploy/scripts/multi_region_run.sh` | Полный стек с отдельным пулом fetch-воркеров на каждый регион |

---

### 5.1 `default_run.sh` — стандартный запуск (один регион)

Запускает полный стек с единым пулом fetch-воркеров без привязки к региону.

#### Быстрый старт

```bash
# Локальные процессы (go run, файлы .env / .worker.env по умолчанию)
./deploy/scripts/default_run.sh

# Docker Compose
./deploy/scripts/default_run.sh --mode docker

# Kubernetes (minikube + Helm)
./deploy/scripts/default_run.sh --mode k8s
```

#### Доступные флаги

| Флаг | Режим | По умолчанию | Описание |
|------|-------|-------------|---------|
| `--mode` | все | `local` | `local`, `docker` или `k8s` |
| `--config <путь>` | local | `.env` | Файл конфигурации API-сервера |
| `--worker-config <путь>` | local | `.worker.env` | Файл конфигурации воркеров |
| `--build` | local | выкл. | Собрать Go-бинарники перед запуском |
| `--no-build` | docker/k8s | выкл. | Пропустить сборку образов |
| `--app-only` | docker | выкл. | Не запускать инфраструктурные сервисы |
| `--tag <тег>` | docker/k8s | `latest` | Тег Docker-образа |
| `--registry <имя>` | docker/k8s | `distributed-crawler` | Префикс имени образа |
| `--port-forward` | k8s | выкл. | Пробросить порты после деплоя |
| `--full-observability` | k8s | выкл. | Включить Prometheus/Grafana/OpenSearch |
| `--jwt-secret <значение>` | k8s | dev-значение | Секрет для подписи JWT |
| `--pg-password <пароль>` | k8s | dev-значение | Пароль PostgreSQL |
| `--default-user-password <пароль>` | k8s | dev-значение | Пароль администратора UI |
| `--messaging-broker <тип>` | k8s | `rabbitmq` | `rabbitmq`, `kafka`, `grpc_memory` |

Пример передачи дополнительных параметров в нижележащий скрипт:

```bash
./deploy/scripts/default_run.sh --mode k8s -- \
  --jwt-secret supersecret \
  --app-set grpcServer.replicaCount=2
```

---

### 5.2 `multi_region_run.sh` — многорегиональный запуск

Запускает стек с отдельным пулом fetch-воркеров для каждого региона. Каждый
пул получает свою метку `WORKER_REGION`. Parser-воркеры всегда запускаются в
едином общем пуле и не являются регион-зависимыми.

Параметры подключения к очередям для каждого регионального пула задаются через
переменные окружения или файл конфигурации воркера при запуске; динамического
обнаружения очередей через базу данных нет. Перед запуском необходимо указать
корректные `RABBITMQ_URL` / `RABBITMQ_CRAWL_QUEUE_NAME` (или аналоги для
Kafka) для каждого регионального пула.

#### Быстрый старт

```bash
# Docker Compose — два региональных пула fetch-воркеров
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west

# Kubernetes
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west --mode k8s

# Локальные процессы
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west --mode local
```

#### Доступные флаги

| Флаг | Режим | По умолчанию | Описание |
|------|-------|-------------|---------|
| `--regions <csv>` | все | **обязательный** | Названия регионов через запятую |
| `--mode` | все | `docker` | `local`, `docker` или `k8s` |
| `--config <путь>` | local | `.env` | Файл конфигурации API-сервера |
| `--worker-config <путь>` | local | `.worker.env` | Конфигурация не-fetch воркеров |
| `--build` | local | выкл. | Собрать Go-бинарники перед запуском |
| `--no-build` | docker/k8s | выкл. | Пропустить сборку образов |
| `--tag <тег>` | docker/k8s | `latest` | Тег Docker-образа |
| `--registry <имя>` | docker | `distributed-crawler` | Префикс имени образа |
| `--port-forward` | k8s | выкл. | Пробросить порты после деплоя |
| `--full-observability` | k8s | выкл. | Включить Prometheus/Grafana/OpenSearch |
| `--jwt-secret <значение>` | k8s | dev-значение | Секрет для подписи JWT |
| `--pg-password <пароль>` | k8s | dev-значение | Пароль PostgreSQL |
| `--messaging-broker <тип>` | k8s | `rabbitmq` | Тип брокера сообщений |

Пример:

```bash
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west --mode k8s -- \
  --jwt-secret supersecret \
  --port-forward
```

#### Что делает каждый режим

**`--mode local`**

1. Запускает `grpc_server`, `parser_worker`, `export_worker`, `scheduler_worker`
   как фоновые процессы (PID-файлы в `.pids/`).
2. Запускает по одному процессу `fetch_worker` на каждый регион с переменной
   `WORKER_REGION=<регион>`.

Логи записываются в `<корень_проекта>/logs/`, файлы fetch-воркеров именуются
`fetch-worker-<регион>.log`. Для остановки всех процессов используйте
`./deploy/scripts/local/stop-all.sh`.

**`--mode docker`**

1. Запускает `deploy/scripts/docker/deploy-all.sh` со всеми компонентами
   **кроме** `fetch-worker`.
2. Запускает `docker compose run --detach --no-deps -e WORKER_REGION=<регион> fetch-worker`
   для каждого региона, создавая по одному отдельному контейнеру.

Просмотр запущенных fetch-воркеров: `docker ps`. Остановка всего стека:
`docker compose down` из корня проекта.

**`--mode k8s`**

Делегирует вызов в `deploy/scripts/k8s/launch-minikube.sh` с параметром:

```
--app-set fetchWorker.regions={us-east,eu-west}
```

Helm создаёт по одному Deployment `fetch-worker` на каждый регион с инжектом
`WORKER_REGION=<регион>` и суффиксом `-<регион>` в имени Deployment.

---

### 5.3 Низкоуровневые скрипты

Launcher-скрипты не заменяют низкоуровневые скрипты в `local/`, `docker/` и
`k8s/` — те по-прежнему доступны для точечных операций:

| Скрипт | Назначение |
|--------|-----------|
| `local/build.sh` | Сборка Go-бинарников |
| `local/run.sh <компонент>` | Запуск отдельного компонента |
| `local/start-all.sh` / `stop-all.sh` | Запуск / остановка всех локальных процессов |
| `docker/deploy-all.sh` | Полный деплой через Docker Compose |
| `docker/deploy-component.sh` | Передеплой отдельного compose-сервиса |
| `docker/teardown.sh` | Удаление всех Docker-контейнеров и томов на хосте |
| `k8s/launch-minikube.sh` | Полный деплой в minikube+Helm со всеми опциями |
| `k8s/port-forward.sh` | Проброс портов сервисов k8s на локальную машину |
| `k8s/teardown.sh` | Удаление Helm-релизов и namespace |

---

## 6. Развертывание с помощью Docker Compose

### 6.1 Запуск инфраструктурных сервисов

Все инфраструктурные сервисы (PostgreSQL, RabbitMQ, MinIO, Redis, Jaeger,
Prometheus, Grafana, OpenSearch, Kafka, OTel Collector) описаны в файле
`docker-compose.yaml` в корне репозитория.

1) Необходимо клонировать репозиторий с программой, выполнив в терминале
команду:
```bash
git clone <url-репозитория>
```

2) Перейти терминалом в директорию с клонированным репозиторием:
```bash
cd distributed-crawler
```

3) Создать файл `.env` в корне репозитория с переменными окружения (за основу
взять содержимое файла `.env` из репозитория) и задать следующие обязательные
параметры:

```bash
# Аутентификация
PG_USER=<имя_пользователя_бд>
PG_PASSWORD=<пароль_бд>
PG_DATABASE_NAME=crawler
PG_PORT=54322

RMQ_USER=guest
RMQ_PWD=<пароль_rabbitmq>

MINIO_USER=minioadmin
MINIO_PWD=<пароль_minio>

REDIS_PWD=<пароль_redis>

GRAFANA_USER=admin
GRAFANA_PWD=<пароль_grafana>
```

4) Запустить инфраструктурные сервисы:
```bash
docker compose up -d
```

5) Убедиться, что все сервисы запущены:
```bash
docker compose ps
```

Порты, через которые можно подключаться к инфраструктурным сервисам:

| Сервис | Порт |
|--------|------|
| PostgreSQL | `54322` |
| RabbitMQ (AMQP) | `5672` |
| RabbitMQ UI | `15672` |
| MinIO API | `9000` |
| MinIO Console | `9001` |
| Redis | `6379` |
| RedisInsight | `5540` |
| Jaeger UI | `16686` |
| Prometheus | `9090` |
| Grafana | `3000` |
| OpenSearch | `9200` |
| OpenSearch Dashboards | `5601` |
| OTel Collector (gRPC) | `4317` |
| Kafka | `9091` |
| Kafka UI | `8080` |

### 6.2 Запуск компонентов приложения

Компоненты приложения запускаются локально (через `go run` или собранные
бинарные файлы) с помощью скриптов из директории `deploy/scripts/local/`.

**Шаг 1. Проверить наличие файлов конфигурации.**

В корне репозитория должны существовать два файла:
- `.env` — конфигурация для API-сервера;
- `.worker.env` — конфигурация для воркеров.

Убедиться, что в `.env` заданы `JWT_SECRET`, `DEFAULT_USER_EMAIL`,
`DEFAULT_USER_PWD`, а строка `PG_DSN` указывает на запущенный PostgreSQL.

**Шаг 2. Применить миграции базы данных.**

```bash
make local-migration-up
```

**Шаг 3. Запустить все компоненты приложения.**

```bash
./deploy/scripts/local/start-all.sh
```

Скрипт запустит в фоне следующие компоненты: `grpc_server`, `fetch_worker`,
`parser_worker`, `export_worker`, `scheduler_worker`. Логи записываются в
директорию `logs/`, идентификаторы процессов — в `.pids/`.

Допускается запустить компоненты из предварительно собранных бинарных файлов
вместо исходного кода:

```bash
# Собрать бинарные файлы
./deploy/scripts/local/build.sh

# Запустить из бинарных файлов
USE_BINARY=1 ./deploy/scripts/local/start-all.sh
```

Можно также использовать произвольный путь к конфигурационным файлам:

```bash
CONFIG_PATH=/etc/crawler/server.env \
WORKER_CONFIG=/etc/crawler/worker.env \
./deploy/scripts/local/start-all.sh
```

**Шаг 4. Проверить работоспособность.**

HTTP-шлюз API-сервера доступен по адресу `http://localhost:8084`.
gRPC API — по адресу `localhost:8083`.
Веб-интерфейс (если запущен отдельно) — по адресу, указанному при его запуске.

---

## 7. Развертывание в Kubernetes с помощью Helm

Все скрипты деплоя расположены в директории `deploy/scripts/k8s/`. Скрипты
являются идемпотентными — при повторном запуске используется `helm upgrade
--install`.

Файлы Helm-чартов:

| Путь | Назначение |
|------|-----------|
| `deploy/helm/distributed-crawler/` | Чарт приложения |
| `deploy/helm/infra/` | Чарт инфраструктуры |
| `deploy/helm/distributed-crawler/values.yaml` | Базовые значения |
| `deploy/helm/distributed-crawler/values-dev.yaml` | Наложение для разработки |
| `deploy/helm/distributed-crawler/values-prod.yaml` | Наложение для production |
| `deploy/helm/distributed-crawler/values-external-infra.yaml` | Наложение при внешней инфраструктуре |

### 7.1 Шаг 1. Сборка Docker-образов

Перед деплоем необходимо собрать Docker-образы всех компонентов приложения.
Dockerfile для каждого компонента расположен в директории `docker/<имя_компонента>/Dockerfile`.

**Сборка всех образов:**
```bash
./deploy/scripts/k8s/build-images.sh
```

**Сборка образов напрямую в Docker-демон minikube** (образы не нужно
загружать в реестр — они сразу доступны кластеру):
```bash
./deploy/scripts/k8s/build-images.sh --minikube
```

**Сборка отдельных компонентов:**
```bash
./deploy/scripts/k8s/build-images.sh grpc-server fetch-worker
```

**Принудительная пересборка без кэша:**
```bash
NO_CACHE=1 ./deploy/scripts/k8s/build-images.sh --minikube
```

Доступные имена компонентов: `grpc-server`, `fetch-worker`, `parser-worker`,
`export-worker`, `ui`.

Переменные окружения скрипта:

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `DOCKER_REGISTRY` | `distributed-crawler` | Префикс имени образа |
| `IMAGE_TAG` | `latest` | Тег образа |
| `NO_CACHE` | `0` | Установить `1` для сборки без кэша |

### 7.2 Шаг 2. Загрузка образов в реестр (для production)

Этот шаг необходим только при деплое в удалённый кластер. При использовании
minikube образы уже находятся внутри кластера после шага 7.1.

```bash
TARGET_REGISTRY=ghcr.io/myorg IMAGE_TAG=v1.2.3 ./deploy/scripts/k8s/push-images.sh
```

Загрузка отдельных компонентов:
```bash
TARGET_REGISTRY=ghcr.io/myorg ./deploy/scripts/k8s/push-images.sh grpc-server fetch-worker
```

Переменные окружения скрипта:

| Переменная | Обязательная | Описание |
|-----------|-------------|---------|
| `TARGET_REGISTRY` | Да | Адрес реестра назначения (например, `ghcr.io/myorg`) |
| `DOCKER_REGISTRY` | Нет (`distributed-crawler`) | Префикс исходного образа |
| `IMAGE_TAG` | Нет (`latest`) | Тег образа |

После загрузки необходимо указать адрес реестра в файле values перед деплоем:
```yaml
# values-prod.yaml или пользовательский файл
grpcServer:
  image:
    repository: ghcr.io/myorg/grpc-server
    tag: "v1.2.3"
fetchWorker:
  image:
    repository: ghcr.io/myorg/fetch-worker
    tag: "v1.2.3"
# ... и т.д. для остальных компонентов
```

### 7.3 Шаг 3. Развертывание инфраструктуры

Чарт `deploy/helm/infra/` разворачивает все инфраструктурные сервисы в
отдельный namespace `infra`. Этот шаг нужен при использовании «режима внешней
инфраструктуры» (рекомендуется для разработки).

**Разработка (минимальные ресурсы):**
```bash
./deploy/scripts/k8s/deploy-infra.sh
```

**Production:**
```bash
VALUES_ENV=prod ./deploy/scripts/k8s/deploy-infra.sh
```

**Пользовательский namespace:**
```bash
NAMESPACE=my-infra VALUES_ENV=prod ./deploy/scripts/k8s/deploy-infra.sh
```

**С переопределением отдельных параметров:**
```bash
./deploy/scripts/k8s/deploy-infra.sh --set postgresql.auth.password=mysecret
```

Переменные окружения скрипта:

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `RELEASE_NAME` | `infra` | Имя Helm-релиза |
| `NAMESPACE` | `infra` | Kubernetes namespace |
| `VALUES_ENV` | `dev` | Наложение значений: `dev` или `prod` |

Разворачиваемые сервисы: PostgreSQL, RabbitMQ, MinIO, Redis, RedisInsight,
OTel Collector, Jaeger, Prometheus, Grafana, OpenSearch, OpenSearch Dashboards.

### 7.4 Шаг 4. Настройка секретов перед деплоем приложения

Перед деплоем приложения необходимо задать безопасные пароли. Это можно
сделать одним из двух способов.

**Способ А — переопределить значения через `--set` или пользовательский файл values:**

```yaml
# my-secrets.yaml (не добавлять в git)
secrets:
  postgres:
    password: "мой-надёжный-пароль"
  rabbitmq:
    password: "мой-надёжный-пароль"
  minio:
    user: "minioadmin"
    password: "мой-надёжный-пароль"
  redis:
    password: "мой-надёжный-пароль"
  auth:
    jwtSecret: "случайная-строка-не-менее-32-символов"
    defaultPassword: "пароль-первого-администратора"
```

**Способ Б — указать имя существующего Kubernetes Secret:**

```yaml
secrets:
  create: false
  existingSecret: "my-crawler-secrets"
```

### 7.5 Шаг 5. Развертывание приложения

Существует два режима развертывания приложения.

**Режим А: внешняя инфраструктура (рекомендуется)**

Используется, если инфраструктура уже развёрнута через `deploy-infra.sh`
(шаг 7.3). Приложение указывает на сервисы в namespace `infra`.

```bash
EXTERNAL_INFRA=true ./deploy/scripts/k8s/deploy-all.sh
```

**Режим Б: самодостаточный режим**

Приложение разворачивается вместе со встроенными подчартами Bitnami
(PostgreSQL, RabbitMQ, MinIO, Redis). Отдельный деплой инфраструктуры не нужен.

```bash
./deploy/scripts/k8s/deploy-all.sh
```

**Production-развертывание:**
```bash
VALUES_ENV=prod EXTERNAL_INFRA=true ./deploy/scripts/k8s/deploy-all.sh
```

**С подключением файла секретов:**
```bash
EXTERNAL_INFRA=true ./deploy/scripts/k8s/deploy-all.sh -f my-secrets.yaml
```

Переменные окружения скрипта:

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `RELEASE_NAME` | `crawler` | Имя Helm-релиза |
| `NAMESPACE` | `crawler` | Kubernetes namespace |
| `VALUES_ENV` | `dev` | Наложение значений: `dev` или `prod` |
| `EXTERNAL_INFRA` | `false` | `true` — использовать отдельный релиз инфраструктуры |

### 7.6 Деплой отдельного компонента

Для обновления одного компонента без передеплоя всего стека:

```bash
./deploy/scripts/k8s/deploy-component.sh grpc-server
./deploy/scripts/k8s/deploy-component.sh fetch-worker
./deploy/scripts/k8s/deploy-component.sh parser-worker
./deploy/scripts/k8s/deploy-component.sh export-worker
./deploy/scripts/k8s/deploy-component.sh ui
```

С переопределением параметров Helm:
```bash
./deploy/scripts/k8s/deploy-component.sh fetch-worker --set fetchWorker.replicaCount=3
```

### 7.7 Проверка состояния после деплоя

Просмотр запущенных подов:
```bash
kubectl get pods -n crawler
kubectl get pods -n infra
```

Просмотр логов компонента:
```bash
kubectl logs -n crawler -l app.kubernetes.io/component=grpc-server -f
kubectl logs -n crawler -l app.kubernetes.io/component=fetch-worker -f
```

### 7.8 Проброс портов для локального доступа

Для доступа к сервисам кластера с локальной машины без настройки Ingress
используется скрипт:

```bash
# Пробросить порты всех сервисов
./deploy/scripts/k8s/port-forward.sh

# Пробросить порты выбранных сервисов
./deploy/scripts/k8s/port-forward.sh postgresql rabbitmq minio
./deploy/scripts/k8s/port-forward.sh grpc-server jaeger grafana
```

Нажать Ctrl+C для остановки всех проброшенных соединений.

Доступные имена сервисов и соответствующие локальные порты:

| Имя сервиса | Локальный порт |
|-------------|---------------|
| `postgresql` | `54322` |
| `rabbitmq` | `5672`, `15672` (UI) |
| `minio` | `9000`, `9001` (UI) |
| `redis` | `6379` |
| `redisinsight` | `8001` |
| `jaeger` | `16686` |
| `prometheus` | `9090` |
| `grafana` | `3000` |
| `opensearch` | `9200` |
| `opensearch-dashboards` | `5601` |
| `grpc-server` | `8083` (gRPC), `8084` (HTTP) |
| `ui` | `8080` |

### 7.9 Настройка Ingress (опционально)

Для доступа через доменное имя без проброса портов необходимо включить Ingress
в файле values:

```yaml
grpcServer:
  ingress:
    enabled: true
    className: "nginx"
    hosts:
      - host: crawler-api.example.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: crawler-api-tls
        hosts:
          - crawler-api.example.com

ui:
  ingress:
    enabled: true
    className: "nginx"
    hosts:
      - host: crawler.example.com
        paths:
          - path: /
            pathType: Prefix
```

---

## 8. Адреса сервисов и интерфейсов

После успешного деплоя и проброса портов сервисы доступны по следующим адресам:

| Сервис | URL | Учётные данные |
|--------|-----|---------------|
| Веб-интерфейс | `http://localhost:8080` | JWT (email администратора из values) |
| gRPC API | `localhost:8083` | JWT-токен в заголовке `Authorization: Bearer <token>` |
| HTTP-шлюз API | `http://localhost:8084` | То же |
| RabbitMQ UI | `http://localhost:15672` | `guest` / пароль из `secrets.rabbitmq.password` |
| MinIO Console | `http://localhost:9001` | `minioadmin` / пароль из `secrets.minio.password` |
| RedisInsight | `http://localhost:8001` | — |
| Jaeger UI | `http://localhost:16686` | — |
| Prometheus | `http://localhost:9090` | — |
| Grafana | `http://localhost:3000` | `admin` / пароль из `GRAFANA_PWD` |
| OpenSearch | `http://localhost:9200` | — |
| OpenSearch Dashboards | `http://localhost:5601` | — |

Для получения JWT-токена администратора необходимо выполнить запрос к
HTTP-шлюзу:
```bash
curl -X POST http://localhost:8084/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"<DEFAULT_USER_PWD>"}'
```

---

## 9. Управление миграциями базы данных

### Локально

```bash
# Применить все новые миграции
make local-migration-up

# Откатить последнюю миграцию
make local-migration-down

# Просмотреть статус миграций
make local-migration-status

# Создать новую миграцию
make local-migration-create NAME=add_something
```

Файлы миграций расположены в директории:
`internal/infra/persistence/postgres/migrations/`

### В Kubernetes

Миграции выполняются автоматически при деплое приложения в виде Kubernetes Job
(`migrations.enabled: true` в values). Job использует тот же образ, что и
`grpc-server`, и применяет все ожидающие миграции перед запуском основных
подов.

Для отключения автоматических миграций при деплое задать в values:
```yaml
migrations:
  enabled: false
```

---

## 10. Остановка и удаление

### Docker Compose

Остановить отдельные компоненты приложения:
```bash
./deploy/scripts/local/stop-all.sh
```

Остановить конкретный компонент:
```bash
./deploy/scripts/local/stop-all.sh grpc_server
```

Остановить и удалить инфраструктурные контейнеры:
```bash
docker compose down
```

Остановить контейнеры и удалить все тома с данными:
```bash
docker compose down -v
```

### Kubernetes

Удалить все Helm-релизы (приложение и инфраструктура):
```bash
./deploy/scripts/k8s/teardown.sh
```

Удалить только приложение:
```bash
APP_ONLY=true ./deploy/scripts/k8s/teardown.sh
```

Удалить только инфраструктуру:
```bash
INFRA_ONLY=true ./deploy/scripts/k8s/teardown.sh
```

Переменные окружения скрипта:

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `APP_NAMESPACE` | `crawler` | Namespace приложения |
| `INFRA_NAMESPACE` | `infra` | Namespace инфраструктуры |
| `APP_ONLY` | `false` | `true` — удалить только приложение |
| `INFRA_ONLY` | `false` | `true` — удалить только инфраструктуру |

> **Внимание.** Удаление Helm-релизов не удаляет PersistentVolumeClaims. Для
> полного удаления данных необходимо дополнительно выполнить:
> ```bash
> kubectl delete pvc --all -n crawler
> kubectl delete pvc --all -n infra
> ```
