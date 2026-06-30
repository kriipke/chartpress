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
	return fmt.Sprintf(`{{- if .Values.ingress.host }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
%s    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  ingressClassName: %s
  rules:
    - host: {{ .Values.ingress.host }}
      http:
        paths:
          - path: {{ default "/" .Values.ingress.path }}
            pathType: Prefix
            backend:
              service:
                name: {{ include (print .Chart.Name ".fullname") . }}
                port:
                  number: {{ .Values.service.port }}
{{- end }}
`, ann, controller)
}

// applyIngress replaces the subchart ingress manifest with a controller-specific
// one, or removes it for "none"/"istio" (istio handled separately in Task 11).
func applyIngress(sub *chart.Chart, controller string) {
	sub.Templates = dropTemplate(sub.Templates, "templates/ingress.yaml")
	switch controller {
	case "none", "istio":
		return
	default:
		sub.Templates = append(sub.Templates, &chart.File{
			Name: "templates/ingress.yaml",
			Data: []byte(ingressTemplate(controller)),
		})
	}
}
