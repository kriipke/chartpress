// internal/engine/traits.go
//
// Trait tailoring: turn a subchart's ResolvedTraits into the right manifests
// and values. Manifest-level pruning drops templates whose values are gone;
// values-level surgery edits the authored values.yaml TEXT (comments are half
// the deliverable) using the `# chartpress:section <name>` markers in
// templates/subchart/values.yaml. Marker lines never survive into output.
package engine

import (
	"fmt"
	"regexp"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
)

// applyTraits tailors one subchart to its resolved traits. It must run before
// generatedSubchartValues is appended (sections are template text) and before
// applyIngress (which needs the effective controller).
func applyTraits(sub *chart.Chart, rt ResolvedTraits, spec Spec) error {
	text := rawValues(sub)

	// --- exposure ---
	switch rt.Exposure {
	case "http":
		if rt.Port != defaultPortHTTP {
			text = replaceSection(text, "service", serviceBlock(rt.Port, "http", ""))
		}
	case "grpc":
		text = replaceSection(text, "service", serviceBlock(rt.Port, "grpc", "grpc"))
		text = replaceSection(text, "probes", grpcProbesBlock)
	case "tcp":
		text = replaceSection(text, "service", serviceBlock(rt.Port, "tcp", ""))
		text = replaceSection(text, "probes", tcpProbesBlock)
	case "none":
		text = dropSection(text, "service")
		text = dropSection(text, "networkPolicy")
		text = replaceSection(text, "probes", workerProbesBlock)
		sub.Templates = dropTemplate(sub.Templates, "templates/service.yaml")
		sub.Templates = dropTemplate(sub.Templates, "templates/networkPolicy.yaml")
	}

	// --- ingress (the template swap happens in applyIngress) ---
	if !rt.Ingress {
		text = dropSection(text, "ingress")
	}

	// --- scaling ---
	switch rt.Scaling {
	case "auto":
		// Keep the template's podCount section as-is: static by default with the
		// dynamic (HPA) block ready to switch on. This is the back-compat anchor —
		// "auto" means autoscaling is AVAILABLE, sized during handoff, not forced
		// on before resources are sized.
	case "fixed":
		text = replaceSection(text, "podCount", fixedPodCountBlock(rt.Pattern))
		sub.Templates = dropTemplate(sub.Templates, "templates/hpa.yaml")
	case "singleton":
		text = replaceSection(text, "podCount", singletonPodCountBlock(rt.Workload))
		text = dropSection(text, "pdb")
		sub.Templates = dropTemplate(sub.Templates, "templates/hpa.yaml")
		sub.Templates = dropTemplate(sub.Templates, "templates/pdb.yaml")
	}

	text = stripSectionMarkers(text)
	if err := setValues(sub, text); err != nil {
		return err
	}
	return applyPatternExtras(sub, rt)
}

// effectiveIngressController returns the controller applyIngress should
// template for this subchart: the platform controller when the subchart is
// routed, "none" otherwise.
func effectiveIngressController(rt ResolvedTraits, rules Rules) string {
	if !rt.Ingress {
		return "none"
	}
	return rules.Ingress
}

// --- section surgery -------------------------------------------------------

func sectionRE(name string) *regexp.Regexp {
	// A section spans its start marker line through the next end marker line.
	return regexp.MustCompile(`(?ms)^# chartpress:section ` + regexp.QuoteMeta(name) + `$.*?^# chartpress:end$\n?`)
}

func dropSection(text, name string) string {
	out := sectionRE(name).ReplaceAllString(text, "")
	return collapseBlankRuns(out)
}

func replaceSection(text, name, block string) string {
	return sectionRE(name).ReplaceAllString(text, strings.TrimRight(block, "\n")+"\n")
}

// stripSectionMarkers removes any surviving marker lines so generated charts
// never leak the mechanism.
func stripSectionMarkers(text string) string {
	lines := strings.Split(text, "\n")
	out := lines[:0]
	for _, l := range lines {
		if strings.HasPrefix(l, "# chartpress:section ") || l == "# chartpress:end" {
			continue
		}
		out = append(out, l)
	}
	return collapseBlankRuns(strings.Join(out, "\n"))
}

