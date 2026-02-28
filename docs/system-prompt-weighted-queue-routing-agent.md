# Системный промт для агента: кастомные очереди и weighted routing по регионам/нодам

Ты работаешь в репозитории `distributed-crawler` (Go backend + gRPC/gRPC-Gateway + Angular UI + workers).

Твоя задача: реализовать поддержку кастомных очередей/топиков с весами (распределением трафика) по регионам/нодам, чтобы администратор мог управлять этим через UI, а backend/coordinator/workers учитывали эти настройки при публикации и потреблении задач.

## Что уже есть в проекте (важно учитывать)

- Сейчас routing в основном фиксированный: `crawl_queue` и `parsing_queue`.
- Жесткая привязка к двум queue keys есть в `internal/config/config.go` (`crawl_queue`, `parsing_queue`).
- Имена очередей/топиков читаются из env:
  - `internal/config/env/rabbitmq.go`
  - `internal/config/env/kafka.go`
- Выбор брокера через `MESSAGING_BROKER` (`rabbitmq`, `kafka`, `grpc_memory`) в `internal/config/env/messaging.go`.
- `OutboxPublisher` публикует в одну очередь (crawl) через `internal/worker/outbox_publisher.go`.
- Fetch/Parser workers потребляют фиксированные очереди в `internal/app/worker_app.go`, `internal/worker/fetch_worker.go`, `internal/worker/parser_worker.go`.
- API/HTTP строится через gRPC + grpc-gateway:
  - proto: `api/v1/models.proto`, `api/v1/service.proto`
  - реализации: `internal/api/*`
  - регистрация сервисов: `internal/app/api_app.go`
- UI (Angular) имеет админские экраны и role guard:
  - роуты: `ui/src/app/app.routes.ts`
  - users/workers pages уже есть как пример admin UI.

## Бизнес-задача (что нужно сделать)

1. Добавить сущность "очередь" (queue endpoint/config) для администрирования через UI.
2. Поддержать несколько очередей одного типа/stage (`crawl`, `parse`) и распределение трафика по весам.
3. Очереди могут быть в разных регионах/нодах и даже на разных хостах (особенно для RabbitMQ).
4. В дефолтной конфигурации всегда должны существовать/быть заданы:
   - `crawl_queue`
   - `parse_queue` (host + queue/topic name + креды/secret reference)
5. В режиме in-memory это не работает.
   - В этом репозитории это соответствует `MESSAGING_BROKER=grpc_memory` (или memory broker mode). Для него UI/Backend должны явно показывать/возвращать "unsupported".
6. Метаданные по очередям/маршрутизации хранятся в БД, но секреты/креды не хранятся в БД в открытом виде:
   - в БД хранить ссылки/ключи для lookup секрета;
   - реальные секреты читаются из файла, проброшенного через volume (например Kubernetes Secret/ConfigMap mount).
7. Должен быть механизм watch этого файла (hot reload или мягкий reload конфигурации без полного redeploy, если возможно).

## Обязательные области изменений

### 1) Proto и API (обязательно)

Нужно изменить `api/v1/models.proto` и `api/v1/service.proto`.

Добавь новые сообщения и endpoint'ы для управления очередями и routing weights, минимум:

- `QueueEndpoint` (описание очереди/топика)
- `QueueSecretRef` (или эквивалент с metadata lookup key)
- `ListQueueEndpointsRequest/Response`
- `CreateQueueEndpointRequest/Response`
- `UpdateQueueEndpointRequest/Response`
- `DeleteQueueEndpointRequest/Response` (или soft delete / deactivate)
- `QueueTrafficRule` / `QueueRouteWeight`
- `ListQueueRoutingRulesRequest/Response`
- `UpsertQueueRoutingRulesRequest/Response`

Рекомендуемый отдельный gRPC сервис (admin-only), например:

- `QueueAdminService`

С HTTP mapping через `google.api.http`, например:

- `GET /api/v1/queues`
- `POST /api/v1/queues`
- `PATCH /api/v1/queues/{id}`
- `DELETE /api/v1/queues/{id}`
- `GET /api/v1/queue-routing`
- `PUT /api/v1/queue-routing`

Если решишь добавить ещё endpoint для preview/validation secret lookup (например `POST /api/v1/queues:validate`), это плюс.

После изменений proto обязательно регенерировать:

- `pkg/v1/*.pb.go`
- `pkg/v1/*.pb.gw.go`
- `pkg/v1/*.pb.validate.go`
- swagger (`api/v1/swagger/api.swagger.json`)

Используй существующий pipeline (`Makefile`, `buf generate`, `make generate`).

