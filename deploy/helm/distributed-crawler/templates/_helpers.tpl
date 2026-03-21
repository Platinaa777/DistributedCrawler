{{/*
Expand the name of the chart.
*/}}
{{- define "distributed-crawler.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "distributed-crawler.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "distributed-crawler.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "distributed-crawler.labels" -}}
helm.sh/chart: {{ include "distributed-crawler.chart" . }}
{{ include "distributed-crawler.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "distributed-crawler.selectorLabels" -}}
app.kubernetes.io/name: {{ include "distributed-crawler.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "distributed-crawler.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "distributed-crawler.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the proper image name
*/}}
{{- define "distributed-crawler.image" -}}
{{- $registryName := .imageRoot.registry | default .global.imageRegistry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $tag := .imageRoot.tag | default "latest" | toString -}}
{{- if $registryName }}
{{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else }}
{{- printf "%s:%s" $repositoryName $tag -}}
{{- end }}
{{- end }}

{{/*
Return the secret name for the application
*/}}
{{- define "distributed-crawler.secretName" -}}
{{- if .Values.secrets.existingSecret }}
{{- .Values.secrets.existingSecret }}
{{- else }}
{{- include "distributed-crawler.fullname" . }}-secrets
{{- end }}
{{- end }}

{{/*
Return the configmap name for the application
*/}}
{{- define "distributed-crawler.configMapName" -}}
{{- include "distributed-crawler.fullname" . }}-config
{{- end }}

{{/*
Infrastructure release name
*/}}
{{- define "distributed-crawler.infraReleaseName" -}}
{{- if .Values.infra.releaseName }}
{{- .Values.infra.releaseName }}
{{- else }}
{{- .Release.Name }}
{{- end }}
{{- end }}

{{/*
Infrastructure namespace
*/}}
{{- define "distributed-crawler.infraNamespace" -}}
{{- if .Values.infra.namespace }}
{{- .Values.infra.namespace }}
{{- else }}
{{- include "distributed-crawler.namespace" . }}
{{- end }}
{{- end }}

{{/*
Build a service FQDN for the configured infra release
*/}}
{{- define "distributed-crawler.infraServiceFQDN" -}}
{{- printf "%s.%s.svc.cluster.local" .service (include "distributed-crawler.infraNamespace" .root) -}}
{{- end }}

{{/*
PostgreSQL host
*/}}
{{- define "distributed-crawler.postgresHost" -}}
{{- if .Values.config.postgres.host }}
{{- .Values.config.postgres.host }}
{{- else }}
{{- include "distributed-crawler.infraServiceFQDN" (dict "root" . "service" (printf "%s-postgresql" (include "distributed-crawler.infraReleaseName" .))) }}
{{- end }}
{{- end }}

{{/*
RabbitMQ host
*/}}
{{- define "distributed-crawler.rabbitmqHost" -}}
{{- if .Values.config.rabbitmq.host }}
{{- .Values.config.rabbitmq.host }}
{{- else }}
{{- include "distributed-crawler.infraServiceFQDN" (dict "root" . "service" (printf "%s-rabbitmq" (include "distributed-crawler.infraReleaseName" .))) }}
{{- end }}
{{- end }}

{{/*
MinIO endpoint (host:port)
*/}}
{{- define "distributed-crawler.minioHost" -}}
{{- if .Values.config.minio.endpoint }}
{{- .Values.config.minio.endpoint }}
{{- else }}
{{- include "distributed-crawler.infraServiceFQDN" (dict "root" . "service" (printf "%s-minio" (include "distributed-crawler.infraReleaseName" .))) }}
{{- end }}
{{- end }}

{{/*
MinIO endpoint (host:port)
*/}}
{{- define "distributed-crawler.minioEndpoint" -}}
{{- if .Values.config.minio.endpoint }}
{{- if .Values.config.minio.port }}
{{- printf "%s:%d" .Values.config.minio.endpoint (int .Values.config.minio.port) }}
{{- else }}
{{- .Values.config.minio.endpoint }}
{{- end }}
{{- else }}
{{- printf "%s:%d" (include "distributed-crawler.minioHost" .) (int .Values.config.minio.port) }}
{{- end }}
{{- end }}

{{/*
Redis host
*/}}
{{- define "distributed-crawler.redisHost" -}}
{{- if .Values.config.redis.host }}
{{- .Values.config.redis.host }}
{{- else }}
{{- include "distributed-crawler.infraServiceFQDN" (dict "root" . "service" (printf "%s-redis-master" (include "distributed-crawler.infraReleaseName" .))) }}
{{- end }}
{{- end }}

{{/*
OTel collector endpoint
*/}}
{{- define "distributed-crawler.otelEndpoint" -}}
{{- if .Values.config.otel.exporterEndpoint }}
{{- .Values.config.otel.exporterEndpoint }}
{{- else }}
{{- printf "%s:%d" (include "distributed-crawler.infraServiceFQDN" (dict "root" . "service" (printf "%s-otelcollector" (include "distributed-crawler.infraReleaseName" .))) ) 4317 }}
{{- end }}
{{- end }}

{{/*
OpenSearch endpoint
*/}}
{{- define "distributed-crawler.opensearchEndpoint" -}}
{{- if .Values.config.opensearch.endpoint }}
{{- .Values.config.opensearch.endpoint }}
{{- else }}
{{- printf "http://%s:%d" (include "distributed-crawler.infraServiceFQDN" (dict "root" . "service" (printf "%s-opensearch" (include "distributed-crawler.infraReleaseName" .))) ) 9200 }}
{{- end }}
{{- end }}

{{/*
gRPC Server name
*/}}
{{- define "distributed-crawler.grpcServer.fullname" -}}
{{- printf "%s-grpc-server" (include "distributed-crawler.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
gRPC Server host used by workers and the UI
*/}}
{{- define "distributed-crawler.grpcServer.host" -}}
{{- if .Values.grpcServer.hostOverride }}
{{- .Values.grpcServer.hostOverride }}
{{- else }}
{{- include "distributed-crawler.grpcServer.fullname" . }}
{{- end }}
{{- end }}

{{/*
gRPC Server labels
*/}}
{{- define "distributed-crawler.grpcServer.labels" -}}
{{ include "distributed-crawler.labels" . }}
app.kubernetes.io/component: grpc-server
{{- end }}

{{/*
gRPC Server selector labels
*/}}
{{- define "distributed-crawler.grpcServer.selectorLabels" -}}
{{ include "distributed-crawler.selectorLabels" . }}
app.kubernetes.io/component: grpc-server
{{- end }}

{{/*
Fetch Worker name
*/}}
{{- define "distributed-crawler.fetchWorker.fullname" -}}
{{- printf "%s-fetch-worker" (include "distributed-crawler.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fetch Worker labels
*/}}
{{- define "distributed-crawler.fetchWorker.labels" -}}
{{ include "distributed-crawler.labels" . }}
app.kubernetes.io/component: fetch-worker
{{- end }}

{{/*
Fetch Worker selector labels
*/}}
{{- define "distributed-crawler.fetchWorker.selectorLabels" -}}
{{ include "distributed-crawler.selectorLabels" . }}
app.kubernetes.io/component: fetch-worker
{{- end }}

{{/*
Parser Worker name
*/}}
{{- define "distributed-crawler.parserWorker.fullname" -}}
{{- printf "%s-parser-worker" (include "distributed-crawler.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Parser Worker labels
*/}}
{{- define "distributed-crawler.parserWorker.labels" -}}
{{ include "distributed-crawler.labels" . }}
app.kubernetes.io/component: parser-worker
{{- end }}

{{/*
Parser Worker selector labels
*/}}
{{- define "distributed-crawler.parserWorker.selectorLabels" -}}
{{ include "distributed-crawler.selectorLabels" . }}
app.kubernetes.io/component: parser-worker
{{- end }}

{{/*
Export Worker name
*/}}
{{- define "distributed-crawler.exportWorker.fullname" -}}
{{- printf "%s-export-worker" (include "distributed-crawler.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Export Worker labels
*/}}
{{- define "distributed-crawler.exportWorker.labels" -}}
{{ include "distributed-crawler.labels" . }}
app.kubernetes.io/component: export-worker
{{- end }}

{{/*
Export Worker selector labels
*/}}
{{- define "distributed-crawler.exportWorker.selectorLabels" -}}
{{ include "distributed-crawler.selectorLabels" . }}
app.kubernetes.io/component: export-worker
{{- end }}

{{/*
Queue secrets volume (mounts queue-secrets.json from the app Secret)
*/}}
{{- define "distributed-crawler.queueSecretsVolume" -}}
- name: queue-secrets
  secret:
    secretName: {{ include "distributed-crawler.secretName" . }}
    items:
      - key: queue-secrets.json
        path: queue-secrets.json
{{- end }}

{{/*
Queue secrets volumeMount
*/}}
{{- define "distributed-crawler.queueSecretsVolumeMount" -}}
- name: queue-secrets
  mountPath: /etc/crawler
  readOnly: true
{{- end }}

{{/*
Wait for a TCP dependency before starting a container
*/}}
{{- define "distributed-crawler.waitForService" -}}
- name: wait-for-{{ .name }}
  image: "{{ .root.Values.dependencyWaiter.image.repository }}:{{ .root.Values.dependencyWaiter.image.tag }}"
  imagePullPolicy: {{ .root.Values.dependencyWaiter.image.pullPolicy }}
  command:
    - sh
    - -ec
    - |
      until nc -z -w 2 {{ .host | quote }} {{ .port }}; do
        echo "waiting for {{ .name }} at {{ .host }}:{{ .port }}"
        sleep {{ .root.Values.dependencyWaiter.pollIntervalSeconds }}
      done
{{- end }}

{{/*
UI name
*/}}
{{- define "distributed-crawler.ui.fullname" -}}
{{- printf "%s-ui" (include "distributed-crawler.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
UI labels
*/}}
{{- define "distributed-crawler.ui.labels" -}}
{{ include "distributed-crawler.labels" . }}
app.kubernetes.io/component: ui
{{- end }}

{{/*
UI selector labels
*/}}
{{- define "distributed-crawler.ui.selectorLabels" -}}
{{ include "distributed-crawler.selectorLabels" . }}
app.kubernetes.io/component: ui
{{- end }}

{{/*
Namespace to use
*/}}
{{- define "distributed-crawler.namespace" -}}
{{- if .Values.namespaceOverride }}
{{- .Values.namespaceOverride }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}
