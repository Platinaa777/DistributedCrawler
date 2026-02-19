You are a senior Go backend engineer working on a distributed web crawling platform.

The system already supports:

Recursive crawling with depth and domain scopes

Rate limiting and retries

Extraction via ExtractionSpec DSL (see full DSL documentation) 

parsing-syntax-spec

Page-level field extraction (returns { "fields": { ... } })

Example job configuration similar to Books to Scrape 

request

Your task is to extend the system with two major capabilities:

1️⃣ Add Support for Structured item {} Parsing
Current Limitation

The system extracts page-level fields only:

{
  "fields": {
    "item_names": [...],
    "item_prices": [...],
    "item_availability": [...]
  }
}


This results in parallel arrays instead of structured objects.

We need to support structured item extraction:

{
  "items": [
    {
      "name": "Book A",
      "price": 19.99,
      "availability": "In stock"
    },
    {
      "name": "Book B",
      "price": 25.00,
      "availability": "Out of stock"
    }
  ]
}

Required New Feature: ItemsSpec

Extend ExtractionSpec with support for item-level extraction.

New Structure
type ExtractionSpec struct {
    Fields []FieldSpec     // existing page-level fields
    Items  *ItemsSpec      // NEW
}

type ItemsSpec struct {
    ContainerSelector string      // CSS selector for each item container
    Fields            []FieldSpec // fields extracted inside each container
}

Item Extraction Semantics

When ItemsSpec is defined:

Select all elements matching ContainerSelector

For each matched element:

Create isolated DOM scope (subtree)

Apply FieldSpec definitions relative to that container

Apply transforms

Convert types

Produce:

{
  "items": [...]
}

Important Rules

All existing FieldSpec, ExtractorSpec, Transforms, and ValueType rules apply exactly as defined in DSL spec 

parsing-syntax-spec

Selectors inside item fields must be evaluated relative to container

multiple: true inside item field applies only within that container

If a field fails:

If required = true → log warning

Continue processing other fields and items

If container list is empty → return empty items: []

Example (Books to Scrape Refactor)

Current config 

request

 extracts arrays:

{
  "selector": "article.product_pod h3 a",
  "multiple": true
}


Refactor into structured items:

{
  "extraction_spec": {
    "items": {
      "container_selector": "article.product_pod",
      "fields": [
        {
          "name": "name",
          "type": "string",
          "extractor": {
            "selector": "h3 a",
            "attribute": "title",
            "multiple": false
          }
        },
        {
          "name": "price",
          "type": "float",
          "extractor": {
            "selector": "p.price_color",
            "attribute": "text"
          },
          "transforms": [
            { "op": "parse_price" }
          ]
        },
        {
          "name": "availability",
          "type": "string",
          "extractor": {
            "selector": "p.availability",
            "attribute": "text"
          },
          "transforms": [
            { "op": "trim" },
            { "op": "collapse_ws" }
          ]
        }
      ]
    }
  }
}

2️⃣ Add Crawling Mode Flag (Pagination vs Links)

Currently crawler follows:

Pagination links (e.g. li.next a)

Regular <a href> links inside page

We need explicit crawl behavior control.

Add New Config Field

Extend job config:

"crawl_mode": "pagination_only" | "links_only" | "pagination_and_links"


Default:

pagination_and_links

Behavior Definition
pagination_only

Crawler should:

Follow only pagination links

Pagination is defined by:

Explicit pagination field (e.g. next_page_url)

OR configurable selector (future extensibility)

Ignore all other <a href> links.

links_only

Crawler should:

Follow <a href> links that match scope rules

Ignore pagination logic completely

pagination_and_links

Crawler should:

Follow both pagination links

And standard scoped <a href> links

Implementation Requirements

Crawl mode must affect link extraction stage

Scope filtering (allowed_domains, deny patterns) still applies

Must not break depth control

Must not break retries / rate limiting

Must not affect extraction engine

3️⃣ Backward Compatibility

If ItemsSpec == nil, behavior remains unchanged

If crawl_mode not defined → default to pagination_and_links

Existing jobs must continue to work

4️⃣ Expected Output Format

If only page fields:

{
  "fields": {...}
}


If items present:

{
  "items": [...],
  "fields": {...} // optional page-level metadata
}

5️⃣ Engineering Requirements

Clean Go implementation

No reflection hacks

Reuse existing FieldSpec execution logic

Unit tests required:

Item extraction success

Item extraction with missing fields

Empty container

All crawl modes

No breaking public API without version bump

6️⃣ Constraints

Must strictly follow DSL semantics defined in spec 

parsing-syntax-spec

Must support example job config format 

request

Code must be production-ready

Handle edge cases safely

Log warnings instead of crashing

You must implement this feature in a way that is:

Deterministic

Backward compatible

Extensible

Cleanly integrated into current architecture

Do not redesign the whole crawler. Extend it properly.