### 2) Backend domain/application/persistence

Добавь новый домен (или модуль) для очередей и правил маршрутизации.

Нужно покрыть:

- модели домена для queue endpoint и routing rules (веса по stage/region/типу очереди);
- application service для CRUD и upsert правил;
- repos + postgres snapshots + converters;
- миграции БД (`internal/infra/persistence/postgres/migrations`).

Минимально в БД должны храниться:

- идентификатор очереди
- broker type (`rabbitmq` / `kafka`)
- stage (`crawl` / `parse`) — можно enum/строка
- display name
- region / node label (или pool label)
- host/brokers metadata (не секреты)
- queue/topic name
- enabled flag
- priority/order (опционально)
- secret lookup metadata (например `secret_provider`, `secret_key`, `secret_path`, `credentials_ref`)
- timestamps/audit fields (если уместно)

И отдельно правила маршрутизации/весов:

- stage (`crawl`/`parse`)
- scope selector (минимум global; желательно `region`, `worker_type`, `job label` на будущее)
- список target queues + weight
- enabled/version

Важно:

- Валидация сумм весов (например >0, не обязательно ровно 100, если нормализуете).
- Нельзя удалить/деактивировать последнюю активную дефолтную очередь для `crawl` или `parse` без fallback.
- Должен быть безопасный fallback на текущие env-based `crawl_queue` / `parsing_queue`, если в БД ничего не настроено.

### 3) Routing logic (coordinator / outbox publisher)

Основная логика выбора очереди для публикации crawl-задач должна учитывать веса.

Сейчас публикация идет через `internal/worker/outbox_publisher.go` в один `queueName`. Нужно переделать так, чтобы:

- маршрут выбирался через policy/resolver (вынеси в отдельный компонент, не хардкодь в `OutboxPublisher`);
- policy читала активные queue endpoints + routing weights из runtime config/state;
- поддерживался weighted selection (детерминированный/sticky вариант приветствуется, например по hash(domain/task_id) + весам);
- работал fallback при недоступности/отсутствии валидного маршрута.

Желательно:

- sticky routing по домену (host) для лучшего reuse `robots.txt`/rate limit cache;
- метрики и логи: выбранная очередь, stage, region, причина fallback.

### 4) Workers (fetch/parser)

Нужно обновить воркеры так, чтобы они могли работать с кастомизированными очередями:

- не только с фиксированными env queue names;
- учитывать конфигурацию очередей/подписок для своего типа/stage;
- для `grpc_memory` режима вернуть/логировать unsupported и использовать старое поведение без новых фич.

Зафиксированный вариант реализации (обязательно):

- Ввести абстракцию региона для воркеров (`region` / logical node pool).
- Добавить обязательный env для всех worker-процессов, например `WORKER_REGION`.
- Воркер consume'ит только очереди своего региона (и своего stage/типа воркера).
- Многорегиональность/многопуловость обеспечивается через несколько deployments/нод, где каждый deployment имеет свой `WORKER_REGION`.

MVP-правило для воркеров:

- fetch worker читает `crawl`-очереди только своего региона;
- parser worker читает `parse`-очереди только своего региона;
- если для региона нет активной очереди нужного stage, использовать безопасный fallback (если явно разрешено конфигом) или запускаться с понятной ошибкой конфигурации.

Что нужно реализовать и задокументировать:

- как `WORKER_REGION` валидируется при старте;
- как выбирается queue subscription для региона;
- как ведет себя worker при изменении конфигурации региона/очередей (restart consume loop / soft reload / restart process).

### 5) Файл конфигурации секретов / volume + watch (обязательно упомянуть и реализовать MVP)

Нужно добавить механизм внешнего файла (через volume), из которого приложение читает секретные данные для queue endpoints.

Требования:

- БД хранит только metadata/lookup ref.
- Файл содержит секреты/креды/connection-specific sensitive values.
- Приложение следит за изменениями файла (watch или polling fallback).
- При изменении файла runtime state обновляется безопасно.

Где это нужно:

- coordinator/outbox publisher (для publish в разные RMQ/Kafka endpoints)
- при необходимости workers (если они сами подключаются к нескольким endpoint’ам)

Можно добавить env/flags, например:

- `QUEUE_SECRETS_FILE_PATH`
- `QUEUE_SECRETS_WATCH_ENABLED`
- `QUEUE_SECRETS_RELOAD_INTERVAL`
- `WORKER_REGION` (обязательный для fetch/parser worker в новом режиме)

Если в проекте нет watcher-библиотеки, можно:

- добавить `fsnotify`, или
- сделать polling по `mtime` как MVP.

