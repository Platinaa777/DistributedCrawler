# Архитектурные неточности (точечный аудит)

Дата аудита: 2026-02-26

Фокус проверки:
1. Маппинг колонок БД (строковые литералы vs константы/шаблоны)
2. Использование concrete-реализаций вместо интерфейсов
3. Обход доменных (DDD) методов через прямые присваивания

## 1. Маппинг колонок БД сделан непоследовательно

### 1.1 `crawl_job_config_repo`: для join-таблицы колонки захардкожены строками

- `internal/infra/persistence/postgres/repos/crawl_job_config_repo.go:18` - есть константы для основной таблицы `crawl_job_configs`.
- `internal/infra/persistence/postgres/repos/crawl_job_config_repo.go:291` - raw SQL с literal-колонками `crawl_job_config_id`, `queue_endpoint_id`, `weight`.
- `internal/infra/persistence/postgres/repos/crawl_job_config_repo.go:305` - `DELETE ... WHERE crawl_job_config_id = $1` через строковый literal.
- `internal/infra/persistence/postgres/repos/crawl_job_config_repo.go:320` - `SELECT queue_endpoint_id, weight ...` через literal-колонки.
- `internal/infra/persistence/postgres/repos/crawl_job_config_repo.go:336` - positional `rows.Scan(&a.EndpointID, &a.Weight)` привязан к порядку в SELECT.
- `internal/infra/persistence/postgres/repos/crawl_job_config_repo.go:359` - batch-SELECT снова повторяет literal-колонки.
- `internal/infra/persistence/postgres/repos/crawl_job_config_repo.go:376` - positional `Scan` привязан к order и alias-less колонкам.

Проблема:
- Для основной таблицы уже выбран подход с `const`-колонками, но join-таблица выпала из этого паттерна.
- Любое переименование колонки в миграции требует ручного поиска по raw SQL и по порядку `Scan`.
- Повышается риск тихого рассинхрона (особенно в batch-версии запроса).

Рекомендация:
- Добавить константы колонок join-таблицы (`configQueueEndpointConfigIDColumn`, `configQueueEndpointIDColumn`, `configQueueEndpointWeightColumn`).
- Собрать общий `[]string`/helper для SELECT-полей join-таблицы и единый scan-helper.

### 1.2 `queue_endpoint_repo`: константы объявлены, но часть SQL все равно дублирует literal-колонки

- `internal/infra/persistence/postgres/repos/queue_endpoint_repo.go:18` - объявлены константы колонок `queue_endpoints`/`queue_routing_rules`.
- `internal/infra/persistence/postgres/repos/queue_endpoint_repo.go:65` - `RETURNING id, display_name, ...` строкой, несмотря на `colQE*`.
- `internal/infra/persistence/postgres/repos/queue_endpoint_repo.go:89` - `Where(sq.Eq{"id": id})` использует literal `"id"` вместо `colQEID`.
- `internal/infra/persistence/postgres/repos/queue_endpoint_repo.go:157` - второй `RETURNING ...` снова строкой.
- `internal/infra/persistence/postgres/repos/queue_endpoint_repo.go:254` - `Upsert` для routing rules возвращает literal-список `RETURNING id, stage, scope, enabled`.
- `internal/infra/persistence/postgres/repos/queue_endpoint_repo.go:264` - прямой `row.Scan(...)` без общего scan-helper для routing rule.

Проблема:
- Внутри одного файла смешаны два стиля: `const`-колонки и строковые списки.
- `RETURNING`-списки дублируют структуру `scanQueueEndpoint(...)`; при добавлении/перестановке колонок легко сломать scan.

Рекомендация:
- Вынести `RETURNING`-списки в константы/хелперы (`queueEndpointReturningColumns`, `queueRoutingRuleReturningColumns`).
- Использовать `colQEID` в `Where(...)` для единообразия.

### 1.3 `crawl_task_repo.Get`: кросс-табличный SELECT и scan завязаны на строковые литералы и порядок полей

