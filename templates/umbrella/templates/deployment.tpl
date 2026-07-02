{{- define "umbrella-chart.deployment" -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  {{- if eq .Values.podCount.type "static" }}
  replicas: {{ .Values.podCount.static }}
  {{- end }}
  {{- with .Values.strategy }}
  strategy:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  selector:
    matchLabels:
      {{- include (print .Chart.Name ".selectorLabels") . | nindent 6 }}
  template:
    {{- include "umbrella-chart.podTemplate" . | nindent 4 }}
{{- end }}
