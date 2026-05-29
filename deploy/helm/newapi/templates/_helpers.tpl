{{/*
Expand the name of the chart.
*/}}
{{- define "new-api.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "new-api.fullname" -}}
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
{{- define "new-api.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "new-api.labels" -}}
helm.sh/chart: {{ include "new-api.chart" . }}
{{ include "new-api.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "new-api.selectorLabels" -}}
app.kubernetes.io/name: {{ include "new-api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Service account name.
*/}}
{{- define "new-api.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "new-api.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "new-api.secretName" -}}
{{- if .Values.secret.existingSecret }}
{{- .Values.secret.existingSecret }}
{{- else }}
{{- printf "%s-secret" (include "new-api.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "new-api.postgresqlName" -}}
{{- printf "%s-postgresql" (include "new-api.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "new-api.mysqlName" -}}
{{- printf "%s-mysql" (include "new-api.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "new-api.redisName" -}}
{{- printf "%s-redis" (include "new-api.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "new-api.databaseHost" -}}
{{- if eq .Values.database.type "postgresql" -}}
{{- default (include "new-api.postgresqlName" .) .Values.database.postgresql.host -}}
{{- else if eq .Values.database.type "mysql" -}}
{{- default (include "new-api.mysqlName" .) .Values.database.mysql.host -}}
{{- end -}}
{{- end }}

{{- define "new-api.redisHost" -}}
{{- default (include "new-api.redisName" .) .Values.redis.host -}}
{{- end }}

{{- define "new-api.sqlDsn" -}}
{{- if .Values.secret.sqlDsn -}}
{{- .Values.secret.sqlDsn -}}
{{- else if eq .Values.database.type "sqlite" -}}
{{- printf "local" -}}
{{- else if eq .Values.database.type "postgresql" -}}
{{- printf "postgresql://%s:%s@%s:%v/%s?sslmode=%s" .Values.database.postgresql.user .Values.database.postgresql.password (include "new-api.databaseHost" .) .Values.database.postgresql.port .Values.database.postgresql.database .Values.database.postgresql.sslMode -}}
{{- else if eq .Values.database.type "mysql" -}}
{{- printf "%s:%s@tcp(%s:%v)/%s" .Values.database.mysql.user .Values.database.mysql.password (include "new-api.databaseHost" .) .Values.database.mysql.port .Values.database.mysql.database -}}
{{- end -}}
{{- end }}

{{- define "new-api.redisConnString" -}}
{{- if .Values.secret.redisConnString -}}
{{- .Values.secret.redisConnString -}}
{{- else if .Values.redis.enabled -}}
{{- if .Values.redis.password -}}
{{- printf "redis://:%s@%s:%v/%v" .Values.redis.password (include "new-api.redisHost" .) .Values.redis.port .Values.redis.database -}}
{{- else -}}
{{- printf "redis://%s:%v/%v" (include "new-api.redisHost" .) .Values.redis.port .Values.redis.database -}}
{{- end -}}
{{- end -}}
{{- end }}

{{- define "new-api.dataPVCName" -}}
{{- if .Values.persistence.data.existingClaim }}
{{- .Values.persistence.data.existingClaim }}
{{- else }}
{{- printf "%s-data" (include "new-api.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "new-api.logsPVCName" -}}
{{- if .Values.persistence.logs.existingClaim }}
{{- .Values.persistence.logs.existingClaim }}
{{- else }}
{{- printf "%s-logs" (include "new-api.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "new-api.postgresqlPVCName" -}}
{{- if .Values.database.postgresql.persistence.existingClaim }}
{{- .Values.database.postgresql.persistence.existingClaim }}
{{- else }}
{{- printf "%s-postgresql" (include "new-api.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "new-api.mysqlPVCName" -}}
{{- if .Values.database.mysql.persistence.existingClaim }}
{{- .Values.database.mysql.persistence.existingClaim }}
{{- else }}
{{- printf "%s-mysql" (include "new-api.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "new-api.redisPVCName" -}}
{{- if .Values.redis.persistence.existingClaim }}
{{- .Values.redis.persistence.existingClaim }}
{{- else }}
{{- printf "%s-redis" (include "new-api.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
