{{- define "umbrella-chart.ingress" -}}
{{- if .Values.ingress.host }}
  {{- if not (or (eq .Values.ingress.type "aws") (eq .Values.ingress.type "emissary")) }}
    {{- fail ".Values.ingress.type must be either 'aws' or 'emissary' when .Values.ingress.host is not empty" }}
  {{- end }}
{{- end }}

{{- if eq .Values.ingress.type "aws" }}

apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
    alb.ingress.kubernetes.io/scheme: {{ default "internal" .Values.ingress.aws.scheme }}
    alb.ingress.kubernetes.io/target-type: {{ default "ip" .Values.ingress.aws.targetType }}
    alb.ingress.kubernetes.io/subnets: {{ required ".Values.ingress.aws.subnet must contain values." ( .Values.ingress.aws.subnets | join "," ) }}
    alb.ingress.kubernetes.io/security-groups: {{  required ".Values.ingress.aws.subnet must contain values." ( .Values.ingress.aws.securityGroups | join "," ) }}
    {{- if .Values.ingress.aws.sslCertificateArn }}
    alb.ingress.kubernetes.io/certificate-arn: {{ .Values.ingress.aws.sslCertificateArn }}
    {{- end }}
spec:
  rules:
    - host: {{ .Values.ingress.host }}
      http:
        paths:
          - path: {{ default "/" .Values.ingress.path }}
            pathType: Prefix
            backend:
              service:
                name: {{ .Release.Name }}-service
                port:
                  number: 80

{{- else if eq .Values.ingress.type "emissary" }}

apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
    {{- if .Values.ingress.emissary.annotations }}
    {{- with .Values.ingress.emissary.annotations }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
    {{- end }}
spec:
  prefix: {{ default "/" .Values.ingress.path }}
  host: {{ .Values.ingress.host }}
  service: {{ .Release.Name }}-service.{{ .Release.Namespace }}.svc.cluster.local
  timeout_ms: {{ default "60000" .Values.ingress.emissary.timeout }}
  retries: {{ default "3" .Values.ingress.emissary.retries }}

{{- else }}
# Ingress type is not specified or invalid
{{- fail "ingress.type must be either 'aws' or 'emissary'" }}
{{- end }}
{{- end }}
