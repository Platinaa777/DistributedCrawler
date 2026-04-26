# Запуск Distributed Crawler на нескольких нодах Kubernetes

Этот документ описывает, как запускать `Distributed Crawler` в multi-node кластере Kubernetes так, чтобы региональные `fetch-worker` размещались на нужных нодах.

Документ ориентирован на сценарий, где:

- `fetch-worker` запускаются отдельно для каждого региона через `fetchWorker.regions`
- привязка к нодам задаётся через `scheduling.regionalNodeAffinity`
- остальные компоненты (`grpc-server`, `parser-worker`, `export-worker`, `ui`) работают как обычно и не требуют региональной привязки

## Как это работает

Если в Helm values указать:

```yaml
fetchWorker:
  regions:
    - us-east
    - eu-west
```

чарт создаст отдельный `Deployment` для каждого региона:

- `fetch-worker-us-east`
- `fetch-worker-eu-west`

Для каждого такого deployment автоматически выставляется переменная окружения:

```text
WORKER_REGION=<region>
```

Если дополнительно включить:

```yaml
scheduling:
  regionalNodeAffinity:
    enabled: true
```

то каждый региональный `fetch-worker` будет стараться запускаться только на нодах с подходящим label.

## Когда это нужно

Используйте этот режим, если:

- в кластере несколько нод или node pool'ов
- часть `fetch-worker` должна обслуживать конкретный регион
- вы хотите размещать `fetch-worker-us-east` и `fetch-worker-eu-west` на разных нодах
- нужно избежать случайного запуска регионального воркера на любой свободной ноде

## Требования

Перед запуском убедитесь, что:

- у вас есть Kubernetes-кластер с несколькими worker-нодами
- `kubectl` подключён к нужному кластеру
- `helm` установлен
- Docker-образы приложения уже доступны или могут быть собраны launcher-скриптом
- названия регионов в `fetchWorker.regions` совпадают со значениями label на нодах

## Шаг 1. Проверить ноды

Сначала посмотрите список нод:

```bash
kubectl get nodes -o wide
```

Полезно сразу вывести существующие labels:

```bash
kubectl get nodes --show-labels
```

## Шаг 2. Разметить ноды по регионам

По умолчанию документация и values используют label:

```text
topology.kubernetes.io/region
```

Пример:

```bash
kubectl label node worker-us-1 topology.kubernetes.io/region=us-east
kubectl label node worker-us-2 topology.kubernetes.io/region=us-east
kubectl label node worker-eu-1 topology.kubernetes.io/region=eu-west
```

Проверка:

```bash
kubectl get nodes -L topology.kubernetes.io/region
```

Если вы используете свой label, например `node-region`, это тоже поддерживается. Тогда его нужно будет указать в `scheduling.regionalNodeAffinity.labelKey`.

## Шаг 3. Подготовить values-файл

Минимальный пример для multi-node запуска:

```yaml
fetchWorker:
  replicaCount: 2
  regions:
    - us-east
    - eu-west

scheduling:
  regionalNodeAffinity:
    enabled: true
    labelKey: topology.kubernetes.io/region
    mode: hard
```

Что означает этот конфиг:

- для каждого региона будет создан отдельный `fetch-worker` deployment
- каждый deployment получит `WORKER_REGION`, равный имени региона
- `mode: hard` запрещает запуск пода на ноде без нужного label

### Разница между `hard` и `soft`

`mode: hard`:

- использует `requiredDuringSchedulingIgnoredDuringExecution`
- если подходящей ноды нет, pod останется в `Pending`

`mode: soft`:

- использует `preferredDuringSchedulingIgnoredDuringExecution`
- Kubernetes постарается выбрать нужную ноду, но при отсутствии такой ноды может запустить pod в другом месте

Для production multi-node сценария обычно безопаснее использовать `hard`.

## Шаг 4. Запуск через launcher-скрипт

Самый удобный способ запуска:

```bash
./deploy/scripts/multi_region_run.sh \
  --regions us-east,eu-west \
  --mode k8s \
  --port-forward
```

Если нужно включить дополнительные параметры:

```bash
./deploy/scripts/multi_region_run.sh \
  --regions us-east,eu-west \
  --mode k8s \
  --tag latest \
  --port-forward \
  -- \
  --app-values /absolute/path/to/multinode-values.yaml
```

Что делает скрипт:

- разворачивает приложение через Helm
- передаёт `fetchWorker.regions`
- настраивает список crawl queues для регионов
- создаёт отдельные deployment'ы `fetch-worker` по регионам

