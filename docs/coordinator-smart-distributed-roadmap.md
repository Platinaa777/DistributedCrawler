# Coordinator (gRPC Server) как "умный" distributed control plane

## Что уже есть в проекте (сильная база)

По коду видно, что у проекта уже есть важные distributed-элементы:

- `grpc-server` поднимает API + фоновые coordinator-задачи (`OutboxPublisher`, `ScheduleWorker`) в `internal/app/api_app.go`.
- Есть `WorkerService` с bidirectional stream (`heartbeat`, `drain`, `force-kill`) и реестр воркеров (`internal/api/worker/worker_impl.go`, `internal/workerhealth/registry.go`).
- Есть `outbox` паттерн для публикации задач в RabbitMQ (`internal/worker/outbox_publisher.go`).
- Есть `FOR UPDATE SKIP LOCKED` для outbox и export-claim, что уже помогает при конкурентной обработке (`internal/infra/persistence/postgres/repos/outbox_repo.go`, `internal/infra/persistence/postgres/repos/crawl_job_repo.go`).
- Есть задел под DB-sharding по `job_id` (`internal/interceptor/shard_key.go`, `internal/infra/persistence/postgres/pg/sharded_client.go`).

Это хороший фундамент. Проблема не в том, что система "не распределённая", а в том, что coordinator пока почти не принимает решений по placement/routing и слабо использует информацию о воркерах.

## Почему сейчас ощущается "просто читает топик"

Сейчас data plane в основном работает так:

- `fetch` worker читает фиксированную очередь `crawl_queue`.
- `parser` worker читает фиксированную очередь `parsing_queue`.
- Coordinator не назначает задачи конкретным воркерам/регионам, а только публикует события и принимает heartbeats.

Из-за этого coordinator пока больше похож на:

- API + background publisher/scheduler
- мониторинг/ручное управление воркерами

а не на полноценный control plane (placement, rebalance, failover, гео-стратегии, capacity-aware routing).

## Главные ограничения (по текущему коду)

### 1. `grpc-server` масштабируется плохо как coordinator

В `internal/app/api_app.go` каждый экземпляр `grpc-server` запускает:

- `runWorker()` -> `OutboxPublisher`
- `runScheduleWorker()` -> `ScheduleWorker`

Это значит:

- если поднять `grpcServer.replicaCount > 1`, каждый pod станет scheduler/coordinator одновременно;
- `OutboxPublisher` частично защищён DB-locking (`SKIP LOCKED`);
- `ScheduleWorker` сейчас читает scheduled configs через `ListAllScheduled(...)` без claim/lease/locking, что может давать гонки/дубликаты при multi-replica coordinator.

### 2. Worker telemetry почти не используется для принятия решений

У вас уже есть `WorkerHeartbeat.active_tasks` в `api/v1/models.proto`, но:

- `internal/worker/monitoring.go` не заполняет `active_tasks` в heartbeat;
- `internal/api/worker/worker_impl.go` не сохраняет `active_tasks` в registry;
- `ListWorkers` фактически не отдаёт полезную нагрузочную метрику.

То есть "умный" coordinator пока не видит текущую загрузку воркеров.

### 3. Очереди не партиционированы по шард/регион/типу capability

Сейчас routing в RMQ фактически фиксированный:

- `crawl_queue`
- `parsing_queue`
- `export_queue`

Нет маршрутизации по:

- региону (`us-east`, `eu-central`, ...)
- типу воркера/капа (`http`, `browser`, `residential-proxy`, ...)
- домену/tenant/shard

### 4. DB-sharding есть, но control plane пока не "shard-aware"

Шардирование в PostgreSQL уже есть по shard key, но coordinator пока не использует это для:

- маршрутизации очередей,
- локальности обработки,
- распределения воркеров по shard ownership.

## Что добавить в coordinator, чтобы он стал "умнее"

Ниже список фич, от практичных (быстрый эффект) до геораспределённых.

## Приоритет 1 (сразу даст ощущение настоящего coordinator)

### 1. Leader election / lease для coordinator-задач

Сделать так, чтобы `ScheduleWorker` и (по желанию) `OutboxPublisher` запускались только у leader coordinator.

Варианты:

- PostgreSQL advisory lock (`pg_try_advisory_lock`)
- таблица `coordinator_leases` + heartbeat lease
- Kubernetes `Lease` (coordination.k8s.io)

Что это даст:

- можно безопасно масштабировать `grpc-server` по API-трафику;
- scheduler не будет дублировать запуск scheduled jobs.

Минимальный шаг:

- разделить роли `grpc-server`:
  - `api-only`
  - `api+leader-workers` (или отдельный deployment `coordinator-runtime`)

### 2. Scheduler claim-модель вместо полного сканирования

