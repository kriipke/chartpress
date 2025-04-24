{{- define "umbrella-chart.daemonset" }}
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  selector:
    matchLabels:
      app: {{ include (print .Chart.Name ".fullname") . }}
  template:
    metadata:
      labels:
        {{- include (print .Chart.Name ".labels") . | nindent 8 }}
      annotations:
        {{- include (print .Chart.Name ".annotations") . | nindent 8 }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: {{ include (print .Chart.Name ".image") . }}
          ports:
            - containerPort: {{ .Values.service.port }}
          {{- include "probes" . | nindent 10 }}
          {{- with .Values.volumeMounts }}
          volumeMounts:
            {{- toYaml . | nindent 12 }}
          {{- end }}
      {{- with .Values.volumes }}
      volumes:
        {{- toYaml . | nindent 8 }}
      {{- end }}
{{- end }}

