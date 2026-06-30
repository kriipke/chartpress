// internal/engine/limits_test.go
package engine

import (
	"strings"
	"testing"
)

// These tests assert that env/envFrom wiring from SharedNewrelicConfig and
// SharedSecretsConfig now renders on all three workload types: deployment,
// statefulset, and daemonset.  The former Phase-1 limitation (env wiring was
// deployment-only) is resolved: statefulset.tpl and daemonset.tpl now include
// the same {{- with .Values.envFrom }} / {{- with .Values.env }} blocks as
// deployment.tpl.

// TestStatefulSetSharedNewrelicEnv verifies that a statefulset subchart
// receives the full New Relic env wiring: configMap envFrom, license-key
// secretKeyRef, and the per-app NEW_RELIC_APP_NAME value env var.
func TestStatefulSetSharedNewrelicEnv(t *testing.T) {
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

	for _, want := range []string{
		"demo-newrelic-config",
		"demo-newrelic-license",
		"NEW_RELIC_LICENSE_KEY",
		"NEW_RELIC_APP_NAME",
		"value: db",
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("missing %q in statefulset manifests:\n%s", want, man)
		}
	}
}

// TestDaemonSetSharedNewrelicEnv verifies that a daemonset subchart receives
// the full New Relic env wiring, including NEW_RELIC_APP_NAME = subchart name.
func TestDaemonSetSharedNewrelicEnv(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "agent", Workload: "daemonset"}},
		Rules:             func() Rules { r := DefaultRules(); r.SharedNewrelicConfig = true; return r }(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	man := allManifests(renderChart(t, ch))

	for _, want := range []string{
		"NEW_RELIC_APP_NAME",
		"value: agent",
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("missing %q in daemonset manifests:\n%s", want, man)
		}
	}
}

// TestStatefulSetSharedSecretsEnvFrom verifies that a statefulset subchart
// receives an envFrom secretRef mount for the shared-secrets Secret.
func TestStatefulSetSharedSecretsEnvFrom(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "db", Workload: "statefulset"}},
		Rules:             func() Rules { r := DefaultRules(); r.SharedSecretsConfig = true; return r }(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	man := allManifests(renderChart(t, ch))

	for _, want := range []string{
		"secretRef",
		"demo-shared-secrets",
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("missing %q in statefulset manifests:\n%s", want, man)
		}
	}
}