### 6) UI (admin)

Добавь admin UI для управления очередями и весами.

Минимум:

- новая страница (admin-only) в Angular, по аналогии с `users`/`workers`
- список queue endpoints
- создание/редактирование queue endpoint
- форма указания:
  - broker type (`rabbitmq`/`kafka`)
  - stage (`crawl`/`parse`)
  - region/node label
  - host/brokers
  - queue/topic name
  - secret ref metadata
  - enabled
- экран/секция routing weights:
  - выбор stage
  - список target queues
  - веса (несколько очередей одновременно)
  - валидация суммы/корректности

UI файлы, которые почти наверняка нужно изменить/добавить:

- `ui/src/app/app.routes.ts`
- `ui/src/app/core/constants/api.constants.ts`
- `ui/src/app/core/services/api/*` (новый `queue-admin-api.service.ts`)
- `ui/src/app/core/models/*` (новые модели/мапперы)
- `ui/src/app/features/*` (новая feature-папка, например `queues`)

Роль доступа: только `ADMINISTRATOR`.

### 7) Service provider / wiring / registration

Добавь все wiring-изменения в backend:

- регистрация нового gRPC service в `internal/app/api_app.go`
- grpc-gateway registration в `internal/app/api_app.go`
- DI/wiring в `internal/app/service_provider.go`
- новые repos/services/api implementations

## Ограничения и требования по совместимости

1. Не ломай существующий flow для текущих `.env` / `.worker.env`.
2. Если новых queue endpoints/rules нет, система должна продолжать работать как сейчас (через `crawl_queue` / `parsing_queue` из env).
3. Для `grpc_memory` (`in_memory`) новая функциональность должна быть явно отключена/недоступна.
4. Секреты не логировать.
5. В БД не хранить plaintext credentials.

## Что именно считать "готово" (acceptance criteria)

- Admin может через UI создать несколько queue endpoints для `crawl` и `parse` (RabbitMQ/Kafka).
- Admin может задать веса распределения трафика на несколько очередей.
- Backend сохраняет metadata в БД, валидирует конфигурацию.
- Есть API endpoints (gRPC + HTTP) для CRUD очередей и управления весами.
- `OutboxPublisher` (или выделенный routing policy слой) публикует crawl tasks в выбранную очередь с учетом весов.
- Workers привязаны к региону через обязательный `WORKER_REGION` и consume'ят только очереди своего региона (MVP-уровень, но рабочий).
- `grpc_memory` режим корректно сообщает `unsupported` для queue management/routing weights.
- Есть механизм чтения секретов из файла (volume) и watch/reload (хотя бы MVP polling).
- Обновлены proto и сгенерированные файлы.
- Добавлены миграции и базовые тесты для ключевой логики (validation + routing selection + repo/service happy path).

## Рекомендуемый порядок выполнения (обязательно следуй)

1. Сначала спроектируй data model и API contract (proto + DB schema).
2. Затем реализуй backend CRUD + repos + migrations.
3. Потом routing policy + integration в `OutboxPublisher`.
4. Потом workers (consume config / subscriptions).
5. Потом UI admin screens.
6. В конце обнови Helm/env/config для volume mount и file watcher настроек.

## Что нужно явно отразить в финальном отчете агента

- Какие proto файлы изменены (`api/v1/models.proto`, `api/v1/service.proto`)
- Какие новые HTTP endpoints добавлены
- Какие миграции созданы
- Как устроен формат файла секретов и где задается путь
- Как работает watch/reload
- Как реализован fallback на старые env queue names
- Ограничения MVP (если не сделан full hot-reload consumers)
- Какие тесты запущены и что не покрыто

## Дополнительные подсказки по текущему коду (полезные точки входа)

- `internal/worker/outbox_publisher.go` — текущая публикация в одну очередь
- `internal/app/service_provider.go` — создание messaging client + outbox publisher
- `internal/app/worker_app.go` — fetch/parser worker wiring и queue name lookup
- `internal/config/env/rabbitmq.go`, `internal/config/env/kafka.go` — текущие env queue/topic names
- `api/v1/service.proto`, `api/v1/models.proto` — API/proto контракты
- `internal/api/user/*` + `ui/src/app/features/users/*` — пример admin CRUD потока
- `deploy/helm/distributed-crawler/templates/*.yaml` — env/secret wiring, место для volume mount

Работай итеративно, но доведи задачу до рабочего состояния end-to-end (backend + proto + UI + workers + migrations + basic tests). Если где-то нужен компромисс для MVP — реализуй и явно зафиксируй его.
