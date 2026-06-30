package deploy

import (
	"strings"
	"testing"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

// renderChart loads and renders chart/ with its default values, returning all
// rendered manifests concatenated.
func renderChart(t *testing.T) string {
	t.Helper()
	ch, err := loader.Load("../../chart")
	if err != nil {
		t.Fatalf("load chart: %v", err)
	}
	vals, err := chartutil.ToRenderValues(ch, chartutil.Values{}, chartutil.ReleaseOptions{
		Name:      "chartpress",
		Namespace: "chartpress-system",
	}, nil)
	if err != nil {
		t.Fatalf("ToRenderValues: %v", err)
	}
	out, err := engine.Render(ch, vals)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	var b strings.Builder
	for _, v := range out {
		b.WriteString(v)
		b.WriteString("\n---\n")
	}
	return b.String()
}

func TestBackendRBACGrantsChartpressConfigs(t *testing.T) {
	man := renderChart(t)
	for _, want := range []string{
		"kind: Role",
		"kind: RoleBinding",
		"kind: ServiceAccount",
		"chartpress.dev",
		"chartpressconfigs",
		"create",
		"watch",
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("rendered chart missing %q", want)
		}
	}
}

func TestBackendDeploymentHasDownwardNamespaceAndSA(t *testing.T) {
	man := renderChart(t)
	for _, want := range []string{
		"serviceAccountName: chartpress-backend",
		"POD_NAMESPACE",
		"fieldRef",
		"metadata.namespace",
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("backend deployment missing %q", want)
		}
	}
}
