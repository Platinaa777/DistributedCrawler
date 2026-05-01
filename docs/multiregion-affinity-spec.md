# Multi-Region Node Affinity Specification

Эта спека описывает, как запускать Distributed Crawler в multi-node Kubernetes кластере так, чтобы региональные `fetch-worker` размещались на нужных нодах.

Подход нужен, когда crawl-трафик должен выходить из конкретного региона, node pool или группы нод. Например, `fetch-worker-us-east` должен работать только на нодах с label `topology.kubernetes.io/region=us-east`, а `fetch-worker-eu-west` — на `eu-west`.

## Как Работает Механика

В Helm chart за региональные fetch-worker отвечает список:

```yaml
fetchWorker:
  regions:
    - us-east
    - eu-west
```

Для каждого региона chart создаёт отдельный `Deployment`:

```text
fetch-worker-us-east
fetch-worker-eu-west
```

В каждый deployment автоматически добавляется:

```text
WORKER_REGION=<region>
```

Fetch-worker с `WORKER_REGION` читает две crawl queues:

- default queue из `RABBITMQ_CRAWL_QUEUE_NAME`, например `crawl_queue`
- региональную queue `<RABBITMQ_CRAWL_QUEUE_NAME>_<region>`, например `crawl_queue_us-east`

Node affinity включается отдельно:

```yaml
scheduling:
  regionalNodeAffinity:
    enabled: true
```

Эта настройка применяется к региональным `fetch-worker` deployments. Остальные компоненты (`grpc-server`, `parser-worker`, `export-worker`, `ui`) запускаются без региональной привязки, если вы явно не зададите им собственные `nodeSelector`, `affinity` или `tolerations`.

## Когда Использовать

Используйте regional node affinity, если:

- в кластере есть несколько worker-нод или node pool'ов
- fetch-трафик должен идти из определённого региона
- разные региональные fetch-worker должны жить на разных нодах
- важно исключить случайный запуск регионального воркера на неподходящей ноде
- вы настраиваете `queue_weights` и хотите, чтобы выбранные очереди реально обслуживались воркерами нужного региона

## Требования

Перед настройкой проверьте, что:

- `kubectl` подключён к нужному кластеру
- установлен Helm 3
- Docker-образы приложения доступны кластеру или собираются launcher-скриптом
- в кластере есть ноды для каждого региона
- значения в `fetchWorker.regions` совпадают со значениями labels на нодах
- API получает полный список crawl queues через `RABBITMQ_CRAWL_QUEUE_NAMES`

## 1. Проверить Ноды

Посмотрите список нод:

```bash
kubectl get nodes -o wide
```

Проверьте текущие labels:

```bash
kubectl get nodes --show-labels
```

Для более удобного вывода по региону:

```bash
kubectl get nodes -L topology.kubernetes.io/region
```

## 2. Разметить Ноды

По умолчанию спека использует стандартный Kubernetes/cloud label:

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

Если в вашем кластере используется другой label, например `node-region`, укажите его в `scheduling.regionalNodeAffinity.labelKey`.

## 3. Подготовить Values

Минимальный values-файл для двух регионов:

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

Что произойдёт:

- chart создаст отдельный fetch-worker deployment для каждого региона
- каждый pod получит `WORKER_REGION`
- для каждого регионального deployment будет сгенерирован `nodeAffinity`
- при `mode: hard` pod останется `Pending`, если подходящей ноды нет

## 4. Выбрать Hard Или Soft Affinity

`mode: hard` использует `requiredDuringSchedulingIgnoredDuringExecution`.

```yaml
scheduling:
  regionalNodeAffinity:
    enabled: true
    mode: hard
```

Выбирайте `hard`, если региональная привязка обязательна. Это лучший вариант для production, где запуск в неправильном регионе хуже, чем `Pending` pod.

`mode: soft` использует `preferredDuringSchedulingIgnoredDuringExecution`.

```yaml
scheduling:
  regionalNodeAffinity:
    enabled: true
    mode: soft
```

Выбирайте `soft`, если regional placement желателен, но воркер может временно стартовать на другой ноде.

## 5. Запустить Через Launcher

Самый удобный путь:

```bash
./deploy/scripts/multi_region_run.sh \
  --regions us-east,eu-west \
  --mode k8s \
  --port-forward
```

С дополнительным values-файлом:

```bash
./deploy/scripts/multi_region_run.sh \
  --regions us-east,eu-west \
  --mode k8s \
  --tag latest \
  --port-forward \
  -- \
  --app-values-file /absolute/path/to/multinode-values.yaml
```

Launcher:

- передаёт `fetchWorker.regions` в Helm
- формирует `RABBITMQ_CRAWL_QUEUE_NAMES`
- разворачивает приложение через app Helm chart
- создаёт отдельные `fetch-worker` deployments по регионам

## 6. Запустить Напрямую Через Helm

