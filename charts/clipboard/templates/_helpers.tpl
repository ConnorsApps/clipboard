{{- define "clipboard.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Don't change how names are generated: the PVC holds user data. */}}
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

{{/* Immutable on the Deployment. */}}
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
