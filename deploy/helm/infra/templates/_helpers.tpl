{{/*
Expand the name of the chart.
*/}}
{{- define "distributed-crawler-infra.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified release name.
*/}}
{{- define "distributed-crawler-infra.fullname" -}}
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
Common labels
*/}}
{{- define "distributed-crawler-infra.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/name: {{ include "distributed-crawler-infra.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Namespace to use
*/}}
{{- define "distributed-crawler-infra.namespace" -}}
{{- if .Values.namespaceOverride }}
{{- .Values.namespaceOverride }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Compute the PostgreSQL service host for this release.
Exposes this release's PostgreSQL so the app chart's values-external-infra.yaml
can reference it without hard-coding the release name.
*/}}
{{- define "distributed-crawler-infra.postgresHost" -}}
{{- printf "%s-postgresql" .Release.Name }}
{{- end }}

{{/*
RabbitMQ host
*/}}
{{- define "distributed-crawler-infra.rabbitmqHost" -}}
{{- printf "%s-rabbitmq" .Release.Name }}
{{- end }}

{{/*
MinIO endpoint (host:port)
*/}}
{{- define "distributed-crawler-infra.minioEndpoint" -}}
{{- printf "%s-minio:9000" .Release.Name }}
{{- end }}

{{/*
Redis master host
*/}}
{{- define "distributed-crawler-infra.redisHost" -}}
{{- printf "%s-redis-master" .Release.Name }}
{{- end }}

{{/*
OTel collector service host
*/}}
{{- define "distributed-crawler-infra.otelCollectorHost" -}}
{{- printf "%s-otelcollector" .Release.Name }}
{{- end }}

{{/*
Jaeger all-in-one service host
*/}}
{{- define "distributed-crawler-infra.jaegerHost" -}}
{{- printf "%s-jaeger-all-in-one" .Release.Name }}
{{- end }}