Если launcher не нужен, используйте `helm upgrade --install`:

```bash
helm upgrade --install distributed-crawler ./deploy/helm/distributed-crawler \
  -n distributed-crawler \
  --create-namespace \
  -f ./deploy/helm/distributed-crawler/values.yaml \
  -f ./multinode-values.yaml
```

Пример `multinode-values.yaml`:

```yaml
fetchWorker:
  enabled: true
  replicaCount: 2
  regions:
    - us-east
    - eu-west

config:
  rabbitmq:
    crawlQueueName: crawl_queue
    crawlQueueNames: crawl_queue,crawl_queue_us-east,crawl_queue_eu-west

scheduling:
  regionalNodeAffinity:
    enabled: true
    labelKey: topology.kubernetes.io/region
    mode: hard
```

Если вы запускаете напрямую через Helm, не забудьте явно передать полный список crawl queues. Launcher делает это автоматически, а ручной Helm-запуск — нет.

## 7. Проверить Deployment'ы

Проверьте, что deployments созданы:

```bash
kubectl get deploy -n distributed-crawler
```

Ожидаемый результат:

```text
fetch-worker-us-east
fetch-worker-eu-west
parser-worker
grpc-server
export-worker
ui
```

Проверьте размещение pod'ов:

```bash
kubectl get pods -n distributed-crawler -o wide
```

Проверьте переменные окружения и affinity у конкретного pod:

```bash
kubectl describe pod <fetch-worker-pod-name> -n distributed-crawler
```

В выводе должны быть:

- `WORKER_REGION=us-east` или `WORKER_REGION=eu-west`
- `Node:` с подходящей нодой
- `Affinity:` с выбранным `labelKey`

## 8. Проверить Сгенерированный Manifest

Перед применением можно проверить chart:

```bash
helm template distributed-crawler ./deploy/helm/distributed-crawler \
  -f ./deploy/helm/distributed-crawler/values.yaml \
  -f ./multinode-values.yaml
```

Быстрая проверка affinity:

```bash
helm template distributed-crawler ./deploy/helm/distributed-crawler \
  -f ./deploy/helm/distributed-crawler/values.yaml \
  -f ./multinode-values.yaml | grep -A 20 nodeAffinity
```

Для `hard` режима должен появиться блок `requiredDuringSchedulingIgnoredDuringExecution`, для `soft` — `preferredDuringSchedulingIgnoredDuringExecution`.

## 9. Проверить Очереди

API должен видеть default и региональные crawl queues:

```bash
curl http://localhost:8084/api/v1/crawl-queues
```

Ожидаемый пример:

```json
{
  "queues": [
    "crawl_queue",
    "crawl_queue_us-east",
    "crawl_queue_eu-west"
  ]
}
```

Если список содержит только default queue, проверьте `RABBITMQ_CRAWL_QUEUE_NAMES` или `config.rabbitmq.crawlQueueNames`.

## Готовый Пример

```yaml
fetchWorker:
  enabled: true
  replicaCount: 3
  regions:
    - us-east
    - eu-west

config:
  rabbitmq:
    crawlQueueName: crawl_queue
    crawlQueueNames: crawl_queue,crawl_queue_us-east,crawl_queue_eu-west

scheduling:
  regionalNodeAffinity:
    enabled: true
    labelKey: topology.kubernetes.io/region
    mode: hard
```

## Troubleshooting

`fetch-worker-*` pod остаётся `Pending`:

- проверьте labels: `kubectl get nodes -L topology.kubernetes.io/region`
- проверьте, что значение региона в `fetchWorker.regions` совпадает с label
- если вы используете `mode: hard`, временно переключитесь на `soft`, чтобы подтвердить, что проблема именно в labels/capacity

Pod запустился не в том регионе:

- проверьте, что `scheduling.regionalNodeAffinity.enabled=true`
- проверьте `labelKey`
- убедитесь, что у `fetchWorker.affinity` не задан собственный affinity: он имеет приоритет над автоматическим regional affinity

UI не показывает региональные очереди:

- проверьте `GET /api/v1/crawl-queues`
- при launcher-запуске проверьте лог `Crawl queues: ...`
- при ручном Helm-запуске задайте `config.rabbitmq.crawlQueueNames`

Очередь есть, но региональный worker её не читает:

- проверьте `WORKER_REGION` в `kubectl describe pod`
- проверьте значение `RABBITMQ_CRAWL_QUEUE_NAME`
- имя региональной очереди строится как `<RABBITMQ_CRAWL_QUEUE_NAME>_<WORKER_REGION>`

Parser-worker не привязан к региону:

- это ожидаемое поведение для основного сценария: parser-worker читает общую parsing queue
- если parser-worker тоже нужно закрепить за нодами, задайте `parserWorker.nodeSelector`, `parserWorker.affinity` или `parserWorker.tolerations` вручную
