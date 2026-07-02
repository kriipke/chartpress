package compose

import (
	"strings"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
)

func mustSpec(t *testing.T, yaml string) (engine.Spec, []string) {
	t.Helper()
	spec, notes, err := ToSpec([]byte(yaml))
	if err != nil {
		t.Fatalf("ToSpec: %v", err)
	}
	return spec, notes
}

func findSub(spec engine.Spec, name string) (engine.Subchart, bool) {
	for _, s := range spec.Subcharts {
		if s.Name == name {
			return s, true
		}
	}
	return engine.Subchart{}, false
}

func hasDep(spec engine.Spec, key string) bool {
	for _, d := range spec.Dependencies {
		if d == key {
			return true
		}
	}
	return false
}

func notesContain(notes []string, sub string) bool {
	for _, n := range notes {
		if strings.Contains(n, sub) {
			return true
		}
	}
	return false
}

// TestImageMapTargetsAreKnownDependencies pins the single-source-of-truth
// invariant: every image the mapper routes to a dependency must be a real key
// in engine's dependency registry, or /generate would emit a TODO stub.
func TestImageMapTargetsAreKnownDependencies(t *testing.T) {
	known := map[string]bool{}
	for _, k := range engine.DependencyKeys() {
		known[k] = true
	}
	for img, key := range imageBaseToKey {
		if !known[key] {
			t.Errorf("image %q maps to %q which is not a known dependency key %v", img, key, engine.DependencyKeys())
		}
	}
}

func TestBuildServiceBecomesSubchart(t *testing.T) {
	spec, _ := mustSpec(t, `
services:
  api:
    build: .
    ports: ["8080:8080"]
`)
	sc, ok := findSub(spec, "api")
	if !ok {
		t.Fatalf("expected subchart 'api', got %+v", spec.Subcharts)
	}
	if sc.Pattern != "api-microservice" {
		t.Errorf("pattern = %q, want api-microservice", sc.Pattern)
	}
	if sc.Port != 8080 {
		t.Errorf("port = %d, want 8080", sc.Port)
	}
}

func TestKnownImageBecomesDependency(t *testing.T) {
	spec, _ := mustSpec(t, `
services:
  db:
    image: postgres:16
  cache:
    image: bitnami/redis:7.2
  api:
    build: .
`)
	if !hasDep(spec, "postgresql") {
		t.Errorf("expected postgresql dependency, deps=%v", spec.Dependencies)
	}
	if !hasDep(spec, "redis") {
		t.Errorf("expected redis dependency (bitnami variant), deps=%v", spec.Dependencies)
	}
	if _, ok := findSub(spec, "db"); ok {
		t.Errorf("db should be a dependency, not a subchart")
	}
	if _, ok := findSub(spec, "cache"); ok {
		t.Errorf("cache should be a dependency, not a subchart")
	}
	if _, ok := findSub(spec, "api"); !ok {
		t.Errorf("api should remain a subchart")
	}
}

func TestUnknownImageBecomesFlaggedSubchart(t *testing.T) {
	spec, notes := mustSpec(t, `
services:
  mail:
    image: mailhog/mailhog:latest
`)
	if _, ok := findSub(spec, "mail"); !ok {
		t.Fatalf("unknown image should become a subchart, got %+v", spec.Subcharts)
	}
	if !notesContain(notes, "mailhog/mailhog") {
		t.Errorf("expected a review note about the unrecognized image, notes=%v", notes)
	}
}

func TestNoPortsIsWorker(t *testing.T) {
	spec, _ := mustSpec(t, `
services:
  worker:
    build: ./worker
`)
	sc, _ := findSub(spec, "worker")
	if sc.Pattern != "worker" {
		t.Errorf("pattern = %q, want worker", sc.Pattern)
	}
	if sc.Port != 0 {
		t.Errorf("port = %d, want 0", sc.Port)
	}
}

