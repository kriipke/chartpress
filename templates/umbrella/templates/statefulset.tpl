{{- define "umbrella-chart.statefulset" }}
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  serviceName: {{ .Chart.Name }}-headless
  replicas: {{ .Values.podCount.static | default 1 }}
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
            {{- if or (hasKey . "value") .valueFrom }}
            - {{ toYaml . | nindent 14 | trim }}
            {{- else if .secretKeyRef }}
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
          {{- with .Values.volumeMounts }}
          volumeMounts:
            {{- toYaml . | nindent 12 }}
          {{- end }}
      {{- with .Values.volumes }}
      volumes:
        {{- toYaml . | nindent 8 }}
      {{- end }}
  volumeClaimTemplates:
    {{- range .Values.persistentVolumeClaims }}
    - metadata:
        name: {{ .name }}
        labels:
          {{- include (print $.Chart.Name ".labels") $ | nindent 10 }}
      spec:
        accessModes: {{ toYaml .accessModes | nindent 8 }}
        resources:
          requests:
            storage: {{ .resources.requests.storage }}
        storageClassName: {{ .storageClassName }}
    {{- end }}
{{- end }}

