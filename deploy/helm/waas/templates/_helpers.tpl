{{/*
Expand the name of the chart.
*/}}
{{- define "waas.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "waas.fullname" -}}
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
{{- define "waas.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "waas.labels" -}}
helm.sh/chart: {{ include "waas.chart" . }}
{{ include "waas.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "waas.selectorLabels" -}}
app.kubernetes.io/name: {{ include "waas.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Service account name
*/}}
{{- define "waas.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "waas.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Namespace
*/}}
{{- define "waas.namespace" -}}
{{ .Release.Namespace }}
{{- end }}

{{/*
PostgreSQL host — in-cluster service name or external host
*/}}
{{- define "waas.postgresql.host" -}}
{{- if .Values.postgresql.enabled }}
{{- printf "%s-postgresql" (include "waas.fullname" .) }}
{{- else }}
{{- .Values.postgresql.host }}
{{- end }}
{{- end }}

{{/*
Redis host — in-cluster service name or external host
*/}}
{{- define "waas.redis.host" -}}
{{- if .Values.redis.enabled }}
{{- printf "%s-redis" (include "waas.fullname" .) }}
{{- else }}
{{- .Values.redis.host }}
{{- end }}
{{- end }}

{{/*
Database URL
*/}}
{{- define "waas.databaseUrl" -}}
{{- printf "postgres://%s:%s@%s:%v/%s?sslmode=require" .Values.postgresql.username .Values.postgresql.password (include "waas.postgresql.host" .) (int .Values.postgresql.port) .Values.postgresql.database }}
{{- end }}

{{/*
Redis URL
*/}}
{{- define "waas.redisUrl" -}}
{{- if .Values.redis.password }}
{{- printf "redis://:%s@%s:%v" .Values.redis.password (include "waas.redis.host" .) (int .Values.redis.port) }}
{{- else }}
{{- printf "redis://%s:%v" (include "waas.redis.host" .) (int .Values.redis.port) }}
{{- end }}
{{- end }}

{{/*
Security context for application pods
*/}}
{{- define "waas.podSecurityContext" -}}
runAsNonRoot: true
runAsUser: 65532
fsGroup: 65532
{{- end }}

{{/*
Security context for application containers
*/}}
{{- define "waas.containerSecurityContext" -}}
allowPrivilegeEscalation: false
readOnlyRootFilesystem: true
runAsNonRoot: true
runAsUser: 65532
capabilities:
  drop:
  - ALL
{{- end }}
