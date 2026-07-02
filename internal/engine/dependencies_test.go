// internal/engine/dependencies_test.go
package engine

import (
	"strings"
	"testing"
)

func depSpec(deps ...string) Spec {
	s := patternSpec(Subchart{Name: "orders-api", Pattern: "api-microservice"})
	s.Dependencies = deps
	return Normalize(s)
}

func TestKnownDependencyEmitted(t *testing.T) {
	ch, err := BuildChart(depSpec("postgresql"), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	// Chart.yaml dependency entry (alias = key), pinned from the registry.
	var found bool
	for _, d := range ch.Metadata.Dependencies {
		if d.Alias == "postgresql" {
			found = true
			if d.Repository == "" || d.Version == "" || d.Version == "TODO" {
				t.Fatalf("known dependency must be pinned, got %+v", d)
			}
		}
	}
	if !found {
		t.Fatalf("postgresql dependency entry missing from Chart.yaml")
	}
	// Values skeleton under the key, with a review-version TODO.
	vals := rawValues(ch)
	if !strings.Contains(vals, "postgresql: {}") {
		t.Fatalf("values skeleton missing for postgresql:\n%s", vals)
	}
	if !strings.Contains(vals, "review the pinned version") {
		t.Fatalf("dependency values must carry the review-version TODO:\n%s", vals)
	}
	// No generated subchart for the dependency.
	for _, d := range ch.Dependencies() {
		if d.Name() == "postgresql" {
			t.Fatalf("dependency must not be a generated subchart")
		}
	}
	// HANDOFF platform item.
	if !strings.Contains(umbrellaFile(ch, "HANDOFF.md"), "Dependency \"postgresql\"") {
		t.Fatalf("HANDOFF.md missing the dependency checklist item")
	}
}

func TestUnknownDependencyStub(t *testing.T) {
	ch, err := BuildChart(depSpec("cockroachdb"), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	var d *depFound
	for _, dep := range ch.Metadata.Dependencies {
		if dep.Name == "cockroachdb" {
			d = &depFound{dep.Version, dep.Repository}
		}
	}
	if d == nil {
		t.Fatalf("unknown dependency should still emit a stub entry")
	}
	if d.version != "TODO" {
		t.Fatalf("unknown dependency version should be a TODO marker, got %q", d.version)
	}
	if !strings.Contains(rawValues(ch), "cockroachdb: {}") {
		t.Fatalf("unknown dependency values stub missing")
	}
	if !strings.Contains(umbrellaFile(ch, "HANDOFF.md"), "choose the upstream chart") {
		t.Fatalf("unknown dependency HANDOFF item should say to choose the chart")
	}
}

type depFound struct{ version, repository string }

func TestNoDependenciesNoChange(t *testing.T) {
	ch, err := BuildChart(patternSpec(Subchart{Name: "api"}), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	if strings.Contains(rawValues(ch), "infrastructure dependencies") {
		t.Fatalf("no dependencies should mean no dependency values block")
	}
}

func TestDependencyKeysSorted(t *testing.T) {
	keys := DependencyKeys()
	if len(keys) == 0 {
		t.Fatalf("registry should not be empty")
	}
	for i := 1; i < len(keys); i++ {
		if keys[i-1] > keys[i] {
			t.Fatalf("DependencyKeys not sorted: %v", keys)
		}
	}
}