- `internal/infra/persistence/postgres/repos/crawl_task_repo.go:145` - `sq.Select(...)` с literal-полями `"t.id"`, `"t.job_id"`, ... и `"j.id"`, `"j.job_config_id"`, ...
- `internal/infra/persistence/postgres/repos/crawl_task_repo.go:152` - join захардкожен строкой `"crawl_jobs j ON t.job_id = j.id"`.
- `internal/infra/persistence/postgres/repos/crawl_task_repo.go:153` - `Where(sq.Eq{"t.id": id.String()})` literal вместо констант.
- `internal/infra/persistence/postgres/repos/crawl_task_repo.go:169` - длинный positional `Scan(...)`, полностью завязанный на порядок SELECT.
- `internal/infra/persistence/postgres/repos/crawl_task_repo.go:187` - scan job-снапшота завершает общий scan-цепочку; любое изменение списка полей ломает все сразу.

Контекст, почему это именно архитектурная неточность:
- В `internal/infra/persistence/postgres/repos/crawl_task_repo.go:16` и ниже уже есть `task*Column` константы.
- В `internal/infra/persistence/postgres/repos/crawl_job_repo.go:20` и ниже уже есть отдельные `crawl_jobs`-константы, но они недоступны/не переиспользуются в `crawl_task_repo.Get`.

Рекомендация:
- Вынести reusable column-list builders/const slices для task/job select-полей (с alias).
- Отдельный helper для `Scan task + job` (или `ScanOneContext` в snapshot с `db` tags/aliases).

## 2. Используется concrete-реализация там, где лучше узкий interface

### 2.1 `QueueAdminImplementation` прибит к `*appqueue.Service`

- `internal/api/queue_admin/queue_admin_impl.go:14` - поле `service *appqueue.Service`
- `internal/api/queue_admin/queue_admin_impl.go:18` - конструктор принимает `*appqueue.Service`

Проблема:
- gRPC adapter зависит от конкретного application service-типа, а не от контракта.
- Сложнее unit-test (нужен реальный `*appqueue.Service` или обертки).
- Любой рефакторинг `appqueue.Service` (даже неиспользуемых методов/полей) тащит риск в API слой.

Почему здесь интерфейс уместен:
- `QueueAdminImplementation` использует только ограниченный набор методов (`ListEndpoints`, `CreateEndpoint`, `UpdateEndpoint`, `DeleteEndpoint`, `ListRoutingRules`, `UpsertRoutingRule`).

Рекомендация:
- В `internal/api/queue_admin` объявить локальный интерфейс (например `QueueAdminService`) с 6 используемыми методами.
- Оставить `appqueue.Service` как одну из реализаций этого интерфейса.

### 2.2 `WorkerImplementation` прибит к `*workerhealth.Registry`

- `internal/api/worker/worker_impl.go:21` - поле `registry *workerhealth.Registry`
- `internal/api/worker/worker_impl.go:25` - конструктор принимает `*workerhealth.Registry`

Проблема:
- gRPC слой знает точный concrete-тип registry.
- Нельзя подменить registry минимальным mock/fake без использования реального `workerhealth.Registry`.
- Интерфейс упростил бы тесты сценариев `ListWorkers`, `DrainWorker`, `ForceKillWorker`, `WorkerStream`.

Рекомендация:
- В `internal/api/worker` объявить узкий интерфейс под реально используемые методы registry.
- Concrete `workerhealth.Registry` оставлять в composition root (`internal/app/...`), а не в API-адаптере.

### 2.3 Контраст внутри API-слоя: местами уже сделано правильно

- `internal/api/crawl_job/job_impl.go:9` - локальный интерфейс `PresignedURLGenerator`.
- `internal/api/crawl_job/job_impl.go:15` - зависимости через интерфейсы `service.CrawlJobService`, `service.CrawlTaskService`.

Вывод:
- В проекте уже есть правильный паттерн инъекции зависимостей через интерфейсы в API-адаптеры.
- `queue_admin` и `worker` сейчас архитектурно выбиваются из принятого подхода.

## 3. DDD-методы доменной модели обходятся прямыми присваиваниями

### 3.1 Реальный баг в доменной модели: `MarkAsParsed(...)` не завершает переход состояния

