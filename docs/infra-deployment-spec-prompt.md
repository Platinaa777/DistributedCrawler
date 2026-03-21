# Infra Deployment Implementation-Spec Prompt

Use this prompt when you want an agent or engineer to produce an implementation spec for hardening this repository's Docker and Kubernetes deployment workflows.

## Prompt

```text
You are reviewing and specifying deployment hardening work for the `distributed-crawler` repository.

Your task is to produce an implementation spec, not to write code yet.

The goal is to make both local Docker deployment and Kubernetes/Helm deployment reliable, repeatable, and environment-driven so the full stack can be brought up from scratch and all components work together correctly.

Work only from the current repo state. Do not give generic infrastructure advice. Ground every recommendation in the actual scripts, compose files, Helm charts, templates, and application env contract that exist in this repository.

## Repo Surfaces You Must Review

- `deploy/scripts/docker/deploy-all.sh`
- `deploy/scripts/docker/deploy-component.sh`
- `deploy/scripts/docker/teardown.sh`
- `docker-compose.yaml`
- `docker-compose.app.yaml`
- `deploy/scripts/k8s/deploy-all.sh`
- `deploy/scripts/k8s/deploy-infra.sh`
- `deploy/scripts/k8s/deploy-component.sh`
- `deploy/scripts/k8s/build-images.sh`
- `deploy/helm/distributed-crawler/**`
- `deploy/helm/infra/**`
- `internal/config/env/**`
- Any directly-related config/template/helper files referenced by the paths above

## Primary Objectives

1. Docker scripts must work correctly.
2. All local infra must come up correctly.
3. All application components must work with each other correctly.
4. Runtime parameters must be passed through env vars in a consistent, auditable way.
5. Helm/Kubernetes deployment must be ready to bootstrap a new cluster repeatedly with minimal manual correction.
6. The deployment contract must be explicit enough that Docker and Helm do not drift from the app's real env requirements.

## Important Current Repo Facts To Account For

- Docker deploy currently uses `deploy/scripts/docker/deploy-all.sh` plus `docker-compose.yaml` and `docker-compose.app.yaml`.
- Kubernetes deploy currently uses `deploy/scripts/k8s/deploy-all.sh` and optionally `deploy/scripts/k8s/deploy-infra.sh`.
- The app consumes configuration from environment variables in `internal/config/env/**`.
- Docker Compose and Helm currently duplicate parts of the app env contract in different shapes.
- The app Helm deploy script comments mention embedded Bitnami subcharts, but the app chart metadata does not declare those dependencies and the chart is templated locally.
- External-infra Helm mode currently depends on fixed service names and cross-namespace DNS assumptions.
- Startup ordering, migrations, secrets, readiness, and inter-service connectivity all need explicit review.

## Canonical Application Env Contract

Treat the application's env inputs as the source of truth. The spec must explain how Docker env files and Helm values generate these without drift:

- `PG_DSN`
- `RABBITMQ_URL`
- `RABBITMQ_*_QUEUE_NAME`
- `REDIS_ADDRESS`
- `REDIS_PWD`
- `MINIO_*`
- `MESSAGING_BROKER`
- `KAFKA_*`
- `MEMORY_BROKER_*`
- `OTEL_*`
- `OPENSEARCH_*`
- `JWT_*`
- `DEFAULT_USER_*`
- `HTTP_CORS_ALLOWED_ORIGINS`
- `QUEUE_SECRETS_*`

Also review any additional env vars that the repo actually consumes and include them if they materially affect deployment correctness.

## Public Interfaces That Must Be Preserved Or Deliberately Revised

### Docker script interface
- `REGISTRY`
- `TAG`
- `APP_ONLY`
- `NO_BUILD`
- pass-through Docker Compose args

### Helm script interface
- `RELEASE_NAME`
- `NAMESPACE`
- `VALUES_ENV`
- `EXTERNAL_INFRA`
- pass-through Helm args

If you propose changes to these interfaces, mark them explicitly as breaking or non-breaking and justify them.

## What You Must Produce

Produce a decision-complete implementation spec with these sections:

1. Executive summary
2. Current-state findings
3. Target deployment architecture
4. Canonical configuration contract
5. Docker deployment design
6. Kubernetes/Helm deployment design
7. Migration and startup ordering design
8. Validation and health-check strategy
9. Acceptance criteria
10. Prioritized implementation backlog
11. Risks, assumptions, and open questions

## Required Content Inside The Spec

### Current-state findings
Identify concrete repo-specific inconsistencies, gaps, and risks. At minimum evaluate:

- mismatch risk between Compose env wiring and Helm env wiring
- mismatch risk between deployment config and `internal/config/env/**`
- misleading or outdated script comments
- readiness vs. simple process-start ordering
- migrations running before dependencies are truly ready
- secret handling consistency across Docker and Helm
- cluster DNS and namespace coupling in external-infra mode
- image build/tag/pull assumptions
- local-vs-cluster differences that can break connectivity

### Target deployment architecture
Define the intended steady-state deployment model for:

- local Docker full stack
- local Docker app-only against existing infra
- Kubernetes self-contained app deployment
- Kubernetes split infra/app deployment

Make the intended ownership of infra vs app resources explicit.

### Canonical configuration contract
Define one source-of-truth mapping model from:

- application env vars
- Docker `.env` or Compose-provided values
- Helm values / secrets / configmaps

Include a mapping table with:

- canonical variable / logical setting
- Docker source
- Helm source
- consuming component(s)
- default behavior
- secret vs non-secret classification

### Docker deployment design
Specify:

- required behavior of `deploy/scripts/docker/deploy-all.sh`
- whether build, infra startup, migration, and app startup are separate phases
- how readiness should be enforced before migration and before app startup
- how app-only mode should behave and what it assumes
- how to verify that components can talk to Postgres, RabbitMQ, MinIO, Redis, gRPC server, UI, and observability services

### Kubernetes/Helm deployment design
Specify:

- required behavior of `deploy/scripts/k8s/deploy-all.sh`
- role of `deploy/scripts/k8s/deploy-infra.sh`
- whether the app chart should remain self-contained, remain split, or support both with an explicit contract
- how self-contained and external-infra modes should resolve service names
- how values overlays should work across dev/prod
- how secrets should be provided
- how image references should be built/pushed/resolved for a new cluster
- how migrations should run and be gated

### Validation and health-check strategy
The spec must require non-mutating validation first:

- `docker compose -f docker-compose.yaml -f docker-compose.app.yaml config`
- `helm template` for the app chart
- `helm template` for the infra chart

Then define runtime validation for:

- full Docker deploy from scratch
- Docker app-only against existing infra
- fresh k8s cluster app deploy in self-contained mode
- fresh k8s cluster split infra/app deploy
- successful migrations before app startup
- grpc-server, workers, UI, DB, broker, MinIO, Redis, and observability wiring

Also include failure scenarios:

- missing env vars
- secret/value mismatch
- wrong namespace or service DNS
- infra not ready before migration or app startup

## Output Requirements

- Be decision complete.
- Do not leave key implementation choices unresolved.
- Prefer concrete repo-specific recommendations over broad alternatives.
- Call out any breaking changes.
- Separate required fixes from optional improvements.
- Include acceptance criteria that another engineer can implement and verify against.
- Include a prioritized backlog ordered by risk reduction and implementation dependency.

## Constraints

- Do not implement changes yet.
- Do not rewrite the whole platform conceptually.
- Stay close to the current repo structure unless a change is necessary to remove a clear correctness risk.
- Assume the deliverable of this task is the spec itself.
```

## Notes

- This prompt is intended to generate a spec, not perform the Docker/Helm fixes directly.
- It is intentionally grounded in the current repo layout and known inconsistencies so the resulting spec is actionable.
