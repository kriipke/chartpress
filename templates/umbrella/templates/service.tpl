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
  type: {{ .Values.service.type | default "ClusterIP" }}
  ports:
    - name: {{ .Values.service.portName | default "http" }}
      port: {{ .Values.service.port }}
      targetPort: {{ .Values.service.targetPort | default .Values.service.port }}
      protocol: TCP
  selector:
    {{- include (print .Chart.Name ".selectorLabels") . | nindent 4 }}
{{- end }}

{{/*
Headless service backing a StatefulSet's serviceName. Emitted via
templates/headless-service.yaml, which chartpress adds to statefulset
subcharts.
*/}}
{{- define "umbrella-chart.headlessService" -}}
apiVersion: v1
kind: Service
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}-headless
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  clusterIP: None
  publishNotReadyAddresses: true
  {{- if .Values.service }}
  ports:
    - name: {{ .Values.service.portName | default "http" }}
      port: {{ .Values.service.port }}
      targetPort: {{ .Values.service.targetPort | default .Values.service.port }}
      protocol: TCP
  {{- end }}
  selector:
    {{- include (print .Chart.Name ".selectorLabels") . | nindent 4 }}
{{- end }}
