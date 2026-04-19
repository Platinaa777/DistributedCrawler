# Example 13

Этот пример показывает краулинг публичного GitHub-профиля с аутентификацией через Bearer token.

## Что лежит в папке

- `request13.json` — конфигурация задания с Bearer token auth.
- `response13.json` — пример результата выполнения.

## Что показывает запрос

`request13.json` описывает сценарий, в котором crawler:

- стартует с `https://github.com/Platinaa777?tab=repositories` — вкладка репозиториев публичного профиля;
- передаёт Bearer token в каждом запросе (повышает лимит GitHub с 60 до 5000 req/час);
- работает в режиме `CRAWL_MODE_PAGINATION_ONLY` — ходит по страницам пагинации репозиториев;
- с уровня профиля (page-level `fields`) извлекает `profile_name`, `profile_login`, `bio`;
- из каждой карточки репозитория (`items`) извлекает `name`, `url`, `description`, `language`, `stars`, `is_fork`.

## Как работает Bearer token

```json
"auth": {
  "bearer_token": "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

Fetcher добавляет заголовок `Authorization: Bearer <token>` к каждому HTTP-запросу. Для публичного GitHub это необязательно, но критично при частом краулинге — без токена лимит 60 req/час, с токеном — 5000 req/час.

## Как получить GitHub Personal Access Token

1. **github.com → Settings → Developer settings → Personal access tokens → Tokens (classic)**
2. **Generate new token** → выбрать scope `public_repo` (для публичных данных scope не нужен вовсе, но токен всё равно требуется для снятия rate-limit)
3. Скопировать токен (показывается один раз) и подставить в `bearer_token`

Проверка токена:
```bash
curl -H "Authorization: Bearer ghp_xxx..." "https://api.github.com/user"
```

## Что важно для спеки

- `allowed_url_patterns` жёстко ограничивает обход только страницами вкладки `?tab=repositories` конкретного пользователя — краулер не уйдёт в чужие профили или внутренние страницы репозиториев.
- Пагинация идёт через `a[rel='next']` — стандартный GitHub-паттерн для всех вкладок профиля.
- GitHub рендерит страницы профиля серверно (SSR), поэтому HTML-парсер работает корректно без headless-браузера.
- `is_fork` извлекается из лейбла `Fork` рядом с именем репозитория; если репозиторий не является форком — поле будет `null`.
