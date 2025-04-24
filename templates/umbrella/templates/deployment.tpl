{{- define "umbrella-chart.deployment" }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  {{- if eq .Values.podCount.type "static" }}
  replicas: {{ .Values.podCount.static }}
  {{- end }}
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
      {{- if .Values.troubleshootingContainer }}
      shareProcessNamespace: true
      {{- end }}
      {{- with .Values.initContainers }}
      initContainers:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      containers:
        {{- if .Values.troubleshootingContainer }}
        {{- include "troubleshootingContainer" . | nindent 8 }}
        {{- end }}
        - name: {{ .Chart.Name }}
          image: {{ include (print .Chart.Name ".image") . }}
          {{- if .Values.service }}
          ports:
            - containerPort: {{ .Values.service.port }}
          {{- end }}
          {{- include "probes" . | nindent 10 }}
          {{- with .Values.lifecycle }}
          lifecycle:
            {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .Values.envFrom }}
          envFrom:
            {{- range . }}
            - {{- if .configMapRef }}
              configMapRef:
                name: {{ .configMapRef.name }}
              {{- else if .secretRef }}
              secretRef:
                name: {{ .secretRef.name }}
              {{- end }}
            {{- end }}
          {{- end }}
          {{- with .Values.env }}
          env:
            {{- range . }}
            {{- $name := .name }}
            {{- if .secretKeyRef }}
              {{- if eq .keys "*" }}
              # All keys in the secret will be mounted as env vars (requires envFrom)
              {{- else }}
                {{- range .keys }}
            - name: {{ . }}
              valueFrom:
                secretKeyRef:
                  name: {{ $.secretKeyRef.name }}
                  key: {{ . }}
                {{- end }}
              {{- end }}
            {{- else if .configMapKeyRef }}
              {{- if eq .keys "*" }}
              # All keys in the configMap will be mounted as env vars (requires envFrom)
              {{- else }}
                {{- range .keys }}
            - name: {{ . }}
              valueFrom:
                configMapKeyRef:
                  name: {{ $.configMapKeyRef.name }}
                  key: {{ . }}
                {{- end }}
              {{- end }}
            {{- end }}
            {{- end }}
          {{- end }}
          {{- with .Values.volumes }}
          volumeMounts:
            {{- range . }}
            - name: {{ .name }}
              mountPath: {{ .mountPath }}
              readOnly: true
            {{- end }}
          {{- end }}
      {{- with .Values.volumes }}
      volumes:
        {{- range . }}
        - name: {{ .name }}
          {{- if .secret }}
          secret:
            secretName: {{ .secret }}
          {{- else if .configMap }}
          configMap:
            name: {{ .configMap }}
          {{- end }}
        {{- end }}
      {{- end }}
{{- end }}
