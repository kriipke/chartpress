// internal/engine/linked_test.go
package engine

import (
	"strings"
	"testing"
)

func TestLinkedFalseInlinesUmbrellaDefines(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             func() Rules { r := DefaultRules(); r.LinkedTemplates = false; return r }(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	var helpers string
	for _, d := range ch.Dependencies() {
		for _, tf := range d.Templates {
			if tf.Name == "templates/_helpers.tpl" {
				helpers = string(tf.Data)
			}
		}
	}
	if !strings.Contains(helpers, `define "umbrella-chart.deployment"`) {
		t.Fatalf("expected umbrella deployment define inlined into subchart helpers:\n%s", helpers)
	}
	// still renders correctly
	man := allManifests(renderChart(t, ch))
	if !strings.Contains(man, "kind: Deployment") {
		t.Fatalf("inlined subchart should still render a Deployment:\n%s", man)
	}
}

func TestLinkedTrueDoesNotInline(t *testing.T) {
	ch, _ := BuildChart(basicSpec(), testdataTemplates) // defaults linked=true
	for _, d := range ch.Dependencies() {
		for _, tf := range d.Templates {
			if tf.Name == "templates/_helpers.tpl" && strings.Contains(string(tf.Data), `define "umbrella-chart.deployment"`) {
				t.Fatalf("linked=true should not inline umbrella defines")
			}
		}
	}
}
