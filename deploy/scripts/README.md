# Deploy Scripts

Two top-level launcher scripts cover the two main deployment topologies.

```
deploy/scripts/
├── default_run.sh        ← single-region (standard) stack
├── multi_region_run.sh   ← multi-region fetch workers
├── local/                ← low-level local process helpers
├── docker/               ← low-level Docker Compose helpers
└── k8s/                  ← low-level Helm/minikube helpers
```

The launchers are thin wrappers. They parse a `--mode` flag and delegate to the
appropriate low-level script in `local/`, `docker/`, or `k8s/`. Any arguments
you pass after `--` are forwarded verbatim to the underlying script.

---

## `default_run.sh` — standard stack, no regions

Starts the full distributed-crawler stack with a single default fetch worker
pool (no `WORKER_REGION` set).

### Quick start

```bash
# Local processes (go run, default .env / .worker.env)
./deploy/scripts/default_run.sh

# Docker Compose
./deploy/scripts/default_run.sh --mode docker

# Kubernetes (minikube + Helm)
./deploy/scripts/default_run.sh --mode k8s
```

### Options

| Flag | Mode | Default | Description |
|------|------|---------|-------------|
| `--mode` | all | `local` | `local`, `docker`, or `k8s` |
| `--config <path>` | local | `.env` | API server config file |
| `--worker-config <path>` | local | `.worker.env` | Worker config file |
| `--build` | local | off | Build Go binaries first |
| `--no-build` | docker/k8s | off | Skip image build |
| `--app-only` | docker | off | Skip infra startup |
| `--tag <tag>` | docker/k8s | `latest` | Image tag |
| `--registry <name>` | docker/k8s | `distributed-crawler` | Image name prefix |
| `--port-forward` | k8s | off | Start port-forward after deploy |
| `--full-observability` | k8s | off | Enable Prometheus/Grafana/OpenSearch |
| `--jwt-secret <value>` | k8s | dev default | JWT secret for the API |
| `--pg-password <pwd>` | k8s | dev default | PostgreSQL password |
| `--default-user-password <pwd>` | k8s | dev default | Admin UI password |
| `--messaging-broker <kind>` | k8s | `rabbitmq` | `rabbitmq`, `kafka`, `grpc_memory` |

Pass extra flags to the underlying script after `--`:

```bash
./deploy/scripts/default_run.sh --mode k8s -- \
  --jwt-secret supersecret \
  --app-set grpcServer.replicaCount=2
```

---

## `multi_region_run.sh` — regional fetch workers

Starts the stack with one fetch worker pool per region. Each pool has its own
`WORKER_REGION` label and uses its own queue connection details. Parser workers
always run in a single shared pool — they are not region-aware.

Queue connection details for each regional pool come entirely from **env vars
or the worker config file** at startup. There is no database-driven queue
endpoint discovery. Configure each regional pool with the correct
`RABBITMQ_URL` / `RABBITMQ_CRAWL_QUEUE_NAME` (or Kafka equivalents) before
starting the stack.

### Quick start

```bash
# Docker Compose — two regional fetch worker pools
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west

# Kubernetes
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west --mode k8s

# Local processes
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west --mode local
```

### Options

| Flag | Mode | Default | Description |
|------|------|---------|-------------|
| `--regions <csv>` | all | **required** | Comma-separated region names |
| `--mode` | all | `docker` | `local`, `docker`, or `k8s` |
| `--config <path>` | local | `.env` | API server config file |
| `--worker-config <path>` | local | `.worker.env` | Worker config for non-fetch workers |
| `--build` | local | off | Build Go binaries first |
| `--no-build` | docker/k8s | off | Skip image build |
| `--tag <tag>` | docker/k8s | `latest` | Image tag |
| `--registry <name>` | docker | `distributed-crawler` | Image name prefix |
| `--port-forward` | k8s | off | Start port-forward after deploy |
| `--full-observability` | k8s | off | Enable Prometheus/Grafana/OpenSearch |
| `--jwt-secret <value>` | k8s | dev default | JWT secret for the API |
| `--pg-password <pwd>` | k8s | dev default | PostgreSQL password |
| `--messaging-broker <kind>` | k8s | `rabbitmq` | Broker type |

Pass extra flags to the underlying script after `--`:

```bash
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west --mode k8s -- \
  --jwt-secret supersecret \
  --port-forward
```

### What each mode does

#### `--mode local`

