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
PostgreSQL host
*/}}
{{- define "distributed-crawler.postgresHost" -}}
{{- if .Values.config.postgres.host }}
{{- .Values.config.postgres.host }}
{{- else if .Values.postgresql.enabled }}
{{- printf "%s-postgresql" .Release.Name }}
{{- else }}
{{- fail "PostgreSQL host must be specified when postgresql.enabled is false" }}
{{- end }}
{{- end }}

{{/*
RabbitMQ host
*/}}
{{- define "distributed-crawler.rabbitmqHost" -}}
{{- if .Values.config.rabbitmq.host }}
{{- .Values.config.rabbitmq.host }}
{{- else if .Values.rabbitmq.enabled }}
{{- printf "%s-rabbitmq" .Release.Name }}
{{- else }}
{{- fail "RabbitMQ host must be specified when rabbitmq.enabled is false" }}
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
{{- else if .Values.minio.enabled }}
{{- printf "%s-minio:%d" .Release.Name (int .Values.config.minio.port) }}
{{- else }}
{{- fail "MinIO endpoint must be specified when minio.enabled is false" }}
{{- end }}
{{- end }}

{{/*
Redis host
*/}}
{{- define "distributed-crawler.redisHost" -}}
{{- if .Values.config.redis.host }}
{{- .Values.config.redis.host }}
{{- else if .Values.redis.enabled }}
{{- printf "%s-redis-master" .Release.Name }}
{{- else }}
{{- fail "Redis host must be specified when redis.enabled is false" }}
{{- end }}
{{- end }}

{{/*
gRPC Server name
*/}}
{{- define "distributed-crawler.grpcServer.fullname" -}}
{{- printf "%s-grpc-server" (include "distributed-crawler.fullname" .) | trunc 63 | trimSuffix "-" }}
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
