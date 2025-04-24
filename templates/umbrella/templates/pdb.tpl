{{- define "umbrella-chart.pdb" }}
{{- if .Values.pdb.enabled }}
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  minAvailable: {{ .Values.pdb.minAvailable }}
  selector:
    matchLabels:
      app: {{ include (print .Chart.Name ".fullname") . }}
{{- end }}
{{- end }}
