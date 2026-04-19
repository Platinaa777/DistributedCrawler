# Example 12

Этот пример показывает краулинг внутренней wiki с HTTP Basic Authentication (логин + пароль).

## Что лежит в папке

- `request12.json` — конфигурация задания с Basic Auth.
- `response12.json` — пример результата выполнения.

## Что показывает запрос

`request12.json` описывает сценарий, в котором crawler:

- стартует с `https://wiki.internal.example.com/docs/` — закрытый корпоративный ресурс;
- передаёт Basic Auth credentials в каждом запросе;
- работает в режиме `CRAWL_MODE_PAGINATION_AND_LINKS` — переходит по ссылкам внутри раздела `/docs/`;
- ограничен паттерном `allowed_url_patterns`, чтобы не уходить за пределы документации;
- исключает страницы редактирования (`/docs/_edit/`);
- с каждой страницы извлекает `page_title`, `last_modified`, `content`;
- в items собирает дочерние страницы (`title`, `url`).

## Как работает Basic Auth

Поля `auth.basic_user` и `auth.basic_password` используются для HTTP Basic Authentication:

```json
"auth": {
  "basic_user": "crawler",
  "basic_password": "s3cr3t"
}
```

Fetcher формирует заголовок `Authorization: Basic <base64(user:password)>` и добавляет его к каждому запросу. В browser-режиме credentials передаются напрямую в URL (`user:pass@host`) при навигации.

## Когда использовать Basic Auth

- Внутренние сервисы и dev-стенды, защищённые nginx `auth_basic`;
- Legacy-системы без поддержки токенов;
- REST API с поддержкой Basic Auth (например, JIRA, Confluence Data Center).

## Что важно для спеки

- Заполняются оба поля `basic_user` и `basic_password`; остальные auth-поля пустые.
- `allowed_url_patterns` здесь важен — без него crawler может уйти в ссылки за пределами документации (например, в систему тикетов или логины).
- `deny_url_patterns` исключает страницы редактирования, чтобы не триггерить CSRF-защиту.
- `max_depth: 2` позволяет зайти на один уровень вложенности: `docs/` → `docs/backend/` → но не глубже.
