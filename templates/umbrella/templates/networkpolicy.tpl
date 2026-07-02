{{- define "umbrella-chart.networkPolicy" -}}
{{- if and .Values.networkPolicy .Values.networkPolicy.enabled }}
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
      {{- include (print .Chart.Name ".selectorLabels") . | nindent 6 }}
  policyTypes:
    - Ingress
  ingress:
    - from:
        {{- toYaml .Values.networkPolicy.ingressFrom | nindent 8 }}
{{- end }}
{{- end }}
