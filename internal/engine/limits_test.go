// internal/engine/limits_test.go
package engine

import (
	"strings"
	"testing"
)

// TestStatefulSetSubchartPhase1Limit documents and locks the known Phase-1
// limitation: env/envFrom wiring from applySharedNewrelicSubchart is written
// into .Values.env/.Values.envFrom, which only deployment.tpl consumes.
// statefulset.tpl has no env/envFrom blocks, so StatefulSet subcharts do NOT
// receive NEW_RELIC_APP_NAME or any other env var wired this way in Phase 1.
// The umbrella-level ConfigMap and Secret still render correctly.
// This test asserts current behavior; revisit in Phase 2.
func TestStatefulSetSubchartPhase1Limit(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "db", Workload: "statefulset"}},
		Rules:             func() Rules { r := DefaultRules(); r.SharedNewrelicConfig = true; return r }(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	man := allManifests(renderChart(t, ch))

	// Umbrella resources MUST render regardless of subchart workload type.
	if !strings.Contains(man, "demo-newrelic-config") {
		t.Fatalf("expected umbrella demo-newrelic-config ConfigMap, got:\n%s", man)
	}
	if !strings.Contains(man, "demo-newrelic-license") {
		t.Fatalf("expected umbrella demo-newrelic-license Secret, got:\n%s", man)
	}

	// KNOWN Phase-1 limitation: the StatefulSet manifest does NOT carry
	// NEW_RELIC_APP_NAME because statefulset.tpl has no env block.
	// This assertion locks current behavior; it must be revisited in Phase 2
	// when env wiring is extended to StatefulSet/DaemonSet workloads.
	if strings.Contains(man, "NEW_RELIC_APP_NAME") {
		t.Fatalf("Phase-1 limit violated: StatefulSet subchart should NOT carry NEW_RELIC_APP_NAME (env wiring is deployment-only); update this test in Phase 2")
	}
}
