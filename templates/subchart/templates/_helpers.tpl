{{- define "component.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "component.selectorLabels" -}}
{{ include "umbrella-chart.selectorLabels" . }}
{{- end }}

{{- define "component.labels" -}}
{{ include "umbrella-chart.labels" . }}
{{- with .Values.additionalLabels }}
{{ toYaml . | trim }}
{{- end }}
{{- end }}

{{- define "component.annotations" -}}
checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
checksum/secrets: {{ include (print $.Template.BasePath "/secrets.yaml") . | sha256sum }}
{{- with .Values.additionalAnnotations }}
{{ toYaml . | trim }}
{{- end }}
{{- end }}

{{- define "component.image" -}}
{{- $registry := .Values.image.registry | default .Values.global.repository -}}
{{- $name := .Values.image.name | default .Chart.Name -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry $name $tag }}
{{- else }}
{{- printf "%s:%s" $name $tag }}
{{- end }}
{{- end }}

{{- define "component.troubleshootingContainer" -}}
- name: shell
  image: busybox:1.28
  command: ["sleep", "3600"]
  securityContext:
    capabilities:
      add:
        - SYS_PTRACE
  stdin: true
  tty: true
{{- end }}

{{/*
Probes render only when their values block is present, so any probe can be
disabled by removing (or nulling) it. httpGet/tcpSocket ports default to
service.port.
*/}}
{{- define "component.probes" -}}
{{- with .Values.livenessProbe }}
livenessProbe:
  {{- include "component.probeSpec" (dict "probe" . "root" $) | trim | nindent 2 }}
{{- end }}
{{- with .Values.readinessProbe }}
readinessProbe:
  {{- include "component.probeSpec" (dict "probe" . "root" $) | trim | nindent 2 }}
{{- end }}
{{- with .Values.startupProbe }}
startupProbe:
  {{- include "component.probeSpec" (dict "probe" . "root" $) | trim | nindent 2 }}
{{- end }}
{{- end }}

{{- define "component.probeSpec" -}}
{{- $probe := .probe }}
{{- $defaultPort := "" }}
{{- with .root.Values.service }}
{{- $defaultPort = .port }}
{{- end }}
{{- with $probe.httpGet }}
httpGet:
  path: {{ .path }}
  port: {{ .port | default $defaultPort }}
{{- end }}
{{- if hasKey $probe "tcpSocket" }}
tcpSocket:
  port: {{ $probe.tcpSocket.port | default $defaultPort }}
{{- end }}
{{- if hasKey $probe "grpc" }}
grpc:
  port: {{ $probe.grpc.port | default $defaultPort }}
{{- end }}
{{- with $probe.exec }}
exec:
  command: {{ toJson .command }}
{{- end }}
{{- range $field := list "initialDelaySeconds" "periodSeconds" "timeoutSeconds" "successThreshold" "failureThreshold" "terminationGracePeriodSeconds" }}
{{- if hasKey $probe $field }}
{{ $field }}: {{ index $probe $field }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Volume helpers share one values schema across all workload kinds:

  volumes:
    - name: app-config
      mountPath: /etc/app/config
      configMap: app-configmap        # or: secret, existingClaim, hostPath, emptyDir
    - name: data                      # statefulset only: becomes a volumeClaimTemplate
      mountPath: /data
      pvc:
        accessModes: ["ReadWriteOnce"]
        resources: { requests: { storage: 1Gi } }
*/}}
{{- define "component.volumeMounts" -}}
{{- range .Values.volumes }}
- name: {{ .name }}
  mountPath: {{ .mountPath }}
  {{- with .subPath }}
  subPath: {{ . }}
  {{- end }}
  readOnly: {{ .readOnly | default false }}
{{- end }}
{{- end }}

{{- define "component.volumes" -}}
{{- range .Values.volumes }}
{{- if .configMap }}
- name: {{ .name }}
  configMap:
    name: {{ .configMap }}
{{- else if .secret }}
- name: {{ .name }}
  secret:
    secretName: {{ .secret }}
{{- else if .existingClaim }}
- name: {{ .name }}
  persistentVolumeClaim:
    claimName: {{ .existingClaim }}
{{- else if .hostPath }}
- name: {{ .name }}
  hostPath:
    path: {{ .hostPath }}
{{- else if hasKey . "emptyDir" }}
- name: {{ .name }}
  emptyDir: {}
{{- else if not .pvc }}
{{- fail (printf "volume %q must set one of: configMap, secret, existingClaim, hostPath, emptyDir, pvc" .name) }}
{{- end }}
{{- end }}
{{- end }}

{{- define "component.volumeClaimTemplates" -}}
{{- range .Values.volumes }}
{{- if .pvc }}
- metadata:
    name: {{ .name }}
  spec:
    accessModes:
      {{- toYaml .pvc.accessModes | nindent 6 }}
    {{- with .pvc.storageClassName }}
    storageClassName: {{ . }}
    {{- end }}
    resources:
      requests:
        storage: {{ .pvc.resources.requests.storage }}
{{- end }}
{{- end }}
{{- end }}
