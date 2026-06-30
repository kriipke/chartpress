// internal/engine/workload_test.go
package engine

import (
	"strings"
	"testing"
)

func TestWorkloadSelection(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts: []Subchart{
			{Name: "api", Workload: "deployment"},
			{Name: "db", Workload: "statefulset"},
			{Name: "agent", Workload: "daemonset"},
		},
		Rules: DefaultRules(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	man := allManifests(renderChart(t, ch))
	for _, want := range []string{"kind: Deployment", "kind: StatefulSet", "kind: DaemonSet"} {
		if !strings.Contains(man, want) {
			t.Fatalf("missing %q in:\n%s", want, man)
		}
	}
	if strings.Count(man, "kind: Deployment") != 1 {
		t.Fatalf("expected exactly 1 Deployment, got:\n%s", man)
	}
}
