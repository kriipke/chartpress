// internal/engine/naming_test.go
package engine

import (
	"strings"
	"testing"
)

func TestResourceNamesMatchChartName(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             func() Rules { r := DefaultRules(); r.ResourceNamesMatchChartName = true; return r }(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	man := allManifests(renderChart(t, ch))
	if !strings.Contains(man, "name: api\n") {
		t.Fatalf("expected resource named exactly 'api', got:\n%s", man)
	}
	if strings.Contains(man, "name: rel-api") {
		t.Fatalf("release-prefixed name should not appear when rule is on:\n%s", man)
	}
}