func collapseBlankRuns(text string) string {
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}
	return text
}

// --- generated section blocks ----------------------------------------------

func serviceBlock(port int, portName, appProtocol string) string {
	var b strings.Builder
	b.WriteString("service:\n")
	todo := "the port your app listens on"
	if portName == "grpc" {
		todo = "the port your gRPC server listens on"
	}
	fmt.Fprintf(&b, "  port: %d # TODO(chartpress): %s\n", port, todo)
	fmt.Fprintf(&b, "  portName: %s\n", portName)
	if appProtocol != "" {
		fmt.Fprintf(&b, "  appProtocol: %s\n", appProtocol)
	}
	b.WriteString("  # targetPort: defaults to port\n")
	b.WriteString("  # type: ClusterIP\n")
	return b.String()
}

const grpcProbesBlock = `# Native gRPC probes: the kubelet calls the standard gRPC health service
# (grpc.health.v1.Health). Ports default to service.port.
# TODO(chartpress): confirm the server registers the gRPC health service; if it
# can't, switch these to an exec probe running grpc_health_probe.
livenessProbe:
  grpc: {}
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 1
  failureThreshold: 3

readinessProbe:
  grpc: {}
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 1
  successThreshold: 1
  failureThreshold: 3
`

const tcpProbesBlock = `# TCP socket probes: the kubelet checks the port accepts connections.
# Ports default to service.port.
livenessProbe:
  tcpSocket: {}
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 1
  failureThreshold: 3

readinessProbe:
  tcpSocket: {}
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 1
  successThreshold: 1
  failureThreshold: 3
`

const workerProbesBlock = `# This workload serves no traffic, so HTTP probes were removed. Give it a
# liveness signal an orchestrator can check — a heartbeat file the main loop
# touches, or a lightweight exec command.
# TODO(chartpress): define a liveness signal for this worker.
# livenessProbe:
#   exec:
#     command: ["sh", "-c", "test $(find /tmp/heartbeat -mmin -1)"]
#   periodSeconds: 30
#   failureThreshold: 3
`

func fixedPodCountBlock(pattern string) string {
	todo := "set the replica count"
	if pattern == "stream-processor" {
		todo = "set to the topic's partition count or less (replicas ≤ partitions); do NOT add an HPA — autoscaling churns consumer-group rebalances"
	}
	return fmt.Sprintf(`podCount:
  # Fixed replica count — no HPA is generated for this workload.
  type: static
  static: 1 # TODO(chartpress): %s
`, todo)
}

func singletonPodCountBlock(workload string) string {
	block := `podCount:
  # singleton — exactly one instance may ever run. Do not raise this and do
  # not add an HPA; if overlap is unsafe the app needs leader election too.
  type: static
  static: 1
`
	if workload == "deployment" {
		block += `
# Recreate prevents two instances overlapping during a rollout.
strategy:
  type: Recreate
`
	}
	return block
}

// --- pattern extras ---------------------------------------------------------

// applyPatternExtras appends the per-pattern front-loaded items that traits
// alone don't express. All extras are values-level so they survive every
// file toggle.
func applyPatternExtras(sub *chart.Chart, rt ResolvedTraits) error {
	switch rt.Pattern {
	case "ml-inference":
		return appendValues(sub, `# Generated by chartpress (ml-inference): model servers boot slowly; a startup
# probe stops the liveness probe from killing the pod during model load.
startupProbe:
  httpGet:
    path: /healthz # TODO(chartpress): your health endpoint
  periodSeconds: 10
  failureThreshold: 60 # TODO(chartpress): periodSeconds x failureThreshold must exceed the slowest model load
`)
	case "realtime-gateway":
		return appendValues(sub, `# Generated by chartpress (realtime-gateway): long-lived connections need a
# drain window on shutdown. Pair the preStop sleep with your client retry.
# TODO(chartpress): set from your measured connection drain time.
# terminationGracePeriodSeconds: 60
# lifecycle:
#   preStop:
#     exec:
#       command: ["sleep", "10"]
`)
	}
	return nil
}
