// internal/engine/ingress.go
package engine

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
)

var ingressClassAnnotations = map[string]string{
	"alb":     "    alb.ingress.kubernetes.io/scheme: internet-facing\n    alb.ingress.kubernetes.io/target-type: ip\n",
	"nginx":   "    nginx.ingress.kubernetes.io/rewrite-target: /\n",
	"gce":     "    kubernetes.io/ingress.class: gce\n",
	"traefik": "",
}

func ingressTemplate(controller string) string {
	ann := ingressClassAnnotations[controller]
	return fmt.Sprintf(`{{- if and .Values.ingress .Values.ingress.host }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
%s    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
    {{- with .Values.ingress.annotations }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
spec:
  ingressClassName: %s
  {{- with .Values.ingress.tls }}
  tls:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  rules:
    - host: {{ .Values.ingress.host }}
      http:
        paths:
          - path: {{ default "/" .Values.ingress.path | quote }}
            pathType: Prefix
            backend:
              service:
                name: {{ include (print .Chart.Name ".fullname") . }}
                port:
                  number: {{ .Values.service.port }}
{{- end }}
`, ann, controller)
}

func istioTemplate() string {
	return `{{- if and .Values.ingress .Values.ingress.host }}
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
spec:
  selector:
    istio: ingressgateway
  servers:
    - port:
        number: 80
        name: http
        protocol: HTTP
      hosts:
        - {{ .Values.ingress.host | quote }}
---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
spec:
  hosts:
    - {{ .Values.ingress.host | quote }}
  gateways:
    - {{ include (print .Chart.Name ".fullname") . }}
  http:
    - match:
        - uri:
            prefix: {{ default "/" .Values.ingress.path | quote }}
      route:
        - destination:
            host: {{ include (print .Chart.Name ".fullname") . }}
            port:
              number: {{ .Values.service.port }}
{{- end }}
`
}

// applyIngress replaces the subchart ingress manifest with a controller-specific
// one, or removes it for "none"; for "istio" emits a Gateway + VirtualService.
func applyIngress(sub *chart.Chart, controller string) {
	sub.Templates = dropTemplate(sub.Templates, "templates/ingress.yaml")
	switch controller {
	case "none":
		return
	case "istio":
		sub.Templates = append(sub.Templates, &chart.File{
			Name: "templates/istio.yaml",
			Data: []byte(istioTemplate()),
		})
		return
	default:
		sub.Templates = append(sub.Templates, &chart.File{
			Name: "templates/ingress.yaml",
			Data: []byte(ingressTemplate(controller)),
		})
	}
}
