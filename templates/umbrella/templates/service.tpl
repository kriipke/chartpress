{{- define "umbrella-chart.service" -}}
apiVersion: v1
kind: Service
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  ports:
    - port: {{ .Values.service.port }}
  selector:
    app: {{ include (print .Chart.Name ".fullname") . }}
{{- end }}
