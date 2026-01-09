# CreateJob Request Contract (Crawler/Parser System)

This document summarizes a practical JSON contract for creating a crawling/parsing **Job**, including:
- a minimal payload (`name + seeds`)
- an extraction template DSL (`fields + transforms + metrics`)
- advanced crawling/runtime options (scope, link discovery, auth, proxy, rate limit, retries, scheduling)
- how `metrics` and `output` work when **Postgres stores scan/run records + extracted JSONB per task/page**

---

## 1) Minimal CreateJob body

Start simple and keep forward-compatible naming.

### Option A (recommended): `seeds`
```json
{
  "name": "try parse bool.dev 2",
  "seeds": [
    "https://bool.dev/blog/detail/voprosy-na-sobesedovanii-dlya-senior-net-developer"
  ]
}
Option B (backward compatible): urls
json
Копировать код
{
  "name": "try parse bool.dev 2",
  "urls": [
    "https://bool.dev/blog/detail/voprosy-na-sobesedovanii-dlya-senior-net-developer"
  ]
}
2) Extraction template (fields/transforms/metrics)
Place your template DSL under extract.template so it stays separate from system/runtime options.

json
Копировать код
{
  "name": "Products crawl",
  "seeds": ["https://example.com/product/123"],
  "extract": {
    "mode": "single_page",
    "template": {
      "version": "1",
      "fields": [
        {
          "name": "product_title",
          "type": "string",
          "required": true,
          "extractor": { "source": "html", "selector_type": "css", "selector": "h1", "multiple": false },
          "transforms": [{ "op": "trim" }]
        },
        {
          "name": "price",
          "type": "float",
          "required": true,
          "extractor": { "source": "html", "selector_type": "css", "selector": ".price, [itemprop=price]", "multiple": false },
          "transforms": [{ "op": "trim" }, { "op": "parse_price" }]
        },
        {
          "name": "in_stock",
          "type": "bool",
          "required": false,
          "extractor": { "source": "text", "selector_type": "regex", "selector": "(in stock|out of stock|нет в наличии|в наличии)", "multiple": false },
          "transforms": [{ "op": "to_stock_bool" }]
        }
      ],
      "metrics": [
        { "name": "price_present", "op": "field_present", "input": "price" }
      ]
    }
  }
}
What metrics means
metrics is observability / quality scoring, not storage.
Example:

json
Копировать код
{ "name": "price_present", "op": "field_present", "input": "price" }
Meaning: after parsing each document, compute whether price exists and aggregate it across the run:

parsed_total

parsed_with_price

parsed_without_price

percentage trends (detect selector/layout breakage)

Recommended storage: aggregate metrics into job_runs.metrics (JSONB) or run_metrics table.

3) Full CreateJob body (with advanced options)
Designed so UI can show:

Basic: name, seeds, template

Advanced (optional): scope, link rules, auth/proxy, throttling, retries, schedule

json
Копировать код
{
  "name": "Monitor competitor prices",
  "description": "Daily crawl product pages and extract price/title/stock",

  "seeds": ["https://shop.example.com/catalog"],

  "extract": {
    "mode": "crawl",
    "template": { "version": "1", "fields": [], "metrics": [] },

    "output": {
      "sink": "postgres",
      "store": {
        "runs_table": "job_runs",
        "results_table": "crawl_results",
        "format": "jsonb",
        "write_mode": "append"
      },
      "raw": {
        "store_html": false
      }
    }
  },

  "scope": {
    "allowed_domains": ["shop.example.com"],
    "deny_url_patterns": ["\\/cart", "\\/checkout"],
    "max_depth": 3,
    "max_pages": 2000
  },

  "link_discovery": {
    "enabled": true,
    "allow_url_patterns": ["\\/product\\/"],
    "deny_url_patterns": ["\\/login", "\\/signup"],
    "canonicalize": true
  },

  "http": {
    "user_agent": "DenisCrawler/1.0",
    "timeout_ms": 15000,
    "headers": { "accept-language": "ru-RU,ru;q=0.9,en;q=0.8" },

    "auth": {
      "type": "bearer",
      "bearer_token": "****"
    },

    "proxy": {
      "enabled": false,
      "rotation": "round_robin",
      "cooldown_ms": 2000,
      "endpoints": [
        "http://user:pass@1.2.3.4:8000",
        "http://user:pass@5.6.7.8:8000"
      ]
    }
  },

  "rate_limit": {
    "scope": "per_domain",
    "max_concurrency": 4,
    "rps": 2.0,
    "jitter_ms": 250
  },

  "retry": {
    "max_attempts": 4,
    "backoff": {
      "type": "exponential",
      "initial_ms": 500,
      "multiplier": 2.0,
      "max_ms": 10000
    },
    "retry_on_status": [429, 500, 502, 503, 504]
  },

  "schedule": {
    "type": "cron",
    "cron": "0 6 * * *",
    "timezone": "Europe/Moscow"
  },

  "tags": ["competitors", "prices", "daily"]
}
4) What output means (aligned to “Postgres stores runs + extracted JSONB results”)
output answers: where and how to store parsed results and raw content.

If you store extracted payloads as JSONB in Postgres tables:

output.store.results_table points to the table receiving {run_id, task_id, url, extracted_jsonb, ...}

output.store.runs_table points to runs/scan tracking table

output.raw controls raw HTML storage strategy:

store_html: false (store nothing)

store_html: true + store to MinIO/S3 and keep only a reference in Postgres

Example: store raw HTML in MinIO

json
Копировать код
"raw": {
  "store_html": true,
  "storage": "minio",
  "bucket": "raw-pages",
  "key_prefix": "job/${job_id}/run/${run_id}/"
}
5) Recommended API separation: Job vs Run
To keep config and execution history clean:

POST /jobs → creates a Job (configuration + template)

POST /jobs/{job_id}/runs → creates a JobRun (execution instance), updates progress counters and metrics

JobRun is NOT part of CreateJob body; it’s produced when you launch.

6) Mapping “options classes” → JSON sections
AuthOptions → http.auth

type: cookie | basic | bearer

cookie: cookie header/string or structured cookies

basic: username/password

bearer_token: token string

ProxyOptions → http.proxy

endpoints, rotation, cooldown_ms, enabled flag

RateLimitPolicy → rate_limit

per-domain throttling: concurrency + rps + jitter

RetryPolicy → retry

attempts + backoff + retryable status codes

ScheduleOptions → schedule

type: cron (+ timezone) or type: once (no schedule)

ScopeRules → scope

allowed domains, deny patterns, max depth/pages

LinkDiscovery → link_discovery

enabled, allow/deny patterns, canonicalization

7) Suggested Postgres tables
job_runs (scan/run tracking)
Fields (typical):

id (run_id), job_id

status, started_at, finished_at

counters jsonb (enqueued/fetched/parsed/failed)

metrics jsonb (aggregated metrics from template)

config_snapshot jsonb (optional: snapshot job config at run time)

crawl_results (per page/task result)
Fields (typical):

id, run_id, task_id

url, status_code, fetched_at

extracted jsonb (the parsed fields)

errors jsonb (parse/fetch failures)

raw_ref text/jsonb (optional pointer to MinIO/S3 raw content)