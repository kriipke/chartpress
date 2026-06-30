// internal/engine/engine_test.go
package engine

import (
	"strings"
	"testing"
)

// testdataTemplates points the engine at the repo's real template sources.
const testdataTemplates = "../../templates"

func basicSpec() Spec {
	return Normalize(Spec{
		UmbrellaChartName: "demo",
		Description:       "demo umbrella",
		Subcharts: []Subchart{
			{Name: "api", Workload: "deployment"},
			{Name: "cache", Workload: "deployment"},
		},
		Rules: DefaultRules(),
	})
}

func TestBuildChartBasic(t *testing.T) {
	ch, err := BuildChart(basicSpec(), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	if ch.Metadata.Name != "demo" {
		t.Fatalf("umbrella name = %q, want demo", ch.Metadata.Name)
	}
	if ch.Metadata.Description != "demo umbrella" {
		t.Fatalf("umbrella description = %q", ch.Metadata.Description)
	}
	if len(ch.Dependencies()) != 2 {
		t.Fatalf("want 2 subcharts, got %d", len(ch.Dependencies()))
	}
	man := allManifests(renderChart(t, ch))
	if strings.Count(man, "kind: Deployment") != 2 {
		t.Fatalf("expected 2 Deployments, got:\n%s", man)
	}
}
