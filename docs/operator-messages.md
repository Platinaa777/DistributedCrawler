# 4 Сообщения оператору

## 4.1 Общий формат ошибки сервера

Приложение логгирует любую ошибку, произошедшую на стороне сервера, в формате цепочки ошибок вида:

```
can't do something: can't do something internal: initial reason
```

Данный лог позволяет пройти по следу ошибки, увидев, что пошло не так на каждом этапе.

**Примеры цепочек из сервисного слоя:**

```
failed to create user: failed to insert user: pq: duplicate key value violates unique constraint "users_email_key"
failed to get refresh token: failed to query refresh_tokens: context deadline exceeded
failed to revoke existing tokens: failed to delete from refresh_tokens: sql: no rows in result set
can't list crawl tasks: failed to query crawl_tasks: pq: relation "crawl_tasks" does not exist
failed to generate presigned URL: RequestError: send request failed: connection refused
```

Логирование выполняется в методе-перехватчике `LogInterceptor`
(`internal/interceptor/log.go`) на уровне zap-логгера. Каждая запись
содержит gRPC-метод, код статуса и полный текст ошибки. Операторы могут
фильтровать логи по полю `grpc.method` или по уровню `ERROR`.

---

## 4.2 OK

**gRPC-код:** `0 OK`
**HTTP-эквивалент:** `200 OK`

Синхронный вызов отработал и завершился полностью корректно. Тело ответа
содержит запрошенные данные согласно контракту соответствующего метода.

**Ситуации, когда возвращается OK:**

| Метод | Что означает |
|-------|-------------|
| `CreateCrawlJob` | Задание на обход создано, ID возвращён в ответе |
| `ListCrawlJobs` | Список заданий (возможно пустой) успешно получен |
| `GetTaskFileURL` | Pre-signed URL для скачивания HTML-снимка сформирован |
| `GetJobExportFileURL` | Pre-signed URL для экспортного файла (JSON/CSV) сформирован |
| `Register` / `Login` | Пользователь зарегистрирован / аутентифицирован, токены выданы |
| `UpdateUserRole` | Роль пользователя успешно изменена |
| `UpsertQueueEndpoint` | Endpoint очереди создан или обновлён |

---

## 4.3 INVALID_ARGUMENT

**gRPC-код:** `3 INVALID_ARGUMENT`
**HTTP-эквивалент:** `400 Bad Request`

Клиент осуществил вызов с некорректными входными данными. Ошибка содержит
описание того, какой именно параметр не прошёл проверку.

**Возможные сообщения:**

| Сообщение | Причина |
|-----------|---------|
| `id is required` | Поле `id` не передано или пустая строка |
| `worker_id is required` | Поле `worker_id` не передано в запросе к воркеру |
| `invalid role` | Значение поля `role` не соответствует ни одной из допустимых ролей (`user`, `admin`) |
| `invalid cursor: <детали>` | Значение `page_token` не является корректным base64 или содержит невалидный JSON |
| `invalid file_type: <тип> (must be 'pages' or 'result')` | Запрошен тип файла задачи, отличный от `pages` или `result` |
| `invalid file_type: <тип> (must be 'json' or 'csv')` | Запрошен тип экспортного файла, отличный от `json` или `csv` |
| `seeds list cannot be empty` | Создание задания без единого стартового URL |
| `invalid scopes.allowed_url_patterns: <детали>` | Один из паттернов фильтрации URL содержит невалидное регулярное выражение |
| `invalid status: <статус>` | Переданный фильтр `status` не соответствует ни одному из допустимых значений `TaskStatus` |
| `created_from cannot be after created_to` | Диапазон дат в фильтре инвертирован |
| `endpoint id is required` | Обновление endpoint очереди вызвано без указания ID |
| `password must be at least 8 characters` | Пароль при регистрации короче 8 символов |
| `invalid email format` | Адрес электронной почты не соответствует формату `user@domain.tld` |

---

## 4.4 NOT_FOUND

**gRPC-код:** `5 NOT_FOUND`
**HTTP-эквивалент:** `404 Not Found`

Запрошенная сущность не была найдена или не существует в системе.

**Возможные сообщения:**

