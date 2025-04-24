{{- define "umbrella-chart.networkPolicy" }}
{{- if .Values.networkPolicy.enabled }}
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  podSelector:
    matchLabels:
      app: {{ include (print .Chart.Name ".fullname") . }}
  policyTypes:
    - Ingress
  ingress:
    - from:
      {{- toYaml .Values.networkPolicy.ingressFrom | nindent 6 }}
{{- end }}
{{- end }}