1. Starts `grpc_server`, `parser_worker`, `export_worker`, `scheduler_worker`
   as background processes (one per component, PID files under `.pids/`).
2. Starts one `fetch_worker` process per region with `WORKER_REGION=<region>`.

Logs land in `<project_root>/logs/`, named `fetch-worker-<region>.log`.
Use `./deploy/scripts/local/stop-all.sh` to stop everything.

#### `--mode docker`

1. Runs `deploy/scripts/docker/deploy-all.sh` with `APP_COMPONENTS_CSV` set to
   start all components **except** `fetch-worker`.
2. Runs `docker compose run --detach --no-deps -e WORKER_REGION=<region> fetch-worker`
   once per region, creating one detached container per region.

Use `docker ps` to see the running fetch-worker containers.
Use `docker compose down` from the project root to stop everything.

#### `--mode k8s`

Delegates to `deploy/scripts/k8s/launch-minikube.sh` with:

```
--app-set fetchWorker.regions={us-east,eu-west}
```

This instructs Helm to create one `fetch-worker` Deployment per region, each
with `WORKER_REGION=<region>` injected and a `-<region>` suffix in the
Deployment name.

---

## Port-forwarding (`k8s/port-forward.sh`)

Opens `kubectl port-forward` tunnels for any combination of infra and app
services. All forwards run in the background; **Ctrl-C** stops them all.

### Quick start

```bash
# Forward everything (infra + app)
./deploy/scripts/k8s/port-forward.sh

# Forward selected services only
./deploy/scripts/k8s/port-forward.sh grpc-server ui
./deploy/scripts/k8s/port-forward.sh postgresql rabbitmq minio
```

Or start it automatically after a deploy with `--port-forward`:

```bash
./deploy/scripts/default_run.sh --mode k8s --port-forward
./deploy/scripts/multi_region_run.sh --regions us-east,eu-west --mode k8s --port-forward
```

### Available services and local ports

| Alias | Local port(s) | URL / note |
|-------|--------------|------------|
| `grpc-server` | `8083`, `8084` | gRPC API / HTTP gateway |
| `ui` | `4200` | Admin UI → http://localhost:4200 |
| `postgresql` | `54322` | `localhost:54322` |
| `rabbitmq` | `5672`, `15672` | AMQP / Management UI → http://localhost:15672 (guest/guest) |
| `minio` | `9000`, `9001` | S3 API / Console → http://localhost:9001 (minioadmin/minioadmin) |
| `redis` | `6379` | `localhost:6379` |
| `redisinsight` | `8001` | http://localhost:8001 |
| `jaeger` | `16686` | http://localhost:16686 |
| `prometheus` | `9090` | http://localhost:9090 |
| `grafana` | `3000` | http://localhost:3000 (admin/changeme-grafana-password) |
| `opensearch` | `9200` | http://localhost:9200 |
| `opensearch-dashboards` | `5601` | http://localhost:5601 |

> Observability services (jaeger, prometheus, grafana, opensearch, opensearch-dashboards)
> are only deployed when `--full-observability` was passed to the launch script.

### Environment overrides

| Variable | Default | Description |
|----------|---------|-------------|
| `INFRA_NAMESPACE` | `infra` | Namespace of the infra Helm release |
| `INFRA_RELEASE` | `infra` | Name of the infra Helm release |
| `APP_NAMESPACE` | `crawler` | Namespace of the app Helm release |
| `APP_RELEASE` | `crawler` | Name of the app Helm release |

```bash
APP_NAMESPACE=my-crawler ./deploy/scripts/k8s/port-forward.sh ui grpc-server
```

---

## Relationship to existing low-level scripts

The launchers do not replace the scripts in `local/`, `docker/`, and `k8s/`.
Those remain useful for targeted operations:

| Script | Purpose |
|--------|---------|
| `local/build.sh` | Build Go binaries |
| `local/run.sh <component>` | Run a single component |
| `local/start-all.sh` / `stop-all.sh` | Start/stop all local processes |
| `docker/deploy-all.sh` | Full Docker Compose deploy |
| `docker/deploy-component.sh` | Redeploy a single compose service |
| `docker/teardown.sh` | Delete all Docker containers and volumes on the host |
| `k8s/launch-minikube.sh` | Full minikube+Helm deploy with all options |
| `k8s/port-forward.sh` | Forward k8s service ports locally |
| `k8s/teardown.sh` | Remove Helm releases and namespace |
