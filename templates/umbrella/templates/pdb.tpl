{{- define "umbrella-chart.pdb" -}}
{{- if and .Values.pdb .Values.pdb.enabled }}
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  {{- with .Values.pdb.minAvailable }}
  minAvailable: {{ . }}
  {{- end }}
  {{- with .Values.pdb.maxUnavailable }}
  maxUnavailable: {{ . }}
  {{- end }}
  selector:
    matchLabels:
      {{- include (print .Chart.Name ".selectorLabels") . | nindent 6 }}
{{- end }}
{{- end }}
