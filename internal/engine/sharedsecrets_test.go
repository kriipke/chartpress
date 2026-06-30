// internal/engine/sharedsecrets_test.go
package engine

import (
	"strings"
	"testing"
)

func TestSharedSecretsEmitsSecretAndEnvFrom(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             func() Rules { r := DefaultRules(); r.SharedSecretsConfig = true; return r }(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	man := allManifests(renderChart(t, ch))
	if !strings.Contains(man, "kind: Secret") || !strings.Contains(man, "demo-shared-secrets") {
		t.Fatalf("expected demo-shared-secrets Secret, got:\n%s", man)
	}
	if !strings.Contains(man, "secretRef") || !strings.Contains(man, "demo-shared-secrets") {
		t.Fatalf("expected envFrom secretRef on subchart, got:\n%s", man)
	}
}