Сейчас `ScheduleWorker` делает `ListAllScheduled(limit, offset)` и сам решает, пора ли запускать.

Лучше:

- в БД выбирать только due-конфиги (`next_run_at <= now`)
- брать их через `FOR UPDATE SKIP LOCKED`
- atomically обновлять `next_run_at/last_run_at`

Это сразу делает scheduling distributed-safe.

### 3. Реально использовать heartbeat для load-aware решений

Расширить heartbeat и registry, чтобы coordinator видел:

- `active_tasks`
- `max_concurrency`
- `queue_lag_local` (опционально)
- `cpu/mem` (если хотите через worker self-report)
- `region`, `zone`, `node`, `capabilities`

Тогда coordinator сможет:

- выбирать, кого drainить первым;
- делать canary/rolling drain;
- позже добавлять rebalance.

## Приоритет 2 (умный routing вместо "все читают один топик")

### 4. Ввести routing key и партиционирование очередей

Перейти от одной очереди на стадию к модели:

- exchange + routing key
- несколько очередей по регионам/шардам/типам воркеров

Примеры:

- `crawl.us-east.http`
- `crawl.eu-central.browser`
- `crawl.shard.03`
- `parse.us-east`

Coordinator (или outbox publisher) будет выбирать routing key на основе политики.

### 5. Placement policy в coordinator

Добавить модуль `PlacementPolicy` (coordinator-side), который решает, куда отправить задачу:

- по домену (sticky routing, чтобы кеш robots/rate-limit был локальный)
- по региону (RTT к target-сайту, data residency)
- по capability (browser-only / http-only)
- по загрузке воркеров
- по стоимости (дешёвый регион по умолчанию)

Это можно внедрить без полного отказа от RMQ:

- coordinator публикует в нужную очередь
- воркеры продолжают pull-consume

### 6. Sticky domain ownership (очень полезно для crawler)

Назначать домен/host на "owner shard/region" (consistent hashing).

Плюсы:

- лучше reuse кешей (`robots.txt`, rate-limit state)
- меньше cross-region дубликатов
- более стабильная нагрузка

## Приоритет 3 (геораспределённость)

### 7. Разделить control plane и data plane по регионам

Рекомендуемая модель:

- **Global coordinator**: принимает job, хранит metadata/policy, решает placement.
- **Regional coordinators**: управляют локальными воркерами, очередями и локальными ограничениями.
- **Regional workers**: fetch/parser/export в своём регионе.

Что держать локально в регионе:

- RabbitMQ
- Redis (rate-limit, robots cache)
- MinIO/S3 bucket (или региональный bucket)
- worker pools

Что держать глобально:

- job metadata / configs
- глобальные policy/quotas
- observability aggregation

### 8. Региональные failover-политики

Coordinator должен уметь:

- пометить регион degraded/unavailable
- перестроить routing на соседний регион
- drainнуть воркеры региона перед maintenance
- ограничить rate при росте ошибок/429

У вас уже есть `DrainWorker` / `ForceKillWorker`, это можно расширить до групповых команд:

- `DrainRegion`
- `DrainWorkerType`
- `SetWorkerPoolWeight`

### 9. Data residency / compliance-aware placement

Для некоторых задач важно хранить/обрабатывать данные в конкретной географии.

Добавьте в job config:

- `preferred_regions`
- `allowed_regions`
- `data_residency_policy`

Coordinator будет валидировать placement и не отправлять задачи "куда попало".

## Приоритет 4 (надежность и "умность" выполнения)

### 10. Task leasing / claim-state поверх queue semantics

Сейчас основная семантика retry идёт через RMQ ack/nack. Для сложных сценариев полезно иметь DB lease:

- `claimed_by`
- `claimed_until`
- `attempt_no`
- `last_error_type`

Это помогает:

- при зависших воркерах
- при cross-region failover
- при ручном rebalance

### 11. DLQ / retry policy per stage + circuit breaker

Добавить на coordinator/publisher уровне:

- DLQ для `crawl` и `parse`
- retry tiers (быстро/медленно)
- circuit breaker по домену/региону (например, массовые `429/403/5xx`)

Coordinator может временно:

- снижать вес региона
- откладывать задачи домена
- перенаправлять на proxy-capable workers

### 12. Backpressure и admission control

Coordinator должен уметь сказать "стоп" созданию новых задач, если:

- queue lag слишком высокий
- parser отстаёт от fetch
- MinIO/Postgres деградируют

Это особенно полезно для recursive crawl, где parser генерирует много новых задач.

## Приоритет 5 (наблюдаемость как источник решений)

### 13. Метрики для control plane decisions

Добавьте метрики, которые coordinator реально использует:

