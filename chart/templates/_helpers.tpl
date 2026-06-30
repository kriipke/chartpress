{{- define "chartpress.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{- define "chartpress.s3env" -}}
- name: S3_ENDPOINT
  value: {{ .Values.s3.endpoint | quote }}
- name: S3_BUCKET
  value: {{ .Values.s3.bucket | quote }}
- name: S3_REGION
  value: {{ .Values.s3.region | quote }}
- name: S3_USE_SSL
  value: {{ .Values.s3.useSSL | quote }}
- name: S3_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      name: {{ .Values.s3.existingSecret | default "chartpress-s3" }}
      key: {{ .Values.s3.accessKeyKey }}
- name: S3_SECRET_KEY
  valueFrom:
    secretKeyRef:
      name: {{ .Values.s3.existingSecret | default "chartpress-s3" }}
      key: {{ .Values.s3.secretKeyKey }}
{{- end -}}
