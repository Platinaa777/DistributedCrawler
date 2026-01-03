# Persistence Layer

Слой персистентности для distributed-crawler, реализующий паттерн Repository и абстракцию над базой данных PostgreSQL.

## Обзор

Этот модуль предоставляет чистую архитектуру для работы с базой данных, разделяя бизнес-логику от деталей инфраструктуры.

## Структура

```
persistence/
├── dbclient.go           # Интерфейс клиента базы данных
├── db.go                 # Интерфейсы для работы с БД
├── dbtransaction.go      # Интерфейс менеджера транзакций
└── postgres/
    ├── pg/
    │   ├── pgclient.go   # Реализация клиента PostgreSQL
    │   └── pgdb.go       # Реализация DB интерфейса
    ├── transaction/
    │   └── transaction.go # Реализация TxManager
    ├── repos/
    │   ├── crawl_job_repo.go
    │   ├── crawl_task_repo.go
    │   ├── page_snapshot_repo.go
    │   └── extracted_record_repo.go
    ├── snapshots/
    │   └── crawl_job_snapshot.go
    ├── converters/
    │   └── crawl_job.go
    └── migrations/
        └── 20251227213007_some.sql
```

## Основные компоненты

### Интерфейсы (абстракции)

#### `Client`
Главная точка входа для работы с БД:
```go
type Client interface {
    DB() DB
    Close() error
}
```

#### `DB`
Объединяет функциональность выполнения запросов, транзакций и пинга:
```go
type DB interface {
    SQLExecer
    Transactor
    Pinger
    Close()
}
```

#### `TxManager`
Менеджер транзакций:
```go
type TxManager interface {
    ReadCommited(ctx context.Context, exec Handler) error
}
```

### Реализация PostgreSQL

#### `pgClient`
Конкретная реализация клиента для PostgreSQL с использованием `pgxpool`.

#### `pgDb`
Реализация интерфейса DB с поддержкой:
- Автоматического выбора между транзакционным и обычным контекстом
- Логирования всех запросов
- Named queries через библиотеку `scany`

#### Transaction Manager
- Поддержка вложенных транзакций (если транзакция уже существует в контексте)
- Автоматический rollback при ошибках и panic
- Изоляция Read Committed

## Оценка кода

### ✅ Сильные стороны

1. **Чистая архитектура**
   - Четкое разделение интерфейсов и реализаций
   - Слой абстракции позволяет легко менять БД в будущем
   - Следование принципам SOLID (особенно D - Dependency Inversion)

2. **Управление транзакциями**
   - Элегантное использование context для передачи транзакции
   - Поддержка вложенных транзакций
   - Автоматический rollback при panic или ошибках
   - Правильная обработка defer для commit/rollback

3. **Композиция интерфейсов**
   - `SQLExecer = NamedExecer + QueryExecer`
   - `DB = SQLExecer + Transactor + Pinger`
   - Это дает гибкость для использования только нужной функциональности

4. **Query builder integration**
   - Использование Squirrel для безопасного построения запросов
   - Структура `Query` с именем для лучшего логирования

5. **Миграции**
   - Использование goose для версионирования схемы БД
   - Правильные внешние ключи с CASCADE
   - Индексы на FK для производительности

### ⚠️ Области для улучшения