- queue lag по очередям/регионам
- in-flight tasks по worker type/region
- p95 fetch latency по доменам
- error rate / 429 rate / robots deny rate
- scheduler due backlog
- outbox publish lag (`now - occurred_at`)

Сейчас у вас уже есть OTel и часть worker metrics, это можно довести до policy-driven autoscaling и routing.

### 14. Worker capabilities и version-awareness

Heartbeat должен содержать:

- `worker_version`
- `supported_features` (например, `browser_fetch`, `captcha_solver`, `js_render`)
- `region/zone`
- `max_concurrency`

Coordinator сможет делать:

- canary rollout по версии
- совместимость задач с feature flags
- постепенный drain старой версии

## Конкретно что менять в вашем проекте (точки входа)

### A. Разделить API и coordinator-runtime

Файлы:

- `internal/app/api_app.go`
- `deploy/helm/distributed-crawler/templates/grpc-server/deployment.yaml`
- `deploy/helm/distributed-crawler/values.yaml`

Что добавить:

- флаги/ENV:
  - `ENABLE_OUTBOX_PUBLISHER=true/false`
  - `ENABLE_SCHEDULE_WORKER=true/false`
  - `COORDINATOR_LEADER_ELECTION=true/false`
- отдельный deployment для leader/background роли (или отдельный бинарь `coordinator_worker`)

### B. Сделать scheduler multi-replica safe

Файлы:

- `internal/worker/scheduler_worker.go`
- `internal/infra/persistence/postgres/repos/crawl_job_config_repo.go`

Что добавить:

- repo-метод типа `ClaimDueScheduledConfigs(now, limit)` с `FOR UPDATE SKIP LOCKED`
- (опционально) optimistic lock / version field у `crawl_job_configs.schedule`

### C. Превратить `WorkerService` в настоящий control plane API

Файлы:

- `api/v1/models.proto`
- `api/v1/service.proto`
- `internal/api/worker/worker_impl.go`
- `internal/workerhealth/registry.go`
- `internal/worker/monitoring.go`

Что добавить в `WorkerHeartbeat`:

- `region`
- `zone`
- `max_concurrency`
- `queue_names`
- `capabilities`
- `worker_version`
- корректная отправка `active_tasks`

Что добавить в `WorkerService`:

- `ListWorkerPools`
- `SetWorkerPoolWeight`
- `DrainWorkersBySelector`
- `ListQueueBacklog` (или отдельный `CoordinatorService`)

### D. Сделать routing policy в outbox publisher / coordinator

Файлы:

- `internal/worker/outbox_publisher.go`
- `internal/infra/messaging/rabbitmq/client.go`
- `internal/infra/messaging/rabbitmq/messages.go`

Что добавить:

- exchange + routing key вместо publish в одну очередь
- policy-функцию: `ResolveRoute(task, jobConfig, policyCtx) -> routingKey`
- headers (region, shard, capability)
- DLQ/TTL/retry queues

### E. Сделать geo-ready job config (минимально)

Файлы:

- `api/v1/models.proto`
- domain models/config converters

Поля (пример):

- `preferred_regions`
- `allowed_regions`
- `placement_strategy` (`latency`, `sticky_domain`, `cost_optimized`)
- `failover_regions`

## Практичный roadmap (без overengineering)

### Этап 1 (1-2 недели) — "Coordinator уже умнее"

- Разделить API и background coordinator роли через ENV flags.
- Добавить leader election.
- Исправить heartbeat/load метрики (`active_tasks`) и хранение их в registry.
- Добавить worker labels: `region`, `version`, `capabilities`.

Результат:

- можно безопасно масштабировать `grpc-server`;
- coordinator начинает принимать решения на основе фактов, а не только "жив/мертв".

### Этап 2 (2-4 недели) — "Routing/placement"

- Exchange + routing key в RabbitMQ.
- Несколько очередей по region/capability.
- Простая placement policy (sticky domain + fallback region).
- Метрики lag/error-rate для decisions.

Результат:

- система становится реально распределённой по маршрутизации, а не только по числу pod'ов.

### Этап 3 (4+ недель) — "Geo-distributed"

- Regional coordinators + local queues/caches.
- Global placement policy.
- Region failover / pool weights / selective drain.
- Compliance-aware placement.

Результат:

- настоящий geo-aware control plane.

## Короткий вывод

Ваш проект уже имеет хороший фундамент распределённой системы (outbox, worker control stream, шардирование, telemetry). Самый большой шаг вперёд для "умного coordinator" — это не переписать всё, а:

1. безопасно масштабировать coordinator (leader election + scheduler claims),
2. начать использовать нагрузочные и capability-метаданные воркеров,
3. добавить routing/placement policy (region/shard/capability-aware),
4. затем вынести это в geo-distributed control plane.

Именно эти шаги уберут ощущение "просто читает топик и всё".
