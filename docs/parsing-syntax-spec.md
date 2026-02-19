# Parsing Syntax Spec (ExtractionSpec)

This document defines the parsing contract used by `ExtractionSpec` in `api/v1/models.proto`.

## 1. Contract (from proto)

```proto
message ExtractorSpec {
  string selector = 1;
  string attribute = 2;
  bool multiple = 3;
  optional int32 index = 4;
}

message TransformSpec {
  string op = 1;
  string arg = 2;
}

message FieldSpec {
  string name = 1;
  string type = 2; // "string" | "int" | "float" | "bool" | "url" | "json"
  bool required = 3;
  ExtractorSpec extractor = 4;
  repeated TransformSpec transforms = 5;
}

message ItemsSpec {
  string container_selector = 1;
  repeated FieldSpec fields = 2;
}

message PaginationSpec {
  string name = 1;
  string selector = 2;
  string attribute = 3;
  bool multiple = 4;
}

message ExtractionSpec {
  repeated FieldSpec fields = 1;
  repeated PaginationSpec pagination = 3;
  optional ItemsSpec items = 4;
}
```

## 2. Important Rules

1. `FieldSpec` supports only: `name`, `type`, `required`, `extractor`, `transforms`.
   - `label` is not part of the contract.
2. `type` must be one of:
   - `string`, `int`, `float`, `bool`, `url`, `json`
3. `TransformSpec.op` supported values:
   - `trim`, `lower`, `upper`, `normalize_url`, `unique`, `limit`, `to_int`, `to_float`, `parse_price`, `html_to_text`, `collapse_ws`, `sha256`
4. `TransformSpec.arg` is a string (JSON-encoded argument when required by transform).
5. `items.container_selector` scopes extraction to repeated item containers.
6. `pagination` defines selectors for next-page URLs.

## 3. Items Extraction Semantics

When `items` is configured:

1. Find all elements by `container_selector`.
2. For each matched container, run each `FieldSpec` relative to that container.
3. Build one object per container.
4. Return a structured `items` array (instead of parallel arrays).

Example result shape:

```json
{
  "fields": {
    "page_title": "All products"
  },
  "items": [
    {
      "name": "A Light in the Attic",
      "price": 51.77,
      "availability": "In stock"
    }
  ]
}
```

## 4. Minimal Example (Books to Scrape)

```json
{
  "extraction_spec": {
    "fields": [
      {
        "name": "page_title",
        "type": "string",
        "required": false,
        "extractor": {
          "selector": "div.page-header h1",
          "attribute": "text",
          "multiple": false
        },
        "transforms": [
          { "op": "trim" },
          { "op": "collapse_ws" }
        ]
      }
    ],
    "items": {
      "container_selector": "article.product_pod",
      "fields": [
        {
          "name": "name",
          "type": "string",
          "required": true,
          "extractor": {
            "selector": "h3 a",
            "attribute": "title",
            "multiple": false
          },
          "transforms": [
            { "op": "trim" },
            { "op": "collapse_ws" }
          ]
        },
        {
          "name": "price",
          "type": "float",
          "required": true,
          "extractor": {
            "selector": "p.price_color",
            "attribute": "text",
            "multiple": false
          },
          "transforms": [
            { "op": "parse_price" }
          ]
        },
        {
          "name": "availability",
          "type": "string",
          "required": false,
          "extractor": {
            "selector": "p.instock.availability",
            "attribute": "text",
            "multiple": false
          },
          "transforms": [
            { "op": "trim" },
            { "op": "collapse_ws" }
          ]
        },
        {
          "name": "url",
          "type": "url",
          "required": true,
          "extractor": {
            "selector": "h3 a",
            "attribute": "href",
            "multiple": false
          },
          "transforms": [
            { "op": "normalize_url" }
          ]
        }
      ]
    },
    "pagination": [
      {
        "name": "next_page",
        "selector": "ul.pager li.next a",
        "attribute": "href",
        "multiple": false
      }
    ]
  }
}
```
