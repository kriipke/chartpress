// internal/engine/annotations_test.go
package engine

import (
	"strings"
	"testing"
)

func TestCommonAnnotationsMergedOntoResources(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             func() Rules { r := DefaultRules(); r.CommonAnnotations = true; return r }(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	man := allManifests(renderChart(t, ch))
	if !strings.Contains(man, "app.kubernetes.io/part-of: demo") {
		t.Fatalf("expected part-of annotation, got:\n%s", man)
	}
	if !strings.Contains(man, "chartpress.dev/managed:") {
		t.Fatalf("expected managed annotation, got:\n%s", man)
	}
}

func TestCommonAnnotationsAbsentByDefault(t *testing.T) {
	ch, _ := BuildChart(basicSpec(), testdataTemplates)
	man := allManifests(renderChart(t, ch))
	if strings.Contains(man, "chartpress.dev/managed") {
		t.Fatalf("managed annotation should be absent by default")
	}
}