- `internal/domain/crawl/models/crawl_task.go:26` - у модели есть поле `ResultCreatedAt`.
- `internal/domain/crawl/models/crawl_task.go:39` - `MarkAsParsed(..., time time.Time)` принимает время перехода.
- `internal/domain/crawl/models/crawl_task.go:40` - меняет `Status`.
- `internal/domain/crawl/models/crawl_task.go:41` - заполняет `ResultObjectKey`.
- `internal/domain/crawl/models/crawl_task.go:42` - заполняет `ResultContentType`.
- `internal/domain/crawl/models/crawl_task.go:43` - заполняет `ResultSizeBytes`.
- Поле `ResultCreatedAt` не устанавливается, параметр `time` не используется.
- `internal/worker/parser_worker.go:287` - вызывается `crawlTask.MarkAsParsed(..., time.Now())`, но переданное время теряется.

Проблема:
- Доменный метод заявлен как источник truth для перехода в `Parsed`, но не фиксирует полный инвариант состояния.
- Возможна неконсистентность: статус и result-метаданные есть, а `ResultCreatedAt == nil`.

Рекомендация:
- В `MarkAsParsed` установить `task.ResultCreatedAt = &time`.
- Лучше переименовать параметр `time` во что-то вроде `parsedAt`, чтобы убрать конфликт с именем пакета `time`.

### 3.2 `FetchWorker`: повторяющиеся прямые мутации `CrawlTask` вместо доменных переходов

- `internal/worker/fetch_worker.go:168` - `task.Status = models.TaskStatusSkipped`
- `internal/worker/fetch_worker.go:170` - `task.ErrorMessage = &errMsg`
- `internal/worker/fetch_worker.go:203` - `task.Status = models.TaskStatusFailed`
- `internal/worker/fetch_worker.go:205` - `task.ErrorMessage = &errMsg`
- `internal/worker/fetch_worker.go:240` - `task.Status = models.TaskStatusFailed`
- `internal/worker/fetch_worker.go:242` - `task.ErrorMessage = &errMsg`
- `internal/worker/fetch_worker.go:374` - `task.Status = models.TaskStatusFailed`
- `internal/worker/fetch_worker.go:376` - `task.ErrorMessage = &errMsg`
- При этом в этом же файле используется доменный метод `MarkAsFetched(...)` (см. обновление успешного кейса).

Проблема:
- Логика переходов размазана по приложению/воркеру.
- Поведение fail/skip дублируется в нескольких ветках.
- Если позже появятся дополнительные инварианты (например очистка result-полей при fail), часть мест останется неактуальной.

Рекомендация:
- Добавить методы `(*CrawlTask) MarkAsFailed(reason string)` и `MarkAsSkipped(reason string)` (или единый `Fail/Skip` API).
- Централизовать изменение `Status` + `ErrorMessage` (+ возможную очистку/нормализацию полей) внутри доменной модели.

### 3.3 `ParserWorker`: та же проблема, но рядом уже используется доменный метод `MarkAsParsed(...)`

- `internal/worker/parser_worker.go:201` - `crawlTask.Status = models.TaskStatusFailed`
- `internal/worker/parser_worker.go:203` - `crawlTask.ErrorMessage = &errMsg`
- `internal/worker/parser_worker.go:235` - `crawlTask.Status = models.TaskStatusFailed`
- `internal/worker/parser_worker.go:237` - `crawlTask.ErrorMessage = &errMsg`
- `internal/worker/parser_worker.go:287` - успешный кейс идет через `crawlTask.MarkAsParsed(...)`
- `internal/worker/parser_worker.go:330` - снова прямое присваивание `Status`
- `internal/worker/parser_worker.go:332` - снова прямое присваивание `ErrorMessage`

Проблема:
- В одном и том же use-case смешаны два стиля обновления aggregate: через метод и прямой доступ к полям.
- Это делает доменную модель частично "анемичной": часть бизнес-переходов живет в воркере.

Рекомендация:
- После фикса `MarkAsParsed(...)` добавить симметричные методы для fail-сценариев и заменить прямые присваивания в `ParserWorker`.

## Короткий приоритет исправлений

1. Исправить `CrawlTask.MarkAsParsed(...)` (`ResultCreatedAt`) - это функциональная неконсистентность, не только стиль.
2. Ввести доменные методы `MarkAsFailed/MarkAsSkipped` и убрать прямые присваивания в `fetch_worker`/`parser_worker`.
3. Нормализовать DB column mapping для join-таблиц и `RETURNING`/`SELECT` списков (константы + scan-helper).
4. В `internal/api/queue_admin` и `internal/api/worker` перейти на узкие интерфейсы зависимостей.