func TestMultiplePortsKeepsFirstWithNote(t *testing.T) {
	spec, notes := mustSpec(t, `
services:
  web:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"
`)
	sc, _ := findSub(spec, "web")
	if sc.Port != 8080 {
		t.Errorf("port = %d, want 8080 (first)", sc.Port)
	}
	if !notesContain(notes, "multiple ports") {
		t.Errorf("expected a multiple-ports note, notes=%v", notes)
	}
}

func TestDeployGlobalIsDaemonset(t *testing.T) {
	spec, _ := mustSpec(t, `
services:
  agent:
    build: .
    deploy:
      mode: global
`)
	sc, _ := findSub(spec, "agent")
	if sc.Workload != "daemonset" {
		t.Errorf("workload = %q, want daemonset", sc.Workload)
	}
}

func TestDeployReplicasFixesScaling(t *testing.T) {
	spec, _ := mustSpec(t, `
services:
  api:
    build: .
    ports: ["80:80"]
    deploy:
      replicas: 3
`)
	sc, _ := findSub(spec, "api")
	if sc.Scaling != "fixed" {
		t.Errorf("scaling = %q, want fixed", sc.Scaling)
	}
}

func TestNameNormalizationNoted(t *testing.T) {
	spec, notes := mustSpec(t, `
services:
  User_API:
    build: .
`)
	if _, ok := findSub(spec, "user-api"); !ok {
		t.Fatalf("expected normalized subchart 'user-api', got %+v", spec.Subcharts)
	}
	if !notesContain(notes, "Renamed service") {
		t.Errorf("expected a rename note, notes=%v", notes)
	}
}

func TestTopLevelNameBecomesUmbrella(t *testing.T) {
	spec, _ := mustSpec(t, `
name: my-platform
services:
  api:
    build: .
`)
	if spec.UmbrellaChartName != "my-platform" {
		t.Errorf("umbrella = %q, want my-platform", spec.UmbrellaChartName)
	}
}

func TestAllInfrastructureNoted(t *testing.T) {
	spec, notes := mustSpec(t, `
services:
  db:
    image: postgres:16
  cache:
    image: redis:7
`)
	if len(spec.Subcharts) != 0 {
		t.Errorf("expected zero subcharts, got %+v", spec.Subcharts)
	}
	if !hasDep(spec, "postgresql") || !hasDep(spec, "redis") {
		t.Errorf("expected both deps, deps=%v", spec.Dependencies)
	}
	if !notesContain(notes, "No application services") {
		t.Errorf("expected an all-infrastructure note, notes=%v", notes)
	}
}

func TestDuplicateNormalizedNameDeduped(t *testing.T) {
	spec, notes := mustSpec(t, `
services:
  user_api:
    build: ./a
  user-api:
    build: ./b
`)
	if _, ok := findSub(spec, "user-api"); !ok {
		t.Fatalf("expected 'user-api', got %+v", spec.Subcharts)
	}
	if _, ok := findSub(spec, "user-api-2"); !ok {
		t.Fatalf("expected deduped 'user-api-2', got %+v", spec.Subcharts)
	}
	if !notesContain(notes, "user-api-2") {
		t.Errorf("expected a dedup note, notes=%v", notes)
	}
}

func TestDeterministicSubchartOrder(t *testing.T) {
	yaml := `
services:
  zeta:
    build: .
  alpha:
    build: .
  mu:
    build: .
`
	spec, _ := mustSpec(t, yaml)
	got := []string{}
	for _, s := range spec.Subcharts {
		got = append(got, s.Name)
	}
	want := []string{"alpha", "mu", "zeta"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestRejectsEmptyAndInvalid(t *testing.T) {
	if _, _, err := ToSpec([]byte("   ")); err == nil {
		t.Error("empty input should error")
	}
	if _, _, err := ToSpec([]byte("services: [unclosed")); err == nil {
		t.Error("invalid YAML should error")
	}
	if _, _, err := ToSpec([]byte(`version: "3.8"`)); err == nil {
		t.Error("no services should error")
	}
}
