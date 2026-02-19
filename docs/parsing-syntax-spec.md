# Документация по DSL экстракции данных (ExtractionSpec)

## Содержание

1. [Обзор ExtractionSpec](#1-обзор-extractionspec)
2. [Fields: структура и жизненный цикл](#2-fields-структура-и-жизненный-цикл)
3. [ExtractorSpec: извлечение данных](#3-extractorspec-извлечение-данных)
4. [CSS-селекторы: полный справочник](#4-css-селекторы-полный-справочник)
5. [Transforms: преобразования данных](#5-transforms-преобразования-данных)
6. [ValueType: типы данных](#6-valuetype-типы-данных)
7. [Полные примеры конфигураций](#7-полные-примеры-конфигураций)
8. [Best Practices и рекомендации](#8-best-practices-и-рекомендации)

---

## 1. Обзор ExtractionSpec

`ExtractionSpec` — это конфигурация для извлечения структурированных данных из HTML-страниц. Спецификация определяет:

- **Fields** — какие данные извлекать со страницы (заголовки, цены, ссылки и т.д.)

### Структура ExtractionSpec

```go
type ExtractionSpec struct {
    Fields  []FieldSpec   // Список полей для извлечения
}
```

### Жизненный цикл обработки

```
HTML → Parse (goquery) → Extract Fields → Apply Transforms → Type Conversion → Result JSON
```

1. **Парсинг HTML** — HTML преобразуется в DOM-дерево через goquery
2. **Извлечение полей** — для каждого `FieldSpec` применяется CSS-селектор и извлекается значение
3. **Применение трансформаций** — последовательно применяются операции из `Transforms`
4. **Конвертация типов** — значение приводится к указанному `ValueType`
5. **Результат** — структура `{ fields: {...} }` сохраняется в JSON

---

## 2. Fields: структура и жизненный цикл

### Структура FieldSpec

```go
type FieldSpec struct {
    Name       string          // Имя поля в выходном JSON (например, "title")
    Label      string          // Человекочитаемое имя для UI (например, "Заголовок страницы")
    Type       ValueType       // Тип данных: string, int, float, bool, url, json
    Required   bool            // Обязательное ли поле (влияет на логирование)
    Extractor  ExtractorSpec   // Правила извлечения из HTML
    Transforms []TransformSpec // Последовательность преобразований
}
```

### Семантика полей

- **Name** — ключ в результирующем JSON. Используйте snake_case (например, `product_price`)
- **Label** — описание для пользователя (например, "Цена товара в рублях")
- **Type** — ожидаемый тип данных после всех преобразований
- **Required** — если `true` и извлечение не удалось, будет записан WARNING в лог, но поле будет `null` в результате
- **Extractor** — определяет, откуда и как извлечь данные
- **Transforms** — применяются последовательно к извлеченному значению

### Жизненный цикл обработки одного поля

```
1. applyExtractor(ExtractorSpec)
   ↓ rawValue (string или []string)

2. applyTransform(Transforms[0])
   ↓
3. applyTransform(Transforms[1])
   ↓
   ... (для каждого Transform)
   ↓ transformedValue

4. convertToType(ValueType)
   ↓ finalValue (приведенное к типу)

5. fields[Name] = finalValue
```

**Важно:**
- Если `rawValue == nil` или пустая строка, поле будет `null` (трансформации не применяются)
- Если `required=true` и произошла ошибка при извлечении, в лог пишется WARNING, но выполнение продолжается
- Ошибки конвертации типов возвращают ошибку и могут прервать обработку поля

### Пример FieldSpec

```json
{
  "name": "product_title",
  "label": "Название товара",
  "type": "string",
  "required": true,
  "extractor": {
    "selector": "h1.product-name",
    "attribute": "text",
    "multiple": false,
    "index": null
  },
  "transforms": [
    { "op": "trim", "arg": null },
    { "op": "collapse_ws", "arg": null }
  ]
}
```

---

## 3. ExtractorSpec: извлечение данных

### Структура ExtractorSpec

```go
type ExtractorSpec struct {
    Selector  string // CSS-селектор (например, "div.price", "meta[property='og:title']")
    Attribute string // Атрибут или псевдо-атрибут ("text", "html", "href", "src", "content" и т.д.)
    Multiple  bool   // Извлечь все совпадения (true) или только первое (false)
    Index     *int   // Если Multiple=true, взять элемент по индексу (0-based; -1 = последний)
}
```

### Параметр `Selector`

CSS-селектор для поиска элементов в DOM. Использует синтаксис goquery (jQuery-подобный).

**Примеры:**
```css
div.product           /* Все <div> с классом "product" */
#main-content         /* Элемент с id="main-content" */
a[href^="https://"]   /* Все ссылки, начинающиеся с https:// */
meta[property='og:image'] /* Meta-тег с property="og:image" */
```

См. подробный справочник в разделе [CSS-селекторы](#4-css-селекторы-полный-справочник).

### Параметр `Attribute`

Определяет, какую часть элемента извлечь.

#### Псевдо-атрибуты (специальные значения)

| Attribute           | Что извлекается                               |
|---------------------|-----------------------------------------------|
| `""` (пустая строка)| Текстовое содержимое элемента (`.Text()`)    |
| `"text"`            | Текстовое содержимое элемента (`.Text()`)    |
| `"innerText"`       | Текстовое содержимое элемента (`.Text()`)    |
| `"html"`            | HTML-содержимое элемента (`.Html()`)          |
| `"innerHTML"`       | HTML-содержимое элемента (`.Html()`)          |

#### Реальные HTML-атрибуты

Если указано любое другое значение (например, `"href"`, `"src"`, `"content"`), система извлекает соответствующий HTML-атрибут.

**Специальная обработка для `href` и `src`:**
- Если атрибут `href` или `src`, относительные URL автоматически резолвятся в абсолютные через `baseURL`
- `baseURL` определяется из `task.FinalURL` (если был редирект) или `task.URL`

**Примеры:**

```json
// Извлечь текст из <h1>
{ "selector": "h1", "attribute": "text" }

// Извлечь href из <a>
{ "selector": "a.download-link", "attribute": "href" }

// Извлечь content из <meta>
{ "selector": "meta[property='og:title']", "attribute": "content" }

// Извлечь src из <img>
{ "selector": "img.product-image", "attribute": "src" }

// Извлечь HTML внутри <div>
{ "selector": "div.description", "attribute": "html" }
```

### Параметр `Multiple`

**`multiple: false`** (по умолчанию)
- Возвращается **одна строка** — значение **первого** найденного элемента
- Если элементов нет — возвращается ошибка

**`multiple: true`**
- Возвращается **массив строк** — значения **всех** найденных элементов
- Если элементов нет — возвращается ошибка
- Пустые строки фильтруются (не добавляются в массив)

**Примеры:**

<details>
<summary><b>Пример HTML</b></summary>

```html
<ul class="tags">
  <li>Python</li>
  <li>Go</li>
  <li>JavaScript</li>
  <li></li> <!-- пустой элемент -->
</ul>
```
</details>

```json
// multiple: false → вернет "Python" (первый элемент)
{
  "selector": "ul.tags li",
  "attribute": "text",
  "multiple": false
}

// multiple: true → вернет ["Python", "Go", "JavaScript"]
{
  "selector": "ul.tags li",
  "attribute": "text",
  "multiple": true
}
```

### Параметр `Index`

Используется **только** если `multiple: true`. Позволяет выбрать один элемент из массива.

**Правила:**
- `index >= 0` — индекс с начала массива (0 = первый элемент)
- `index < 0` — индекс с конца массива (-1 = последний элемент, -2 = предпоследний)
- Если индекс выходит за границы массива — возвращается ошибка

**Примеры:**

<details>
<summary><b>Пример HTML</b></summary>

```html
<ul class="breadcrumb">
  <li>Главная</li>
  <li>Категория</li>
  <li>Подкатегория</li>
  <li>Товар</li>
</ul>
```
</details>

```json
// Первый элемент (index: 0) → "Главная"
{
  "selector": "ul.breadcrumb li",
  "attribute": "text",
  "multiple": true,
  "index": 0
}

// Второй элемент (index: 1) → "Категория"
{
  "selector": "ul.breadcrumb li",
  "attribute": "text",
  "multiple": true,
  "index": 1
}

// Последний элемент (index: -1) → "Товар"
{
  "selector": "ul.breadcrumb li",
  "attribute": "text",
  "multiple": true,
  "index": -1
}

// Предпоследний элемент (index: -2) → "Подкатегория"
{
  "selector": "ul.breadcrumb li",
  "attribute": "text",
  "multiple": true,
  "index": -2
}
```

**Важно:** Если `multiple: false`, параметр `index` игнорируется.

---

## 4. CSS-селекторы: полный справочник

CSS-селекторы используются для поиска элементов в HTML. Библиотека goquery поддерживает большинство стандартных селекторов CSS3.

### 4.1 Базовые селекторы

| Селектор | Описание | HTML пример | Что выберет |
|----------|----------|-------------|-------------|
| `tag` | По имени тега | `<p>Text</p>` | Все `<p>` |
| `.class` | По классу | `<div class="box">` | Все элементы с классом `box` |
| `#id` | По ID | `<div id="main">` | Элемент с `id="main"` |
| `*` | Универсальный | Любой элемент | Все элементы |

**Примеры:**

```html
<div class="container">
  <h1 id="title">Заголовок</h1>
  <p class="text">Параграф 1</p>
  <p class="text highlight">Параграф 2</p>
</div>
```

| Селектор | Результат |
|----------|-----------|
| `h1` | `<h1 id="title">` |
| `.text` | Оба `<p>` |
| `#title` | `<h1 id="title">` |
| `.highlight` | Второй `<p>` |

### 4.2 Комбинаторы (отношения между элементами)

| Селектор | Имя | Описание |
|----------|-----|----------|
| `A B` | Потомок (descendant) | `B` внутри `A` на любом уровне вложенности |
| `A > B` | Прямой потомок (child) | `B` непосредственно внутри `A` |
| `A + B` | Соседний элемент (adjacent) | `B` сразу после `A` (на том же уровне) |
| `A ~ B` | Все следующие (siblings) | Все `B` после `A` (на том же уровне) |

**Примеры:**

```html
<div class="parent">
  <span>Span 1</span>
  <div class="child">
    <span>Span 2</span>
  </div>
  <span>Span 3</span>
  <span>Span 4</span>
</div>
```

| Селектор | Результат |
|----------|-----------|
| `div span` | Все 4 `<span>` (потомки на любом уровне) |
| `div > span` | `Span 1`, `Span 3`, `Span 4` (прямые дети `div`) |
| `.child + span` | `Span 3` (первый `<span>` после `.child`) |
| `.child ~ span` | `Span 3`, `Span 4` (все `<span>` после `.child`) |

**Важно:** Пробел в селекторе `div span` означает "любой `<span>` внутри `<div>`", а не "элемент с классом `div span`"!

### 4.3 Атрибутные селекторы

| Селектор | Описание |
|----------|----------|
| `[attr]` | Элементы с атрибутом `attr` |
| `[attr="value"]` | `attr` точно равен `"value"` |
| `[attr^="value"]` | `attr` **начинается** с `"value"` |
| `[attr$="value"]` | `attr` **заканчивается** на `"value"` |
| `[attr*="value"]` | `attr` **содержит** `"value"` |
| `[attr~="value"]` | `attr` содержит слово `"value"` (через пробел) |
| `[attr\|="value"]` | `attr` начинается с `"value"` или `"value-"` |

**Примеры:**

```html
<a href="https://example.com">Link 1</a>
<a href="http://example.com">Link 2</a>
<a href="/page">Link 3</a>
<input type="text" name="username">
<input type="password" name="pass">
<meta property="og:title" content="Title">
<meta property="og:image" content="Image">
```

| Селектор | Результат |
|----------|-----------|
| `a[href]` | Все ссылки с атрибутом `href` |
| `a[href^="https://"]` | Link 1 (начинается с `https://`) |
| `a[href^="http"]` | Link 1, Link 2 (начинаются с `http`) |
| `a[href$=".pdf"]` | Ссылки, заканчивающиеся на `.pdf` |
| `a[href*="example"]` | Link 1, Link 2 (содержат `example`) |
| `input[type="text"]` | `<input type="text">` |
| `meta[property='og:title']` | `<meta property="og:title">` |

**Примечание:** Для атрибутов с двоеточием (например, `og:title`) используйте одинарные кавычки внутри двойных: `"meta[property='og:title']"`.

### 4.4 Групповая выборка

| Селектор | Описание |
|----------|----------|
| `A, B, C` | Все элементы, соответствующие `A` **или** `B` **или** `C` |

**Примеры:**

```html
<h1>Main Title</h1>
<h2>Subtitle 1</h2>
<h2>Subtitle 2</h2>
<p>Paragraph</p>
```

| Селектор | Результат |
|----------|-----------|
| `h1, h2` | `<h1>`, оба `<h2>` |
| `h1, p` | `<h1>`, `<p>` |

### 4.5 Псевдоклассы (ограниченная поддержка)

**Поддерживаются:**
- `:first-child` — первый дочерний элемент
- `:last-child` — последний дочерний элемент
- `:nth-child(n)` — n-й дочерний элемент
- `:first-of-type` — первый элемент данного типа
- `:last-of-type` — последний элемент данного типа

**Примеры:**

```html
<ul>
  <li>Item 1</li>
  <li>Item 2</li>
  <li>Item 3</li>
</ul>
```

| Селектор | Результат |
|----------|-----------|
| `li:first-child` | `Item 1` |
| `li:last-child` | `Item 3` |
| `li:nth-child(2)` | `Item 2` |

**Важно:** Поддержка псевдоклассов зависит от движка goquery/cascadia. Не все CSS3 псевдоклассы (например, `:hover`, `:focus`) доступны. Если селектор не работает, используйте `multiple: true` + `index`.

### 4.6 Практические примеры

<details>
<summary><b>Пример 1: Извлечь все ссылки на PDF-файлы</b></summary>

```html
<div class="documents">
  <a href="/file1.pdf">Document 1</a>
  <a href="/file2.doc">Document 2</a>
  <a href="/file3.pdf">Document 3</a>
</div>
```

**Селектор:** `a[href$=".pdf"]`
**Результат:** Link на `file1.pdf` и `file3.pdf`

```json
{
  "selector": "a[href$='.pdf']",
  "attribute": "href",
  "multiple": true
}
```
</details>

<details>
<summary><b>Пример 2: Извлечь OpenGraph URL из meta-тега</b></summary>

```html
<head>
  <meta property="og:url" content="https://example.com/page">
  <meta property="og:title" content="Page Title">
</head>
```

**Селектор:** `meta[property='og:url']`
**Attribute:** `content`
**Результат:** `"https://example.com/page"`

```json
{
  "selector": "meta[property='og:url']",
  "attribute": "content",
  "multiple": false
}
```
</details>

<details>
<summary><b>Пример 3: Извлечь текст из вложенных элементов</b></summary>

```html
<div class="article">
  <div class="content">
    <p>Paragraph 1</p>
    <p>Paragraph 2</p>
  </div>
</div>
```

**Селектор:** `div.article div.content p`
**Результат (multiple: false):** `"Paragraph 1"`
**Результат (multiple: true):** `["Paragraph 1", "Paragraph 2"]`

```json
{
  "selector": "div.article div.content p",
  "attribute": "text",
  "multiple": true
}
```
</details>

---

## 5. Transforms: преобразования данных

Трансформации применяются **последовательно** к извлеченному значению. Каждая операция принимает результат предыдущей.

### 5.1 Список всех трансформаций

| Операция | Тип входа | Тип выхода | Описание |
|----------|-----------|------------|----------|
| `trim` | `string` | `string` | Удаляет пробелы в начале и конце |
| `lower` | `string` | `string` | Преобразует в нижний регистр |
| `upper` | `string` | `string` | Преобразует в верхний регистр |
| `collapse_ws` | `string` | `string` | Заменяет множественные пробелы на один |
| `html_to_text` | `string` | `string` | Удаляет HTML-теги |
| `normalize_url` | `string` | `string` | Нормализует URL (парсит и форматирует) |
| `unique` | `[]string` | `[]string` | Удаляет дубликаты из массива |
| `limit` | `[]string` | `[]string` | Ограничивает длину массива (arg: число) |
| `parse_int` | `string` | `int64` | Парсит строку в целое число |
| `parse_float` | `string` | `float64` | Парсит строку в дробное число |
| `parse_price` | `string` | `float64` | Извлекает число из строки (например, "$19.99" → 19.99) |
| `sha256` | `string` | `string` | Вычисляет SHA-256 хеш |

### 5.2 Детальное описание операций

#### `trim`

Удаляет пробелы, табуляцию, переводы строк в начале и конце строки.

**До:**
```
"  Hello World  \n"
```

**После:**
```
"Hello World"
```

**Конфигурация:**
```json
{ "op": "trim", "arg": null }
```

---

#### `lower`

Преобразует все символы в нижний регистр.

**До:**
```
"Hello World"
```

**После:**
```
"hello world"
```

**Конфигурация:**
```json
{ "op": "lower", "arg": null }
```

---

#### `upper`

Преобразует все символы в верхний регистр.

**До:**
```
"Hello World"
```

**После:**
```
"HELLO WORLD"
```

**Конфигурация:**
```json
{ "op": "upper", "arg": null }
```

---

#### `collapse_ws`

Заменяет множественные пробелы (включая табы и переводы строк) на один пробел, а также убирает пробелы в начале/конце.

**До:**
```
"Hello    World\n\n  Test"
```

**После:**
```
"Hello World Test"
```

**Конфигурация:**
```json
{ "op": "collapse_ws", "arg": null }
```

**Полезно для:** Очистки текста, скопированного из HTML, где могут быть лишние пробелы.

---

#### `html_to_text`

Удаляет все HTML-теги из строки (простая очистка через регулярное выражение).

**До:**
```
"<p>Hello <b>World</b></p>"
```

**После:**
```
"Hello World"
```

**Конфигурация:**
```json
{ "op": "html_to_text", "arg": null }
```

**Важно:** Это упрощенная очистка. Для более точной работы с HTML используйте `attribute: "text"` вместо `attribute: "html"`.

---

#### `normalize_url`

Парсит URL и возвращает его нормализованную форму.

**До:**
```
"https://example.com/page?query=test&other=value#fragment"
```

**После:**
```
"https://example.com/page?query=test&other=value"
```

**Конфигурация:**
```json
{ "op": "normalize_url", "arg": null }
```

**Примечание:** Фрагмент (часть после `#`) удаляется автоматически при извлечении ссылок в `extractElementValue`, поэтому эта трансформация в основном полезна для валидации URL.

---

#### `unique`

Удаляет дубликаты из массива строк (сохраняет порядок первого вхождения).

**До:**
```json
["apple", "banana", "apple", "cherry", "banana"]
```

**После:**
```json
["apple", "banana", "cherry"]
```

**Конфигурация:**
```json
{ "op": "unique", "arg": null }
```

**Применяется только к:** `[]string`

---

#### `limit`

Ограничивает длину массива указанным числом (обрезает массив).

**До:**
```json
["a", "b", "c", "d", "e"]
```

**После (с arg: 3):**
```json
["a", "b", "c"]
```

**Конфигурация:**
```json
{ "op": "limit", "arg": 3 }
```

**Применяется только к:** `[]string`

**Типы `arg`:** Поддерживаются `int`, `int64`, `float64`, `json.Number`, `string` (будет преобразовано в число).

**Пример использования:**
```json
{
  "extractor": {
    "selector": "ul.tags li",
    "attribute": "text",
    "multiple": true
  },
  "transforms": [
    { "op": "limit", "arg": 5 }
  ]
}
```

---

#### `parse_int`

Парсит строку в целое число (`int64`).

**До:**
```
"12345"
```

**После:**
```
12345
```

**Конфигурация:**
```json
{ "op": "parse_int", "arg": null }
```

**Применяется к:** `string`
**Ошибки:** Если строка не является числом, возвращается исходное значение.

---

#### `parse_float`

Парсит строку в дробное число (`float64`).

**До:**
```
"123.45"
```

**После:**
```
123.45
```

**Конфигурация:**
```json
{ "op": "parse_float", "arg": null }
```

**Применяется к:** `string`

---

#### `parse_price`

Извлекает числовое значение из строки с ценой (удаляет валюты, пробелы и т.д.).

**До:**
```
"$19.99"
"1 234,56 ₽"
"Price: 99.00 USD"
```

**После:**
```
19.99
1234.56
99.00
```

**Конфигурация:**
```json
{ "op": "parse_price", "arg": null }
```

**Применяется к:** `string`

**Алгоритм:** Извлекает первое найденное число (включая дробную часть) с помощью регулярного выражения `\d+\.?\d*`.

**Примеры:**
- `"$19.99"` → `19.99`
- `"1,234.56"` → `1` (внимание: запятая не поддерживается как разделитель!)
- `"Price: 99"` → `99.0`

**Важно:** Для европейских форматов (например, `1 234,56`) может потребоваться дополнительная обработка строки перед `parse_price`.

---

#### `sha256`

Вычисляет SHA-256 хеш строки и возвращает его в шестнадцатеричном формате.

**До:**
```
"hello world"
```

**После:**
```
"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
```

**Конфигурация:**
```json
{ "op": "sha256", "arg": null }
```

**Применяется к:** `string`

**Полезно для:** Генерации уникальных идентификаторов, дедупликации контента.

---

### 5.3 Цепочки трансформаций

Трансформации применяются **последовательно**. Результат одной операции становится входом для следующей.

**Пример 1: Очистка текста**

```json
{
  "name": "clean_title",
  "extractor": { "selector": "h1", "attribute": "text" },
  "transforms": [
    { "op": "trim" },
    { "op": "collapse_ws" },
    { "op": "lower" }
  ]
}
```

**Шаги:**
1. Извлечение: `"  Hello   World  \n"`
2. `trim`: `"Hello   World"`
3. `collapse_ws`: `"Hello World"`
4. `lower`: `"hello world"`

---

**Пример 2: Извлечение первых 5 уникальных тегов**

```json
{
  "name": "top_tags",
  "extractor": {
    "selector": "ul.tags li",
    "attribute": "text",
    "multiple": true
  },
  "transforms": [
    { "op": "unique" },
    { "op": "limit", "arg": 5 }
  ]
}
```

**Шаги:**
1. Извлечение: `["Python", "Go", "Python", "JavaScript", "Rust", "Go"]`
2. `unique`: `["Python", "Go", "JavaScript", "Rust"]`
3. `limit`: `["Python", "Go", "JavaScript", "Rust"]` (меньше 5, не обрезается)

---

**Пример 3: Парсинг цены и преобразование в float**

```json
{
  "name": "product_price",
  "type": "float",
  "extractor": { "selector": "span.price", "attribute": "text" },
  "transforms": [
    { "op": "parse_price" }
  ]
}
```

**Шаги:**
1. Извлечение: `"$19.99"`
2. `parse_price`: `19.99` (float64)
3. Конвертация в `type: "float"`: `19.99`

---

## 6. ValueType: типы данных

После применения всех трансформаций значение конвертируется в указанный тип.

### 6.1 Поддерживаемые типы

| Тип | Описание | Примеры значений |
|-----|----------|------------------|
| `string` | Строка | `"hello"`, `"123"` |
| `int` | Целое число | `123`, `-456` |
| `float` | Дробное число | `123.45`, `-0.99` |
| `bool` | Логическое | `true`, `false` |
| `url` | URL (валидируется) | `"https://example.com"` |
| `json` | Любой JSON | `{"key": "value"}`, `[1, 2, 3]` |

### 6.2 Правила конвертации

#### `string`

Любое значение конвертируется в строку через `fmt.Sprintf("%v", value)`.

**Примеры:**
- `123` → `"123"`
- `true` → `"true"`
- `[]string{"a", "b"}` → `"[a b]"`

**Ошибки:** Не возникают.

---

#### `int`

Конвертирует значение в `int64`.

**Поддерживаемые входные типы:**
- `int`, `int64` → возвращается как есть
- `float64` → обрезается дробная часть
- `json.Number` → парсится в int64
- `string` → парсится через `strconv.ParseInt`

**Примеры:**
- `"123"` → `123`
- `123.99` → `123`
- `"abc"` → **ошибка**

**Ошибки:** Если строка не является числом.

---

#### `float`

Конвертирует значение в `float64`.

**Поддерживаемые входные типы:**
- `float64`, `float32` → возвращается как есть
- `int`, `int64` → конвертируется в float
- `string` → парсится через `strconv.ParseFloat`

**Примеры:**
- `"123.45"` → `123.45`
- `123` → `123.0`
- `"abc"` → **ошибка**

**Ошибки:** Если строка не является числом.

---

#### `bool`

Конвертирует значение в `bool`.

**Поддерживаемые входные типы:**
- `bool` → возвращается как есть
- `string` → парсится через `strconv.ParseBool`

**Поддерживаемые строковые значения:**
- `"true"`, `"1"`, `"t"`, `"T"`, `"TRUE"`, `"True"` → `true`
- `"false"`, `"0"`, `"f"`, `"F"`, `"FALSE"`, `"False"` → `false`

**Примеры:**
- `"true"` → `true`
- `"1"` → `true`
- `"yes"` → **ошибка**

**Ошибки:** Если строка не соответствует поддерживаемым значениям.

---

#### `url`

Проверяет, что строка является валидным URL.

**Валидация:** Парсится через `url.Parse`. Если парсинг не удался — возвращается ошибка.

**Примеры:**
- `"https://example.com"` → `"https://example.com"`
- `"/relative/path"` → `"/relative/path"` (валидный относительный URL)
- `"not a url"` → **ошибка**

**Возвращается:** Строка (но валидированная как URL).

---

#### `json`

Значение остается в исходном формате (нативный Go тип).

**Примеры:**
- `"string"` → `"string"`
- `123` → `123`
- `[]string{"a", "b"}` → `[]string{"a", "b"}`
- `map[string]any{"key": "value"}` → `map[string]any{"key": "value"}`

**Полезно для:** Хранения массивов, объектов без преобразования в строку.

---

### 6.3 Обработка ошибок конвертации

Если конвертация типа не удалась (например, `"abc"` → `int`), возвращается ошибка, и **поле не добавляется в результат** (или устанавливается в `null`).

**Пример:**

```json
{
  "name": "invalid_number",
  "type": "int",
  "extractor": { "selector": "span.text", "attribute": "text" }
}
```

Если `<span class="text">` содержит `"abc"`, поле `invalid_number` будет `null` в результате.

---

## 7. Полные примеры конфигураций

### Пример 1: Простое извлечение заголовка и описания

**HTML:**
```html
<html>
<head>
  <meta property="og:title" content="Product Page">
  <meta property="og:description" content="Best product ever">
</head>
<body>
  <h1>Product Title</h1>
</body>
</html>
```

**Конфигурация:**
```json
{
  "extraction_spec": {
    "fields": [
      {
        "name": "og_title",
        "label": "OpenGraph Title",
        "type": "string",
        "required": true,
        "extractor": {
          "selector": "meta[property='og:title']",
          "attribute": "content",
          "multiple": false
        },
        "transforms": [
          { "op": "trim" }
        ]
      },
      {
        "name": "og_description",
        "label": "OpenGraph Description",
        "type": "string",
        "required": false,
        "extractor": {
          "selector": "meta[property='og:description']",
          "attribute": "content",
          "multiple": false
        },
        "transforms": []
      }
    ]
  }
}
```

**Результат:**
```json
{
  "fields": {
    "og_title": "Product Page",
    "og_description": "Best product ever"
  }
}
```

---

### Пример 2: Извлечение цены и преобразование в float

**HTML:**
```html
<div class="product">
  <span class="price">$19.99</span>
</div>
```

**Конфигурация:**
```json
{
  "extraction_spec": {
    "fields": [
      {
        "name": "product_price",
        "label": "Product Price",
        "type": "float",
        "required": true,
        "extractor": {
          "selector": "span.price",
          "attribute": "text",
          "multiple": false
        },
        "transforms": [
          { "op": "parse_price" }
        ]
      }
    ]
  }
}
```

**Результат:**
```json
{
  "fields": {
    "product_price": 19.99
  }
}
```

---

### Пример 3: Извлечение списка тегов (multiple + unique + limit)

**HTML:**
```html
<ul class="tags">
  <li>Python</li>
  <li>Go</li>
  <li>Python</li>
  <li>JavaScript</li>
  <li>Rust</li>
  <li>Go</li>
</ul>
```

**Конфигурация:**
```json
{
  "extraction_spec": {
    "fields": [
      {
        "name": "tags",
        "label": "Article Tags",
        "type": "json",
        "required": false,
        "extractor": {
          "selector": "ul.tags li",
          "attribute": "text",
          "multiple": true
        },
        "transforms": [
          { "op": "unique" },
          { "op": "limit", "arg": 3 }
        ]
      }
    ]
  }
}
```

**Результат:**
```json
{
  "fields": {
    "tags": ["Python", "Go", "JavaScript"]
  }
}
```

---

### Пример 4: Извлечение последнего элемента breadcrumb (index: -1)

**HTML:**
```html
<ul class="breadcrumb">
  <li>Home</li>
  <li>Category</li>
  <li>Subcategory</li>
  <li>Product</li>
</ul>
```

**Конфигурация:**
```json
{
  "extraction_spec": {
    "fields": [
      {
        "name": "current_page",
        "label": "Current Page Name",
        "type": "string",
        "required": true,
        "extractor": {
          "selector": "ul.breadcrumb li",
          "attribute": "text",
          "multiple": true,
          "index": -1
        },
        "transforms": [
          { "op": "trim" }
        ]
      }
    ]
  }
}
```

**Результат:**
```json
{
  "fields": {
    "current_page": "Product"
  }
}
```

---

### Пример 5: Извлечение OpenGraph URL с преобразованием в верхний регистр

**HTML:**
```html
<head>
  <meta property="og:url" content="https://example.com/page">
</head>
```

**Конфигурация:**
```json
{
  "extraction_spec": {
    "fields": [
      {
        "name": "og_url",
        "type": "string",
        "required": true,
        "label": "OpenGraph URL",
        "extractor": {
          "selector": "meta[property='og:url']",
          "attribute": "content",
          "multiple": false,
          "index": 0
        },
        "transforms": [
          { "op": "upper", "arg": "" }
        ]
      }
    ]
  }
}
```

**Построчное объяснение:**
1. **`name: "og_url"`** — ключ в результирующем JSON
2. **`type: "string"`** — ожидаем строку
3. **`required: true`** — если извлечение не удастся, запишем WARNING в лог
4. **`label: "OpenGraph URL"`** — человекочитаемое имя для UI
5. **`selector: "meta[property='og:url']"`** — находим meta-тег с атрибутом `property="og:url"`
6. **`attribute: "content"`** — извлекаем значение атрибута `content`
7. **`multiple: false`** — берем только первый элемент (хотя обычно такой meta-тег один)
8. **`index: 0`** — игнорируется при `multiple: false`
9. **`transforms: [{ "op": "upper" }]`** — преобразуем URL в верхний регистр

**Результат:**
```json
{
  "fields": {
    "og_url": "HTTPS://EXAMPLE.COM/PAGE"
  }
}
```

**Примечание:** Преобразование URL в верхний регистр обычно не имеет практического смысла, но иллюстрирует работу трансформаций.

---

### Пример 6: Извлечение всех внешних ссылок

**HTML:**
```html
<div>
  <a href="https://example.com">Link 1</a>
  <a href="/page">Link 2</a>
  <a href="http://another.com">Link 3</a>
</div>
```

**Конфигурация:**
```json
{
  "extraction_spec": {
    "fields": [
      {
        "name": "external_links",
        "label": "External Links",
        "type": "json",
        "required": false,
        "extractor": {
          "selector": "a[href^='http']",
          "attribute": "href",
          "multiple": true
        },
        "transforms": []
      }
    ]
  }
}
```

**Результат:**
```json
{
  "fields": {
    "external_links": [
      "https://example.com",
      "http://another.com"
    ]
  }
}
```

---

### Пример 7: Извлечение автора и описания

**HTML:**
```html
<article>
  <span class="author">John Doe</span>
  <p class="description">This is a long description with many words for testing purposes.</p>
</article>
```

**Конфигурация:**
```json
{
  "extraction_spec": {
    "fields": [
      {
        "name": "author",
        "label": "Author Name",
        "type": "string",
        "required": false,
        "extractor": {
          "selector": "span.author",
          "attribute": "text",
          "multiple": false
        },
        "transforms": [
          { "op": "trim" }
        ]
      },
      {
        "name": "description",
        "label": "Article Description",
        "type": "string",
        "required": false,
        "extractor": {
          "selector": "p.description",
          "attribute": "text",
          "multiple": false
        },
        "transforms": [
          { "op": "trim" },
          { "op": "collapse_ws" }
        ]
      }
    ]
  }
}
```

**Результат:**
```json
{
  "fields": {
    "author": "John Doe",
    "description": "This is a long description with many words for testing purposes."
  }
}
```

---

### Пример 8: Извлечение изображений

**HTML:**
```html
<body>
  <img src="/local/image1.jpg">
  <img src="https://cdn.example.com/image2.jpg">
  <a href="https://example.com">Link 1</a>
  <a href="/page">Link 2</a>
</body>
```

**Конфигурация:**
```json
{
  "extraction_spec": {
    "fields": [
      {
        "name": "images",
        "label": "All Images",
        "type": "json",
        "required": false,
        "extractor": {
          "selector": "img",
          "attribute": "src",
          "multiple": true
        },
        "transforms": []
      }
    ]
  }
}
```

**Результат (если `baseURL = "https://example.com"`):**
```json
{
  "fields": {
    "images": [
      "https://example.com/local/image1.jpg",
      "https://cdn.example.com/image2.jpg"
    ]
  }
}
```

**Примечание:** Относительные URL в `src` автоматически резолвятся в абсолютные.

---

### Пример 9: Извлечение HTML-контента и очистка

**HTML:**
```html
<div class="content">
  <p>First <b>paragraph</b></p>
  <p>Second paragraph</p>
</div>
```

**Конфигурация:**
```json
{
  "extraction_spec": {
    "fields": [
      {
        "name": "content_html",
        "label": "Content HTML",
        "type": "string",
        "required": false,
        "extractor": {
          "selector": "div.content",
          "attribute": "html",
          "multiple": false
        },
        "transforms": []
      },
      {
        "name": "content_text",
        "label": "Content Text",
        "type": "string",
        "required": false,
        "extractor": {
          "selector": "div.content",
          "attribute": "html",
          "multiple": false
        },
        "transforms": [
          { "op": "html_to_text" },
          { "op": "collapse_ws" }
        ]
      }
    ]
  }
}
```

**Результат:**
```json
{
  "fields": {
    "content_html": "<p>First <b>paragraph</b></p>\n  <p>Second paragraph</p>",
    "content_text": "First paragraph Second paragraph"
  }
}
```

---

### Пример 10: Извлечение чисел из текста (рейтинг)

**HTML:**
```html
<div class="rating">Rating: 4.5 out of 5</div>
```

**Конфигурация:**
```json
{
  "extraction_spec": {
    "fields": [
      {
        "name": "rating",
        "label": "Product Rating",
        "type": "float",
        "required": true,
        "extractor": {
          "selector": "div.rating",
          "attribute": "text",
          "multiple": false
        },
        "transforms": [
          { "op": "parse_price" }
        ]
      }
    ]
  }
}
```

**Результат:**
```json
{
  "fields": {
    "rating": 4.5
  }
}
```

**Примечание:** `parse_price` извлекает первое число из строки (в данном случае `4.5`).

---

### Пример 11: Комплексная конфигурация для e-commerce

**HTML:**
```html
<html>
<head>
  <meta property="og:title" content="Product XYZ">
  <meta property="og:image" content="https://example.com/image.jpg">
</head>
<body>
  <h1 class="product-title">Product XYZ</h1>
  <span class="price">$129.99</span>
  <div class="description">
    <p>High quality product with many features.</p>
  </div>
  <ul class="features">
    <li>Feature 1</li>
    <li>Feature 2</li>
    <li>Feature 3</li>
  </ul>
  <div class="stock">In stock: 15 items</div>
  <a href="https://example.com/reviews">Reviews</a>
  <a href="/contact">Contact</a>
</body>
</html>
```

**Конфигурация:**
```json
{
  "extraction_spec": {
    "fields": [
      {
        "name": "og_title",
        "label": "OpenGraph Title",
        "type": "string",
        "required": true,
        "extractor": {
          "selector": "meta[property='og:title']",
          "attribute": "content",
          "multiple": false
        },
        "transforms": []
      },
      {
        "name": "og_image",
        "label": "OpenGraph Image",
        "type": "url",
        "required": false,
        "extractor": {
          "selector": "meta[property='og:image']",
          "attribute": "content",
          "multiple": false
        },
        "transforms": []
      },
      {
        "name": "product_title",
        "label": "Product Title",
        "type": "string",
        "required": true,
        "extractor": {
          "selector": "h1.product-title",
          "attribute": "text",
          "multiple": false
        },
        "transforms": [
          { "op": "trim" },
          { "op": "collapse_ws" }
        ]
      },
      {
        "name": "price",
        "label": "Product Price",
        "type": "float",
        "required": true,
        "extractor": {
          "selector": "span.price",
          "attribute": "text",
          "multiple": false
        },
        "transforms": [
          { "op": "parse_price" }
        ]
      },
      {
        "name": "description",
        "label": "Product Description",
        "type": "string",
        "required": false,
        "extractor": {
          "selector": "div.description p",
          "attribute": "text",
          "multiple": false
        },
        "transforms": [
          { "op": "trim" }
        ]
      },
      {
        "name": "features",
        "label": "Product Features",
        "type": "json",
        "required": false,
        "extractor": {
          "selector": "ul.features li",
          "attribute": "text",
          "multiple": true
        },
        "transforms": []
      },
      {
        "name": "stock_count",
        "label": "Stock Count",
        "type": "int",
        "required": false,
        "extractor": {
          "selector": "div.stock",
          "attribute": "text",
          "multiple": false
        },
        "transforms": [
          { "op": "parse_int" }
        ]
      }
    ]
  }
}
```

**Результат:**
```json
{
  "fields": {
    "og_title": "Product XYZ",
    "og_image": "https://example.com/image.jpg",
    "product_title": "Product XYZ",
    "price": 129.99,
    "description": "High quality product with many features.",
    "features": ["Feature 1", "Feature 2", "Feature 3"],
    "stock_count": 15
  }
}
```

---

## 8. Best Practices и рекомендации

### 8.1 Проектирование устойчивых селекторов

**✅ Хорошо:**
- Используйте семантические CSS-классы и ID: `"h1.product-title"`, `"#main-content"`
- Предпочитайте data-атрибуты (если есть): `"div[data-product-id]"`
- Используйте атрибутные селекторы для meta-тегов: `"meta[property='og:title']"`

**❌ Плохо:**
- Хрупкие селекторы с nth-child: `"div:nth-child(3) > p:nth-child(5)"` (может сломаться при изменении структуры)
- Слишком общие селекторы: `"div"` (может вернуть не те элементы)
- Сложные цепочки без смысловой привязки: `"body > div > div > div > p"`

### 8.2 Обработка отсутствующих данных

- Используйте `required: false` для опциональных полей
- Не полагайтесь на порядок элементов — используйте `multiple: true` + `index` с осторожностью

### 8.3 Оптимизация производительности

- Избегайте дублирования селекторов (не извлекайте одно и то же поле дважды)
- Используйте `multiple: false`, если вам нужен только первый элемент
- Применяйте `limit` к массивам, если вам не нужны все элементы

### 8.4 Отладка проблем с извлечением

**Если поле возвращает `null`:**
1. Проверьте, что CSS-селектор находит элементы (используйте DevTools браузера)
2. Убедитесь, что `attribute` указан правильно
3. Проверьте, что элемент не скрыт через JavaScript (парсер видит только исходный HTML)

**Если массив пустой:**
1. Проверьте, что `multiple: true`
2. Убедитесь, что элементы содержат непустой текст/атрибут
3. Проверьте фильтрацию через трансформации (`unique`, `limit`)

**Если тип не конвертируется:**
1. Проверьте формат данных (например, `"$19.99"` нужно обработать через `parse_price`)
2. Используйте `type: "string"` для отладки (чтобы увидеть сырое значение)
3. Добавьте трансформации для очистки (`trim`, `collapse_ws`)

### 8.5 Работа с динамическим контентом

**Ограничение:** Парсер работает только с исходным HTML (SSR). JavaScript-рендеринг (SPA, CSR) не поддерживается.

**Решение:**
- Для SPA используйте headless-браузер (Puppeteer, Playwright) для рендеринга страницы перед парсингом
- Ищите API-эндпоинты, которые возвращают JSON (часто доступны в Network вкладке DevTools)

### 8.6 Тестирование конфигураций

**Рекомендации:**
1. Начните с простых селекторов и постепенно добавляйте сложность
2. Тестируйте на нескольких страницах одного типа (например, разные товары)
3. Используйте preview API (`/api/v1/previews`) для быстрой проверки конфигураций
4. Проверяйте edge cases: пустые страницы, страницы с отсутствующими элементами

### 8.7 Документирование конфигураций

- Используйте поле `label` для описания назначения поля
- Добавляйте комментарии в JSON (если поддерживается) или ведите отдельную документацию
- Сохраняйте примеры HTML-страниц для тестирования

---

## Дополнительные ресурсы

### Полезные ссылки

- **goquery документация:** https://github.com/PuerkitoBio/goquery
- **CSS Selectors Reference:** https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Selectors
- **JSON Schema для валидации:** Рассмотрите использование JSON Schema для валидации конфигураций

### Поддержка

Если вы столкнулись с проблемами или нашли баг, создайте issue в репозитории проекта или обратитесь к команде разработки.

---

**Версия документа:** 1.0
**Дата последнего обновления:** 2026-01-11
**Автор:** Distributed Crawler Team
