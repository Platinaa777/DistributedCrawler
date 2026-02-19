# TODO: сайты для парсинга (диплом)

Обновлено: 2026-02-17

## Цель
Подобрать сайты-магазины, на которых можно показать, что `distributed-crawler` умеет:
- краулить по `seeds` + `allowed_domains` + `max_depth`;
- уважать `robots.txt` (`respect_robots_txt=true`);
- обрабатывать пагинацию и извлечение полей через `ExtractionSpec`;
- делать экспорт JSON/CSV и метрики по задачам.

## Приоритет P0 (рекомендую начать с них)

1. **Books to Scrape** — https://books.toscrape.com/
- Почему: классический учебный магазин, 1000 товаров, статическая пагинация, стабильная разметка.
- Что парсить: `title`, `price`, `availability`, `rating`, `category`, `product_url`, `image_url`.
- Что показать в дипломе: массовый обход каталога + метрики полноты полей + экспорт CSV.

2. **ScrapeMe (WooCommerce shop)** — https://scrapeme.live/shop/
- Почему: реалистичный интернет-магазин (WooCommerce), много карточек товаров (сейчас отображается 755 товаров).
- robots: https://scrapeme.live/robots.txt (разрешено всё, кроме `/wp-admin/`).
- Что парсить: `name`, `price`, `sku` (если есть), `categories/tags`, `stock`, `images`.
- Что показать: работа `pagination` + нормализация цен + дедуп URL.

3. **web-scraping.dev (e-commerce scenarios)** — https://www.web-scraping.dev/
- Почему: легальная тренировочная e-commerce платформа с реалистичными сценариями (pagination, auth, GraphQL, CSRF).
- robots: https://www.web-scraping.dev/robots.txt (общий allow, частичные ограничения для отдельных ботов).
- Что парсить: каталог `/products`, карточки `/product/{id}`, отзывы `/reviews`.
- Что показать: сравнение HTML extraction vs API-like endpoints, отказоустойчивость воркеров.

## Приоритет P1 (добавить для «вау-эффекта»)

4. **ScrapingTest e-commerce playground** — https://scrapingtest.com/
- Почему: сценарии магазина с `pagination`, `load more`, `infinite scroll`, anti-bot challenge.
- Что парсить: имя товара, бренд, цена, рейтинг, количество отзывов.
- Что показать: где хватает HTTP fetcher, а где нужен browser fetcher.

5. **OpenCart Demo Store** — https://demo.opencart.com/
- Подтверждение демо: https://www.opencart.com/index.php?route=demonstration%2Fdemonstration
- robots: https://demo.opencart.com/robots.txt (есть ограничения на часть query-параметров).
- Что парсить: каталог, карточки, breadcrumbs, характеристики.
- Что показать: корректная работа `allowed_domains`, `deny_url_patterns`, `respect_robots_txt`.

## Приоритет P2 (для production-части диплома)

6. **eBay Developers API** — https://developer.ebay.com/api-documentation
- Почему: показать промышленный канал сбора данных легально через официальный API.
- Что показать: отдельный ingestion-пайплайн + сравнение с HTML-скрейпингом по качеству данных.

7. **Etsy Open API v3** — https://developer.etsy.com/documentation/
- Важно: Etsy прямо запрещает обход API скрейпингом страниц для коммерческого доступа — использовать API.
- Что показать: юридически корректный сбор + OAuth + единая схема данных в экспорте.

8. **Walmart Developer Portal** — https://developer.walmart.com/documentation/item-object-v4-0-2/
- Почему: enterprise-уровень витрины и каталогов через официальные интерфейсы.
- Что показать: масштабируемость, rate-limit policy, повторные попытки и мониторинг.

## Сайты, которые лучше не брать в диплом как основной источник

- `webscraper.io/test-sites/e-commerce` — в robots запрещён раздел e-commerce: https://webscraper.io/robots.txt
- Крупные коммерческие магазины без явного разрешения (anti-bot, риск блокировки, риск нарушений ToS).

## План демонстрации (минимум для сильной защиты)

1. Запустить 3 job’а: `Books to Scrape`, `ScrapeMe`, `web-scraping.dev`.
2. Для каждого показать:
- конфиг `seeds/scopes/extraction_spec/pagination`;
- граф задач (сколько enqueued/completed/failed);
- экспорт `report.csv` и `report.json`;
- метрики качества данных (заполненность полей, дубликаты, ошибки extraction).
3. Отдельно показать один API-источник (например eBay API) как «production-ready и legal-first» трек.

## Источники

- https://toscrape.com/
- https://books.toscrape.com/catalogue/page-2.html
- https://scrapeme.live/shop/
- https://scrapeme.live/robots.txt
- https://www.web-scraping.dev/
- https://www.web-scraping.dev/robots.txt
- https://scrapingtest.com/
- https://scrapingtest.com/ecommerce/load-more
- https://www.opencart.com/index.php?route=demonstration%2Fdemonstration
- https://demo.opencart.com/robots.txt
- https://developer.ebay.com/api-documentation
- https://developer.etsy.com/documentation/
- https://developer.walmart.com/documentation/item-object-v4-0-2/
- https://webscraper.io/robots.txt
