# Region Parameter

## Summary

`region` is a startup-only label injected into fetch worker processes. It identifies
which logical/geographic pool a worker belongs to. All queue connection details
(broker URL, queue name) are read from env vars at startup — there is no
database-driven queue endpoint discovery based on region.

Region does **not** affect:

- crawl scope or URL filtering
- job creation or task routing logic
- parser workers (they are always region-agnostic)
- Queue Admin endpoints or routing rules

---

## How It Works

### 1. Helm Layer

`fetchWorker.regions` fans out one Kubernetes `Deployment` per region:

- empty list → one deployment, `WORKER_REGION` not set
- non-empty list → one deployment per region, each with `WORKER_REGION=<region>`

Deployment names get a `-<region>` suffix; pod labels get `worker-region: <region>`.

`parserWorker.regions` follows the same Helm expansion but `WORKER_REGION` has no
effect on parser workers at runtime.

Example:

```yaml
fetchWorker:
  replicaCount: 3
  regions:
    - us-east
    - eu-west
```

Creates:
- `...fetch-worker-us-east`, 3 replicas, `WORKER_REGION=us-east`
- `...fetch-worker-eu-west`, 3 replicas, `WORKER_REGION=eu-west`

### 2. Docker / Local

Pass `WORKER_REGION=<region>` as an env var when launching a fetch worker container
or process. The `multi_region_run.sh` script handles this automatically.

Each regional worker pool must be configured with the correct queue connection
details in its env vars or config file (`.worker.env`):

| Env var | Purpose |
|---------|---------|
| `WORKER_REGION` | Region label (logged, used by monitoring) |
| `RABBITMQ_URL` | Broker URL for this pool |
| `RABBITMQ_CRAWL_QUEUE_NAME` | Queue this pool consumes from |
| `RABBITMQ_PARSING_QUEUE_NAME` | Queue this pool publishes to after fetch |

For Kafka workers, use the equivalent `KAFKA_*` env vars.

### 3. Worker Behavior

`WORKER_REGION` is read at startup for logging and monitoring purposes only.
It does **not** trigger any database lookup or broker override at runtime.
All queue configuration comes entirely from env vars.

---

## Deployment Topologies

### Default (single-region)

```bash
./deploy/scripts/default_run.sh
```

One fetch worker pool. No `WORKER_REGION` set. All workers use the queue names
defined in the static broker config.

### Multi-region

```bash
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west
```

One fetch worker pool per region, each with `WORKER_REGION` set.

To route different regions to different queues, supply different env vars
per worker pool. For Docker: use per-service env overrides in Compose.
For local: use per-region `.worker.env` files. For k8s: use Helm values with
per-region broker settings injected via ConfigMap or Secret.

---

## What Region Influences

`region` influences:

- fetch worker deployment topology (one Deployment per region in Helm)
- pod labeling (`worker-region: <region>`)
- log output and worker monitor registration label

`region` does **not** influence:

- which broker connection a worker opens (comes from env vars)
- which queue/topic a worker consumes from (comes from env vars)
- crawl logic, URL discovery, parser extraction
- job scheduling or task prioritization
- weighted queue selection logic (job routing is separate)
