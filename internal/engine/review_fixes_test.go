// internal/engine/review_fixes_test.go
//
// Regression tests for issues raised in code review of the Phase 1 engine.
package engine

import (
	"strings"
	"testing"
)

// TestPlaceholderReplacementHandlesOverlappingTokens guards against the
// map-iteration-order bug in replacePlaceholders: an umbrella name that embeds
// the "component" placeholder (e.g. "my-component", a valid name) must not be
// re-substituted. The subchart's `include "umbrella-chart.deployment"` must
// resolve to the umbrella's renamed `my-component.deployment`, not a corrupted
// `my-api.deployment`. With the old unordered ReplaceAll loop this rendered
// incorrectly (or errored) depending on Go map iteration order.
func TestPlaceholderReplacementHandlesOverlappingTokens(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "my-component",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             DefaultRules(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	man := allManifests(renderChart(t, ch))
	if !strings.Contains(man, "kind: Deployment") {
		t.Fatalf("subchart include should resolve to the umbrella define despite the overlapping 'component' token:\n%s", man)
	}
}

// TestEnvVarWithEmptyValueIsPreserved guards against silently dropping an env
// var whose value is the empty string. The deployment/statefulset/daemonset env
// block must key off the PRESENCE of the `value` key (hasKey), not its
// truthiness, so `{ name: FLAG, value: "" }` still renders.
func TestEnvVarWithEmptyValueIsPreserved(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             DefaultRules(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	// Inject a user-style env entry with an explicit empty value onto the subchart.
	for _, d := range ch.Dependencies() {
		d.Values["env"] = []interface{}{
			map[string]interface{}{"name": "OPTIONAL_FLAG", "value": ""},
		}
	}
	man := allManifests(renderChart(t, ch))
	if !strings.Contains(man, "OPTIONAL_FLAG") {
		t.Fatalf("env var with empty value should be preserved, not dropped:\n%s", man)
	}
}
