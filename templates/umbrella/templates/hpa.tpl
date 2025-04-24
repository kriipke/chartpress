{{- define "umbrella-chart.hpa" -}}
{{- if eq .Values.podCount.type "dynamic" }}
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ include (print .Chart.Name ".fullname") . }}
  minReplicas: {{ .Values.podCount.dynamic.minReplicas }}
  maxReplicas: {{ .Values.podCount.dynamic.maxReplicas }}
  metrics:
    {{- with .Values.podCount.dynamic.metrics }}
      {{- toYaml . | nindent 4 }}
    {{- else }}
      {{- with .Values.podCount.dynamic.targetMemoryUtilizationPercentage }}
      - type: Resource
        resource:
          name: memory
          target:
            type: Utilization
            averageUtilization: {{ . }}
      {{- end }}
      {{- with .Values.podCount.dynamic.targetCPUUtilizationPercentage }}
      - type: Resource
        resource:
          name: cpu
          target:
            type: Utilization
            averageUtilization: {{ . }}
      {{- end }}
    {{- end }}
  {{- with .Values.podCount.dynamic.behavior }}
  behavior:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end }}
{{- end }}
