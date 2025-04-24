{{- define "component.fullname" -}}
{{- template "umbrella-chart.fullname" . }}-{{ .Chart.Name }}
{{- end }}

{{- define "component.labels" -}}
{{- template "umbrella-chart.labels" . }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
chart-name: {{ .Chart.Name }}
{{- if .Values.additionalLabels }}
{{ toYaml .Values.additionalLabels | trim | nindent 0 }}
{{- end }}
{{- end }}

{{- define "component.annotations" -}}
checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
checksum/secrets: {{ include (print $.Template.BasePath "/secrets.yaml") . | sha256sum }}
{{- if .Values.additionalAnnotations }}
{{ toYaml .Values.additionalAnnotations | trim | nindent 0 }}
{{- end }}
{{- end }}

{{- define "component.image" -}}
{{- $registry := .Values.image.registry | default .Values.global.repository -}}
{{- $name := .Values.image.name | default .Chart.Name -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- printf "%s/%s:%s" $registry $name $tag -}}
{{- end }}

{{- define "troubleshootingContainer" -}}
- name: shell
  image: busybox:1.28
  command: ["sleep", "3600"]
  securityContext:
    capabilities:
      add:
      - SYS_PTRACE
  stdin: true
  tty: true
{{- end }}

{{- define "probes" -}}
livenessProbe:
{{- if .Values.livenessProbe.httpGet }}
  httpGet:
    path: {{ .Values.livenessProbe.httpGet.path }}
    port: {{ .Values.livenessProbe.httpGet.port | default .Values.service.port }}
{{- end }}
{{- if .Values.livenessProbe.tcpSocket }}
  tcpSocket:
    port: {{ .Values.livenessProbe.tcpSocket.port | default .Values.service.port }}
{{- end }}
{{- if .Values.livenessProbe.exec }}
  exec:
    command: {{ toJson .Values.livenessProbe.exec.command }}
{{- end }}
  initialDelaySeconds: {{ .Values.livenessProbe.initialDelaySeconds }}
  periodSeconds: {{ .Values.livenessProbe.periodSeconds }}
  timeoutSeconds: {{ .Values.livenessProbe.timeoutSeconds }}
  failureThreshold: {{ .Values.livenessProbe.failureThreshold }}
readinessProbe:
{{- if .Values.readinessProbe.httpGet }}
  httpGet:
    path: {{ .Values.readinessProbe.httpGet.path }}
    port: {{ .Values.readinessProbe.httpGet.port | default .Values.service.port }}
{{- end }}
{{- if .Values.readinessProbe.tcpSocket }}
  tcpSocket:
    port: {{ .Values.readinessProbe.tcpSocket.port | default .Values.service.port }}
{{- end }}
{{- if .Values.readinessProbe.exec }}
  exec:
    command: {{ toJson .Values.readinessProbe.exec.command }}
{{- end }}
  initialDelaySeconds: {{ .Values.readinessProbe.initialDelaySeconds }}
  periodSeconds: {{ .Values.readinessProbe.periodSeconds }}
  timeoutSeconds: {{ .Values.readinessProbe.timeoutSeconds }}
  successThreshold: {{ .Values.readinessProbe.successThreshold }}
  failureThreshold: {{ .Values.readinessProbe.failureThreshold }}
startupProbe:
{{- if .Values.startupProbe.httpGet }}
  httpGet:
    path: {{ .Values.startupProbe.httpGet.path }}
    port: {{ .Values.startupProbe.httpGet.port | default .Values.service.port }}
{{- end }}
{{- if .Values.startupProbe.tcpSocket }}
  tcpSocket:
    port: {{ .Values.startupProbe.tcpSocket.port | default .Values.service.port }}
{{- end }}
{{- if .Values.startupProbe.exec }}
  exec:
    command: {{ toJson .Values.startupProbe.exec.command }}
{{- end }}
  initialDelaySeconds: {{ .Values.startupProbe.initialDelaySeconds }}
  periodSeconds: {{ .Values.startupProbe.periodSeconds }}
  timeoutSeconds: {{ .Values.startupProbe.timeoutSeconds }}
  failureThreshold: {{ .Values.startupProbe.failureThreshold }}
{{- end }}

