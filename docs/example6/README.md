# Example 6

Этот пример показывает минимальную конфигурацию cron-задачи без отдельного приложенного response-файла.

## Что лежит в папке

- `cron-request.json` - конфигурация периодического задания.
- `All products _ Books to Scrape - Sandbox.html` - HTML страницы, под которую настроены селекторы.

## Что показывает запрос

`cron-request.json` демонстрирует:

- `job_type = JOB_TYPE_SCHEDULED`;
- `schedule.cron = "* * * * *"`;
- обход `books.toscrape.com` в режиме `CRAWL_MODE_PAGINATION_ONLY`;
- извлечение page-level поля `page_title`;
- извлечение item-полей `name`, `price`, `availability`, `url`.

По сути это эталонный пример запроса для создания периодической задачи.

## Что генерируется в ответ

Отдельный response-файл в папке не приложен, но ожидаемый результат имеет ту же форму, что и в Example 5:

- верхний уровень: `job_id`, `exported_at`, `results`, `total_tasks`;
- каждая обработанная страница:
  - `task_id`, `url`, `status`;
  - `fields.page_title`;
  - `items` со списком книг.

## Что важно для спеки

- Этот пример полезен именно как reference для payload на создание scheduled job.
- Если нужен пример фактического результата выполнения, его нужно смотреть в `docs/example5/response5.json`.
