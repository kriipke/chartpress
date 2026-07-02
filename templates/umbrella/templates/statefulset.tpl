{{- define "umbrella-chart.statefulset" -}}
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  serviceName: {{ include (print .Chart.Name ".fullname") . }}-headless
  {{- if eq .Values.podCount.type "static" }}
  replicas: {{ .Values.podCount.static }}
  {{- end }}
  {{- with .Values.podManagementPolicy }}
  podManagementPolicy: {{ . }}
  {{- end }}
  {{- with .Values.updateStrategy }}
  updateStrategy:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  selector:
    matchLabels:
      {{- include (print .Chart.Name ".selectorLabels") . | nindent 6 }}
  template:
    {{- include "umbrella-chart.podTemplate" . | nindent 4 }}
  {{- $claims := include (print .Chart.Name ".volumeClaimTemplates") . | trim }}
  {{- if $claims }}
  volumeClaimTemplates:
    {{- $claims | nindent 4 }}
  {{- end }}
{{- end }}
