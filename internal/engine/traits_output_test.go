// internal/engine/traits_output_test.go
//
// Trait tailoring end-to-end: build charts from the real templates and check
// the generated files, values, and rendered manifests per pattern.
package engine

import (
	"strings"
	"testing"

	"helm.sh/helm/v3/pkg/chart"
)

func depByName(t *testing.T, ch *chart.Chart, name string) *chart.Chart {
	t.Helper()
	for _, d := range ch.Dependencies() {
		if d.Name() == name {
			return d
		}
	}
	t.Fatalf("subchart %q not found", name)
	return nil
}

func templateNames(ch *chart.Chart) map[string]bool {
	out := map[string]bool{}
	for _, tpl := range ch.Templates {
		out[tpl.Name] = true
	}
	return out
}

func patternSpec(subcharts ...Subchart) Spec {
	return Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         subcharts,
		Rules:             DefaultRules(),
	})
}

func TestWorkerOutput(t *testing.T) {
	ch, err := BuildChart(patternSpec(Subchart{Name: "emailer", Pattern: "worker"}), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	sub := depByName(t, ch, "emailer")
	tpls := templateNames(sub)
	for _, gone := range []string{"templates/service.yaml", "templates/ingress.yaml", "templates/networkPolicy.yaml"} {
		if tpls[gone] {
			t.Fatalf("worker should not ship %s", gone)
		}
	}
	vals := rawValues(sub)
	for _, gone := range []string{"\nservice:", "\ningress:", "\nnetworkPolicy:", "httpGet"} {
		if strings.Contains(vals, gone) {
			t.Fatalf("worker values should not contain %q:\n%s", gone, vals)
		}
	}
	if !strings.Contains(vals, "liveness signal") {
		t.Fatalf("worker values should carry the liveness-signal handoff stub:\n%s", vals)
	}
	if strings.Contains(vals, "chartpress:section") || strings.Contains(vals, "chartpress:end") {
		t.Fatalf("section markers leaked into output:\n%s", vals)
	}
	man := allManifests(renderChart(t, ch))
	for _, gone := range []string{"kind: Service", "kind: Ingress", "kind: NetworkPolicy"} {
		if strings.Contains(man, gone) {
			t.Fatalf("worker platform should not render %s:\n%s", gone, man)
		}
	}
	if !strings.Contains(man, "kind: Deployment") {
		t.Fatalf("worker must still render its Deployment")
	}
}

func TestSchedulerSingletonOutput(t *testing.T) {
	ch, err := BuildChart(patternSpec(Subchart{Name: "dispatcher", Pattern: "scheduler"}), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	sub := depByName(t, ch, "dispatcher")
	if templateNames(sub)["templates/hpa.yaml"] {
		t.Fatalf("singleton should not ship hpa.yaml")
	}
	vals := rawValues(sub)
	if !strings.Contains(vals, "singleton") || !strings.Contains(vals, "static: 1") {
		t.Fatalf("singleton podCount pin missing:\n%s", vals)
	}
	if strings.Contains(vals, "\npdb:") {
		t.Fatalf("singleton should not carry a pdb block:\n%s", vals)
	}
	man := allManifests(renderChart(t, ch))
	if !strings.Contains(man, "type: Recreate") {
		t.Fatalf("singleton deployment must use Recreate:\n%s", man)
	}
	if !strings.Contains(man, "replicas: 1") {
		t.Fatalf("singleton must pin replicas: 1:\n%s", man)
	}
}

func TestGrpcServiceOutput(t *testing.T) {
	ch, err := BuildChart(patternSpec(Subchart{Name: "pricing", Pattern: "grpc-service"}), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	sub := depByName(t, ch, "pricing")
	if templateNames(sub)["templates/ingress.yaml"] {
		t.Fatalf("grpc-service defaults to no ingress")
	}
	man := allManifests(renderChart(t, ch))
	if !strings.Contains(man, "appProtocol: grpc") {
		t.Fatalf("grpc service port must carry appProtocol:\n%s", man)
	}
	if !strings.Contains(man, "grpc:") || !strings.Contains(man, "port: 50051") {
		t.Fatalf("grpc probes/port missing:\n%s", man)
	}
	if strings.Contains(man, "httpGet") {
		t.Fatalf("grpc service should not carry httpGet probes:\n%s", man)
	}
}

func TestSharedEnvOptOut(t *testing.T) {
	spec := patternSpec(
		Subchart{Name: "api", Pattern: "api-microservice"},
		Subchart{Name: "web", Pattern: "web-frontend"}, // shared_env: false by default
	)
	spec.Rules.SharedSecretsConfig = true
	spec = Normalize(spec)
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	if !strings.Contains(rawValues(depByName(t, ch, "api")), "demo-shared-secrets") {
		t.Fatalf("api should mount the shared secret")
	}
	if strings.Contains(rawValues(depByName(t, ch, "web")), "demo-shared-secrets") {
		t.Fatalf("web-frontend (shared_env: false) must not mount the shared secret")
	}
	// The umbrella-level Secret still renders for the consumers that want it.
	man := allManifests(renderChart(t, ch))
	if !strings.Contains(man, "demo-shared-secrets") {
		t.Fatalf("umbrella shared Secret must still render")
	}
}

func TestMlInferenceStartupProbeExtra(t *testing.T) {
	ch, err := BuildChart(patternSpec(Subchart{Name: "embedder", Pattern: "ml-inference"}), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	man := allManifests(renderChart(t, ch))
	if !strings.Contains(man, "startupProbe") {
		t.Fatalf("ml-inference must front-load a startupProbe:\n%s", man)
	}
	if templateNames(depByName(t, ch, "embedder"))["templates/hpa.yaml"] {
		t.Fatalf("ml-inference is fixed scaling — no hpa.yaml")
	}
}

func TestIngressOverrideOnGrpc(t *testing.T) {
	ch, err := BuildChart(patternSpec(Subchart{Name: "pricing", Pattern: "grpc-service", Ingress: boolPtr(true)}), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	if !templateNames(depByName(t, ch, "pricing"))["templates/ingress.yaml"] {
		t.Fatalf("explicit ingress override must ship ingress.yaml")
	}
}

func TestBackCompatSubchartValuesUnchanged(t *testing.T) {
	// A pre-pattern spec must produce a subchart whose values keep today's
	// shape: service on 8080, httpGet probes, static podCount with the dynamic
	// block available, ingress and networkPolicy blocks present.
	ch, err := BuildChart(basicSpec(), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	vals := rawValues(depByName(t, ch, "api"))
	for _, want := range []string{"port: 8080", "httpGet", "type: static", "minReplicas", "\ningress:", "\nnetworkPolicy:", "\npdb:"} {
		if !strings.Contains(vals, want) {
			t.Fatalf("back-compat values missing %q:\n%s", want, vals)
		}
	}
	if strings.Contains(vals, "chartpress:section") {
		t.Fatalf("markers leaked:\n%s", vals)
	}
	tpls := templateNames(depByName(t, ch, "api"))
	for _, want := range []string{"templates/service.yaml", "templates/ingress.yaml", "templates/hpa.yaml", "templates/networkPolicy.yaml"} {
		if !tpls[want] {
			t.Fatalf("back-compat template %s missing", want)
		}
	}
}
