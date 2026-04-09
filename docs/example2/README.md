# Example 2

Этот пример показывает комбинированный обход каталога и карточек товаров на `web-scraping.dev`.

## Что лежит в папке

- `request2.json` - конфигурация задания.
- `example-response2.json` - пример результата.
- `items-list.html` - пример HTML страницы списка товаров.
- `item-information.html` - пример HTML карточки товара.

## Что показывает запрос

`request2.json` описывает задание, в котором crawler:

- стартует с `https://www.web-scraping.dev/products`;
- работает в режиме `CRAWL_MODE_PAGINATION_AND_LINKS`;
- обходит и страницы листинга, и product pages;
- ограничен доменом `www.web-scraping.dev`;
- исключает лишние разделы через `deny_url_patterns`;
- на страницах каталога собирает item-поля:
  - `name`, `price`, `url`, `thumbnail`, `short_description`;
- на карточках товара собирает page-level fields:
  - `name`, `description`, `price`, `original_price`, `images`, `variants`,
  - `feature_labels`, `feature_values`,
  - `pack_versions`, `pack_weights`, `pack_dimensions`,
  - `reviews`.

Это пример "полного e-commerce обхода", когда листинг дает краткие карточки, а product page дает детальные характеристики.

## Что генерируется в ответ

`example-response2.json` показывает смешанный результат:

- для страниц каталога:
  - `fields` обычно пустые или `null`;
  - `items` содержит список товаров со страницы;
- для карточек товара:
  - `fields` содержит полную карточку товара;
  - `items` может содержать сопутствующие элементы, найденные по `container_selector` на странице;
- весь экспорт объединяется в один массив `results`;
- `total_tasks` показывает общее число посещенных страниц.

## Что важно для спеки

- В одном задании могут встречаться разные типы страниц и, как следствие, разное наполнение `fields` и `items`.
- Для листинга основной полезный результат находится в `items`.
- Для карточки товара основной полезный результат находится в `fields`.
- Пример показывает, что `reviews` приходит как строка с JSON, если для поля не задан отдельный transform/parsing шаг.
