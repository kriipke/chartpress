// internal/engine/newrelic_test.go
package engine

import (
	"strings"
	"testing"
)

func TestSharedNewrelicEmitsConfigSecretAndAppName(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             func() Rules { r := DefaultRules(); r.SharedNewrelicConfig = true; return r }(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	man := allManifests(renderChart(t, ch))
	for _, want := range []string{
		"demo-newrelic-config", "demo-newrelic-license",
		"NEW_RELIC_LICENSE_KEY", "NEW_RELIC_APP_NAME", "value: api",
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("missing %q in:\n%s", want, man)
		}
	}
}
