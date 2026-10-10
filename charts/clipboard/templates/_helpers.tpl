{{- define "clipboard.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Base name for every object. Do not change how names are generated: the PVC holds
user data, and a different name on upgrade orphans it.
*/}}
{{- define "clipboard.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "clipboard.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Immutable on the Deployment: never add or change a label here. */}}
{{- define "clipboard.selectorLabels" -}}
app.kubernetes.io/name: {{ include "clipboard.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "clipboard.labels" -}}
helm.sh/chart: {{ include "clipboard.chart" . }}
{{ include "clipboard.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{- define "clipboard.secretName" -}}
{{- default (include "clipboard.fullname" .) .Values.existingSecret -}}
{{- end -}}

{{- define "clipboard.pvcName" -}}
{{- default (printf "%s-files" (include "clipboard.fullname" .)) .Values.persistence.existingClaim -}}
{{- end -}}

{{/*
Fail fast if values written for the old HULL-based chart (1.x) are still set.
Otherwise the `hull:` block is silently ignored and the release deploys without
the settings the user thinks they made.
*/}}
{{- define "clipboard.rejectHullValues" -}}
{{- if .Values.hull }}
{{- fail "values.hull.* is set, but this chart no longer uses HULL (vidispine/hull) and the `hull:` key has no effect. Move your settings to the new keys (config.passwords, config.mongodbUri, persistence.*, resources, ...); see \"Upgrading from 1.x\" in the README." -}}
{{- end -}}
{{- end -}}
