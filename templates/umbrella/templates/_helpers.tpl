{{- define "umbrella-chart.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Selector labels: the immutable identity of a workload. Keep this pair stable —
it is used by Deployment/StatefulSet/DaemonSet selectors, Services,
NetworkPolicies, and PodDisruptionBudgets.
*/}}
{{- define "umbrella-chart.selectorLabels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "umbrella-chart.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{ include "umbrella-chart.selectorLabels" . }}
{{- with .Chart.AppVersion }}
app.kubernetes.io/version: {{ . | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Shared pod template used by every workload kind. Per-chart helpers
(<chart>.labels, <chart>.annotations, <chart>.probes, <chart>.volumes, ...)
are resolved dynamically so each subchart keeps its own identity.
*/}}
{{- define "umbrella-chart.podTemplate" -}}
metadata:
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
    {{- with .Values.podAnnotations }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
spec:
  {{- include "umbrella-chart.podSpec" . | trim | nindent 2 }}
{{- end }}

{{- define "umbrella-chart.podSpec" -}}
{{- with .Values.imagePullSecrets }}
imagePullSecrets:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.serviceAccountName }}
serviceAccountName: {{ . }}
{{- end }}
{{- with .Values.terminationGracePeriodSeconds }}
terminationGracePeriodSeconds: {{ . }}
{{- end }}
{{- if .Values.troubleshootingContainer }}
shareProcessNamespace: true
{{- end }}
{{- with .Values.podSecurityContext }}
securityContext:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.initContainers }}
initContainers:
  {{- toYaml . | nindent 2 }}
{{- end }}
containers:
  {{- include "umbrella-chart.containers" . | trim | nindent 2 }}
{{- with .Values.nodeSelector }}
nodeSelector:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.affinity }}
affinity:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.tolerations }}
tolerations:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- $podVolumes := include (print .Chart.Name ".volumes") . | trim }}
{{- if $podVolumes }}
volumes:
  {{- $podVolumes | nindent 2 }}
{{- end }}
{{- end }}

{{- define "umbrella-chart.containers" -}}
{{- if .Values.troubleshootingContainer }}
{{- include (print .Chart.Name ".troubleshootingContainer") . }}
{{- end }}
- name: {{ .Chart.Name }}
  image: {{ include (print .Chart.Name ".image") . }}
  imagePullPolicy: {{ .Values.image.pullPolicy | default "IfNotPresent" }}
  {{- with .Values.command }}
  command:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.args }}
  args:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- if .Values.service }}
  ports:
    - name: {{ .Values.service.portName | default "http" }}
      containerPort: {{ .Values.service.targetPort | default .Values.service.port }}
      protocol: TCP
  {{- end }}
  {{- with .Values.securityContext }}
  securityContext:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- $probes := include (print .Chart.Name ".probes") . | trim }}
  {{- if $probes }}
  {{- $probes | nindent 2 }}
  {{- end }}
  {{- with .Values.lifecycle }}
  lifecycle:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- if or .Values.config .Values.secrets .Values.envFrom }}
  envFrom:
    {{- if .Values.config }}
    - configMapRef:
        name: {{ include (print .Chart.Name ".fullname") . }}
    {{- end }}
    {{- if .Values.secrets }}
    - secretRef:
        name: {{ include (print .Chart.Name ".fullname") . }}
    {{- end }}
    {{- with .Values.envFrom }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  {{- end }}
  {{- with .Values.env }}
  env:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .Values.resources }}
  resources:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- $mounts := include (print .Chart.Name ".volumeMounts") . | trim }}
  {{- if $mounts }}
  volumeMounts:
    {{- $mounts | nindent 4 }}
  {{- end }}
{{- end }}