| Сообщение | Причина |
|-----------|---------|
| `task not found: <детали>` | Задача с указанным UUID не существует в базе данных |
| `job not found: <детали>` | Задание с указанным UUID не существует в базе данных |
| `HTML page file not available for this task` | Задача завершилась, но HTML-снимок не был сохранён в MinIO (объект отсутствует в метаданных задачи) |
| `result file not available for this task` | Задача завершилась, но файл результатов парсинга не был сохранён в MinIO |
| `JSON export not available for this job` | Задание завершилось, но JSON-экспорт ещё не был сформирован или завершился с ошибкой |
| `CSV export not available for this job` | Задание завершилось, но CSV-экспорт ещё не был сформирован или завершился с ошибкой |
| `user not found` | Пользователь с указанным ID не найден при попытке изменить его роль |

---

## 4.5 PERMISSION_DENIED

**gRPC-код:** `7 PERMISSION_DENIED`
**HTTP-эквивалент:** `403 Forbidden`

Вызывающая сторона не имеет привилегий на вызов данной операции. Аутентификация прошла успешно, однако роль пользователя недостаточна для выполнения запрошенного действия.

**Возможные сообщения:**

| Сообщение | Причина |
|-----------|---------|
| `access denied` | Роль пользователя не входит в список разрешённых ролей для данного метода (настраивается в RBAC-правилах, `internal/auth/authorization.go`) |
| `missing role` | Токен прошёл верификацию, но claim `role` отсутствует в JWT |
| `invalid role` | Claim `role` присутствует, но его значение не удалось разобрать как известную роль |
| `role is not allowed` | Попытка присвоить пользователю роль `Administrator` через API (данная роль зарезервирована и не назначается через публичный интерфейс) |

**Примечание:** привилегии проверяются интерцептором `RBACInterceptor`
(`internal/auth/authorization.go`) после успешной верификации JWT. Порядок
перехватчиков: `Log → Validate → ShardKey → JWTAuth → RBAC → Handler`.

---

## 4.6 INTERNAL

**gRPC-код:** `13 INTERNAL`
**HTTP-эквивалент:** `500 Internal Server Error`

Произошла внутренняя ошибка сервера. Формат ошибки приведён в п. 4.1.
Данный код сигнализирует о проблеме на стороне инфраструктуры или
неожиданном состоянии системы.

**Возможные сообщения:**

| Сообщение | Причина |
|-----------|---------|
| `failed to generate presigned URL: <детали>` | Не удалось сформировать временную ссылку для скачивания файла из MinIO (недоступность хранилища, неверные credentials) |
| `failed to encode cursor: <детали>` | Ошибка при сериализации токена пагинации в JSON; свидетельствует о неожиданном типе данных в результате запроса |
| `failed to list users` | Запрос к базе данных при получении списка пользователей завершился ошибкой |
| `failed to list jobs` | Запрос к базе данных при получении списка заданий завершился ошибкой |
| `failed to update user role` | Транзакция обновления роли пользователя завершилась ошибкой базы данных |

Во всех перечисленных случаях полная цепочка ошибки записывается в лог
(см. п. 4.1). Оператору следует обратиться к журналу сервера с уточнением
timestamp и gRPC-метода для диагностики причины.

---

## 4.7 UNAUTHENTICATED

**gRPC-код:** `16 UNAUTHENTICATED`
**HTTP-эквивалент:** `401 Unauthorized`

Клиент запросил ресурс, требующий авторизации, но не предоставил корректные
авторизационные данные, либо процесс аутентификации не был корректно завершён.

**Возможные сообщения:**

| Сообщение | Причина |
|-----------|---------|
| `missing metadata` | gRPC-запрос отправлен без метаданных (отсутствует заголовок `Authorization` целиком) |
| `missing authorization header` | Метаданные присутствуют, но ключ `authorization` в них не указан |
| `invalid authorization header format` | Заголовок присутствует, но не соответствует схеме `Bearer <token>` |
| `missing or invalid authorization header` | Общая ошибка разбора заголовка авторизации |
| `invalid or expired token` | JWT не прошёл верификацию подписи или истёк срок его действия (`exp` claim) |
| `invalid credentials` | Неверная пара email/пароль при вызове `Login` |
| `invalid refresh token` | Refresh-токен не найден в базе данных, уже использован или отозван |

**Примечание:** верификация JWT выполняется интерцептором `JWTAuthInterceptor`
(`internal/auth/middleware.go`). Методы `Login` и `Register` являются
публичными и не требуют токена; все остальные методы защищены.
