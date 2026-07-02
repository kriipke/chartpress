// internal/engine/artifact_test.go
//
// Regression tests for the saved artifact. Helm's chartutil.Save/SaveDir write
// values.yaml from chart.Raw, so mutations made only to chart.Values render
// fine in-memory but silently vanish from the chart a user downloads. These
// tests reload the chart from disk — exactly what `helm install` sees — and
// render THAT.
package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	helmengine "helm.sh/helm/v3/pkg/engine"
)

// renderSaved generates the chart to disk, reloads it with the standard helm
// loader, and renders it with default values.
func renderSaved(t *testing.T, spec Spec) (string, string) {
	t.Helper()
	out := t.TempDir()
	chartDir, err := GenerateChart(spec, testdataTemplates, out)
	if err != nil {
		t.Fatalf("GenerateChart: %v", err)
	}
	ch, err := loader.Load(chartDir)
	if err != nil {
		t.Fatalf("loader.Load(saved chart): %v", err)
	}
	vals, err := chartutil.ToRenderValues(ch, chartutil.Values{}, chartutil.ReleaseOptions{
		Name: "rel", Namespace: "default",
	}, nil)
	if err != nil {
		t.Fatalf("ToRenderValues: %v", err)
	}
	rendered, err := helmengine.Render(ch, vals)
	if err != nil {
		t.Fatalf("render saved chart: %v", err)
	}
	values, err := os.ReadFile(filepath.Join(chartDir, "values.yaml"))
	if err != nil {
		t.Fatalf("read saved umbrella values.yaml: %v", err)
	}
	return allManifests(rendered), string(values)
}

// TestSavedChartRendersWithAllRules is the golden gate: the chart as saved to
// disk must render with every shared-config rule enabled. Before the Raw/Values
// fix this failed with "nil pointer evaluating interface {}.data".
func TestSavedChartRendersWithAllRules(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts: []Subchart{
			{Name: "api", Workload: "deployment"},
			{Name: "db", Workload: "statefulset"},
			{Name: "agent", Workload: "daemonset"},
		},
		Rules: func() Rules {
			r := DefaultRules()
			r.CommonAnnotations = true
			r.SharedSecretsConfig = true
			r.SharedNewrelicConfig = true
			return r
		}(),
	})
	man, umbrellaValues := renderSaved(t, spec)

	// Shared-config wiring must survive into the saved subchart values.
	for _, want := range []string{
		"demo-shared-secrets",
		"demo-newrelic-config",
		"NEW_RELIC_APP_NAME",
		"app.kubernetes.io/part-of: demo",
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("saved chart render missing %q:\n%s", want, man)
		}
	}

	// The umbrella values.yaml is generated from the spec, not the template
	// repo's example file.
	for _, want := range []string{"api:", "db:", "agent:", "sharedSecrets:", "newrelic:"} {
		if !strings.Contains(umbrellaValues, want) {
			t.Fatalf("umbrella values.yaml missing %q:\n%s", want, umbrellaValues)
		}
	}
	if strings.Contains(umbrellaValues, "job-runner") {
		t.Fatalf("umbrella values.yaml still contains the template repo example:\n%s", umbrellaValues)
	}
}

// TestSavedChartSelectorsMatchPodLabels guards the install-blocking bug where
// workload selectors used an `app:` label that never appeared on pods.
func TestSavedChartSelectorsMatchPodLabels(t *testing.T) {
	man, _ := renderSaved(t, Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             DefaultRules(),
	}))
	if !strings.Contains(man, "app.kubernetes.io/name: api") {
		t.Fatalf("expected selector labels in manifests:\n%s", man)
	}
	// The legacy selector shape (`app: <fullname>`) must be gone everywhere.
	if strings.Contains(man, "\n      app: ") || strings.Contains(man, "\n    app: ") {
		t.Fatalf("legacy app: selector label still present:\n%s", man)
	}
}

// TestSavedStatefulSetHasHeadlessService: a statefulset subchart must ship the
// headless Service its serviceName references.
func TestSavedStatefulSetHasHeadlessService(t *testing.T) {
	man, _ := renderSaved(t, Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "db", Workload: "statefulset"}},
		Rules:             DefaultRules(),
	}))
	if !strings.Contains(man, "serviceName: rel-db-headless") {
		t.Fatalf("statefulset serviceName mismatch:\n%s", man)
	}
	if !strings.Contains(man, "name: rel-db-headless") || !strings.Contains(man, "clusterIP: None") {
		t.Fatalf("headless service missing:\n%s", man)
	}
}

// TestSavedChartHasNoPlaceholderManifests: the old configmap/secrets stubs
// rendered kind-less documents that broke kubectl apply.
func TestSavedChartHasNoPlaceholderManifests(t *testing.T) {
	man, _ := renderSaved(t, Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             DefaultRules(),
	}))
	for _, stale := range []string{"sample: data", "secret: data"} {
		if strings.Contains(man, stale) {
			t.Fatalf("placeholder manifest %q still rendered:\n%s", stale, man)
		}
	}
}