Если у вас уже есть свой values-файл, удобнее передавать его через аргументы Helm после `--`.

## Шаг 5. Прямой запуск через Helm

Если нужен ручной деплой без launcher-скрипта, используйте `helm upgrade --install`.

Пример:

```bash
helm upgrade --install distributed-crawler ./deploy/helm/distributed-crawler \
  -n distributed-crawler \
  --create-namespace \
  -f ./deploy/helm/distributed-crawler/values.yaml \
  -f ./multinode-values.yaml
```

Пример файла `multinode-values.yaml`:

```yaml
fetchWorker:
  enabled: true
  replicaCount: 2
  regions:
    - us-east
    - eu-west

scheduling:
  regionalNodeAffinity:
    enabled: true
    labelKey: topology.kubernetes.io/region
    mode: hard
```

## Шаг 6. Проверить результат

Проверьте, что deployment'ы создались:

```bash
kubectl get deploy -n distributed-crawler
```

Ожидаемо вы увидите что-то близкое к:

```text
fetch-worker-us-east
fetch-worker-eu-west
parser-worker
grpc-server
export-worker
ui
```

Проверьте, на каких нодах реально запущены pod'ы:

```bash
kubectl get pods -n distributed-crawler -o wide
```

Проверьте affinity в сгенерированном манифесте:

```bash
helm template distributed-crawler ./deploy/helm/distributed-crawler \
  -f ./multinode-values.yaml | grep -A 20 nodeAffinity
```

Если хотите убедиться для конкретного pod:

```bash
kubectl describe pod <fetch-worker-pod-name> -n distributed-crawler
```

В выводе должны быть видны:

- `WORKER_REGION=us-east` или `WORKER_REGION=eu-west`
- `Node:` с нужной нодой
- `Affinity:` с вашим `labelKey`

## Готовый пример для двух регионов

```yaml
fetchWorker:
  replicaCount: 3
  regions:
    - us-east
    - eu-west

scheduling:
  regionalNodeAffinity:
    enabled: true
    labelKey: topology.kubernetes.io/region
    mode: hard
```

При таком конфиге:

- создастся `fetch-worker-us-east` с `3` репликами
- создастся `fetch-worker-eu-west` с `3` репликами
- `us-east` реплики будут планироваться на ноды с `topology.kubernetes.io/region=us-east`
- `eu-west` реплики будут планироваться на ноды с `topology.kubernetes.io/region=eu-west`

## Если нужен свой label вместо `topology.kubernetes.io/region`

Пример:

```yaml
fetchWorker:
  regions:
    - moscow
    - frankfurt

scheduling:
  regionalNodeAffinity:
    enabled: true
    labelKey: node-region
    mode: hard
```

Тогда ноды должны быть размечены так:

```bash
kubectl label node worker-1 node-region=moscow
kubectl label node worker-2 node-region=frankfurt
```

## Типовые проблемы

### Pod завис в `Pending`

Чаще всего причина одна из этих:

- на нодах нет нужного label
- значение в `fetchWorker.regions` не совпадает со значением label
- включён `mode: hard`, но подходящей ноды нет
- на целевой ноде не хватает CPU или памяти

Полезная команда:

```bash
kubectl describe pod <fetch-worker-pod-name> -n distributed-crawler
```

### Воркеры создались, но запустились не на тех нодах

Проверьте:

- включён ли `scheduling.regionalNodeAffinity.enabled: true`
- не используется ли `mode: soft`
- не задан ли вручную `fetchWorker.affinity`, который перекрывает авто-generated affinity

Важно: если в values явно указан `fetchWorker.affinity`, он имеет приоритет над автоматической региональной привязкой.

### Регион есть в values, но deployment не появился

Проверьте итоговый values и шаблоны:

```bash
helm template distributed-crawler ./deploy/helm/distributed-crawler \
  --set "fetchWorker.regions={us-east,eu-west}" \
  --set scheduling.regionalNodeAffinity.enabled=true
```

## Рекомендуемый production-подход

Для multi-node кластера обычно достаточно следующей схемы:

- разметить ноды через `topology.kubernetes.io/region`
- задать `fetchWorker.regions` списком регионов
- включить `scheduling.regionalNodeAffinity.enabled: true`
- использовать `mode: hard`
- проверять размещение через `kubectl get pods -o wide`

Этого достаточно, чтобы региональные `fetch-worker` стабильно запускались на нужных нодах без ручной привязки каждого deployment.
