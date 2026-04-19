# Multi-Region Affinity Plan (Fetch Workers)

## Scope

Реализуем nodeAffinity **только для fetch-worker**-ов при использовании `regions: [...]`.
Остальные компоненты (grpc-server, parser-worker, export-worker) не трогаем.
Docker/local dev не затрагивается — фича активируется только когда `regions` непустой и `scheduling.regionalNodeAffinity.enabled: true`.

## Как это работает

Пользователь лейблит ноды в k8s:
```bash
kubectl label node node-us topology.kubernetes.io/region=us-east
kubectl label node node-eu topology.kubernetes.io/region=eu-west
```

Задаёт регионы в values:
```yaml
fetchWorker:
  regions: ["us-east", "eu-west"]

scheduling:
  regionalNodeAffinity:
    enabled: true
    labelKey: topology.kubernetes.io/region
    mode: hard   # hard = requiredDuring... / soft = preferredDuring...
```

Helm генерирует два Deployment-а:
- `fetch-worker-us-east` → nodeAffinity → ноды с `topology.kubernetes.io/region=us-east`
- `fetch-worker-eu-west` → nodeAffinity → ноды с `topology.kubernetes.io/region=eu-west`

Если ноды с нужным лейблом нет → при `mode: hard` под не запустится (Pending), при `mode: soft` запустится на любой ноде.

## Шаги реализации

### Шаг 1: Добавить `scheduling` блок в `values.yaml`

Добавить после секции `podDisruptionBudget`:

```yaml
# =============================================================================
# Scheduling Configuration
# =============================================================================
scheduling:
  # nodeAffinity for regional fetch-worker deployments.
  # Only applies when fetchWorker.regions is non-empty.
  regionalNodeAffinity:
    enabled: false
    # Node label key to match against the region name.
    # Standard cloud label: topology.kubernetes.io/region
    # Custom label example: node-region
    labelKey: "topology.kubernetes.io/region"
    # hard: requiredDuringSchedulingIgnoredDuringExecution
    # soft: preferredDuringSchedulingIgnoredDuringExecution
    mode: "soft"
```

### Шаг 2: Обновить `templates/fetch-worker/deployment.yaml`

В секции `spec:` пода (там где уже есть `{{- with $root.Values.fetchWorker.affinity }}`) добавить логику:

```yaml
{{- if $root.Values.fetchWorker.affinity }}
affinity:
  {{- toYaml $root.Values.fetchWorker.affinity | nindent 8 }}
{{- else if and $region $root.Values.scheduling.regionalNodeAffinity.enabled }}
affinity:
  {{- include "distributed-crawler.regionalNodeAffinity" (dict "root" $root "region" $region) | nindent 8 }}
{{- end }}
```

Логика мержа:
- Явный `fetchWorker.affinity` → берётся как есть (backward compatible)
- Если пустой + есть `$region` + `regionalNodeAffinity.enabled: true` → инжектим nodeAffinity
- Иначе → ничего (текущее поведение)

### Шаг 3: Добавить helper в `templates/_helpers.tpl`

```yaml
{{/*
nodeAffinity for a regional fetch-worker deployment.
Args: dict "root" . "region" $region
*/}}
{{- define "distributed-crawler.regionalNodeAffinity" -}}
{{- $root := .root -}}
{{- $region := .region -}}
nodeAffinity:
  {{- if eq $root.Values.scheduling.regionalNodeAffinity.mode "hard" }}
  requiredDuringSchedulingIgnoredDuringExecution:
    nodeSelectorTerms:
      - matchExpressions:
          - key: {{ $root.Values.scheduling.regionalNodeAffinity.labelKey }}
            operator: In
            values:
              - {{ $region }}
  {{- else }}
  preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      preference:
        matchExpressions:
          - key: {{ $root.Values.scheduling.regionalNodeAffinity.labelKey }}
            operator: In
            values:
              - {{ $region }}
  {{- end }}
{{- end }}
```

### Шаг 4: Добавить пример в `values-prod.yaml`

Добавить закомментированный пример в конец файла:

```yaml
# Multi-region fetch workers example.
# Requires nodes labeled with topology.kubernetes.io/region=<region>.
#
# fetchWorker:
#   regions: ["us-east", "eu-west"]
#
# scheduling:
#   regionalNodeAffinity:
#     enabled: true
#     labelKey: topology.kubernetes.io/region
#     mode: hard
```

## Что НЕ меняем

- `parser-worker`, `grpc-server`, `export-worker` — без изменений
- `docker-compose` — без изменений
- `affinity: {}` в values для всех компонентов остаётся (ручной override)
- HPA, PDB — без изменений

## Проверка

После реализации запустить `helm template` и убедиться:

```bash
# Без регионов — affinity отсутствует (как сейчас)
helm template . --set fetchWorker.regions="{}" | grep -A5 affinity

# С регионами, soft mode
helm template . \
  --set "fetchWorker.regions={us-east,eu-west}" \
  --set scheduling.regionalNodeAffinity.enabled=true \
  --set scheduling.regionalNodeAffinity.mode=soft \
  | grep -A 15 nodeAffinity

# С регионами, hard mode
helm template . \
  --set "fetchWorker.regions={us-east,eu-west}" \
  --set scheduling.regionalNodeAffinity.enabled=true \
  --set scheduling.regionalNodeAffinity.mode=hard \
  | grep -A 15 nodeAffinity

# Явный affinity override должен перекрыть авто-инжект
helm template . \
  --set "fetchWorker.regions={us-east}" \
  --set scheduling.regionalNodeAffinity.enabled=true \
  --set-json 'fetchWorker.affinity={"nodeAffinity": {"requiredDuringSchedulingIgnoredDuringExecution": {}}}' \
  | grep -A 15 affinity
```