1. **Обработка nullable полей**
   - В [converters/crawl_job.go:14](internal/infra/persistence/postgres/converters/crawl_job.go#L14) и [converters/crawl_job.go:24](internal/infra/persistence/postgres/converters/crawl_job.go#L24) есть TODO для `CompletedAt`
   - **Решение**: Использовать указатель `*time.Time` в domain модели или создать helper:
     ```go
     func SaveCrawlJobToSnapshot(crawlJob models.CrawlJob) *snapshots.CrawlJobSnapshot {
         snapshot := &snapshots.CrawlJobSnapshot{
             ID:        crawlJob.ID,
             Name:      crawlJob.Name,
             Status:    crawlJob.Status,
             CreatedAt: crawlJob.CreatedAt,
         }
         if crawlJob.CompletedAt != nil {
             snapshot.CompletedAt = sql.NullTime{
                 Time:  *crawlJob.CompletedAt,
                 Valid: true,
             }
         }
         return snapshot
     }
     ```

2. **Логирование**
   - [pgdb.go:98-104](internal/infra/persistence/postgres/pg/pgdb.go#L98-L104): используется `log.Println` вместо структурированного логгера
   - Context передается в log.Println как первый параметр, что некорректно
   - **Рекомендация**: Использовать `slog` или `zerolog` для структурированного логирования

3. **Обработка ошибок**
   - Некоторые репозитории ([crawl_task_repo.go](internal/infra/persistence/postgres/repos/crawl_task_repo.go), [extracted_record_repo.go](internal/infra/persistence/postgres/repos/extracted_record_repo.go), [page_snapshot_repo.go](internal/infra/persistence/postgres/repos/page_snapshot_repo.go)) пустые
   - В [pgclient.go:18](internal/infra/persistence/postgres/pg/pgclient.go#L18) используется `errors.Errorf` вместо `errors.Wrap` или `fmt.Errorf` с `%w`
   - **Рекомендация**: Использовать `fmt.Errorf` с `%w` для wrapping ошибок (стандарт с Go 1.13)

4. **Именование**
   - В [dbtransaction.go:6](internal/infra/persistence/dbtransaction.go#L6) опечатка: `ReadCommited` → `ReadCommitted`

5. **Константы**
   - [pgdb.go:18](internal/infra/persistence/postgres/pg/pgdb.go#L18): `TxKey` может быть unexported, если не используется вне пакета
   - Тип `key string` - хорошая практика для избежания коллизий в context

6. **Тестирование**
   - Нет тестов
   - **Рекомендация**: Добавить unit-тесты с использованием mock DB или testcontainers

### 🔧 Технические замечания

1. **Зависимости**
   - `github.com/pkg/errors` - deprecated, рекомендуется стандартный `errors` + `fmt.Errorf`
   - Все остальные зависимости актуальны

2. **Производительность**
   - Использование connection pool (pgxpool) - отлично
   - Prepared statements не используются, но для динамических запросов это нормально

3. **Безопасность**
   - Используется параметризация запросов через Squirrel - защита от SQL injection ✓
   - Placeholder format `sq.Dollar` корректен для PostgreSQL

## Использование

### Инициализация

```go
import (
    "distributed-crawler/internal/infra/persistence"
    "distributed-crawler/internal/infra/persistence/postgres/pg"
    "distributed-crawler/internal/infra/persistence/postgres/transaction"
)

// Создание клиента
client, err := pg.New(ctx, "postgres://user:pass@localhost/dbname")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Получение DB
db := client.DB()

// Создание transaction manager
txManager := transaction.NewTransactorManager(db)
```

### Работа с репозиториями

```go
import "distributed-crawler/internal/infra/persistence/postgres/repos"

// Создание репозитория
repo := repos.NewCrawlRepository(client)

// Создание записи
id, err := repo.Create(ctx, crawlJob)

// Получение записи
job, err := repo.Get(ctx, id)
```

### Работа с транзакциями

```go
err := txManager.ReadCommited(ctx, func(ctx context.Context) error {
    // Все операции внутри используют одну транзакцию
    id, err := repo.Create(ctx, job1)
    if err != nil {
        return err // автоматический rollback
    }

    _, err = repo.Create(ctx, job2)
    return err // commit при nil, rollback при ошибке
})
```

## Рекомендации по развитию

1. **Приоритет 1**: Исправить TODO с `CompletedAt`
2. **Приоритет 1**: Заменить `log.Println` на структурированное логирование
3. **Приоритет 2**: Исправить `ReadCommited` → `ReadCommitted`
4. **Приоритет 2**: Реализовать пустые репозитории
5. **Приоритет 3**: Добавить тесты
6. **Приоритет 3**: Мигрировать с `github.com/pkg/errors` на стандартные errors

## Итоговая оценка: 8/10

Код демонстрирует хорошее понимание архитектурных паттернов и чистой архитектуры. Основной функционал реализован качественно, особенно управление транзакциями. Основные минусы - незавершенная реализация некоторых компонентов и несколько технических долгов, которые легко исправить.
