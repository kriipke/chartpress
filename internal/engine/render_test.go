// internal/engine/render_test.go
package engine

import (
	"testing"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

// renderChart renders an in-memory chart (umbrella + subchart deps) with default
// values and returns a map of rendered-template-path -> manifest YAML.
func renderChart(t *testing.T, ch *chart.Chart) map[string]string {
	t.Helper()
	vals, err := chartutil.ToRenderValues(ch, chartutil.Values{}, chartutil.ReleaseOptions{
		Name:      "rel",
		Namespace: "default",
	}, nil)
	if err != nil {
		t.Fatalf("ToRenderValues: %v", err)
	}
	out, err := engine.Render(ch, vals)
	if err != nil {
		t.Fatalf("engine.Render: %v", err)
	}
	return out
}

// allManifests concatenates every rendered manifest into one string for substring
// assertions.
func allManifests(m map[string]string) string {
	var b string
	for _, v := range m {
		b += v + "\n"
	}
	return b
}
