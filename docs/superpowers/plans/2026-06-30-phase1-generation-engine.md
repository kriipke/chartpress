# Phase 1 — Generation Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a new, self-contained `internal/engine` package that turns a rich chartpress **spec** into a Helm umbrella chart honoring all six rules, the three workloads, file toggles, and descriptions — proven by in-process golden-render tests.

**Architecture:** Pure, additive Go package. `BuildChart(spec, templatesDir) (*chart.Chart, error)` loads the `templates/umbrella` + `templates/subchart` sources, mutates the in-memory `*chart.Chart` (templates, values, metadata, files) per the spec's rules using **Go-side template generation**, and returns the assembled umbrella chart. `GenerateChart(spec, templatesDir, outputRoot) (string, error)` = `BuildChart` + `chartutil.SaveDir`. Tests render the in-memory chart with the Helm engine and assert on the output. Touches **no** existing file (`internal/server`, `cmd/`) — those keep working until Phase 2 repoints them.

**Tech Stack:** Go 1.23+, `helm.sh/helm/v3` (`pkg/chart`, `pkg/chart/loader`, `pkg/chartutil`, `pkg/engine` — already in `go.mod`), `sigs.k8s.io/yaml` / `gopkg.in/yaml.v2`, standard `testing`.

## Global Constraints

- Go module: `github.com/kriipke/chartpress`, Go `1.23.0` (toolchain `go1.23.6`). API group is **`chartpress.dev`** (not `kriipke.dev`).
- Scaffold-only: the engine sets **structure + rule-driven resources**; it never injects concrete per-component values (images/ports/hosts) — `values.yaml` stays the template defaults.
- Allowed workloads: `deployment`, `statefulset`, `daemonset` only (reject `job`/`cronjob`).
- `rules.ingress` is a single string ∈ `{alb, nginx, traefik, istio, gce, none}`.
- Rule defaults: `ingress: "alb"`, `linked_templates: true`, `generate_umbrella_readme: true`, `generate_subchart_readme: true`, `include_docs: true`; everything else `false`.
- Name regex (chart + subchart): `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`.
- Common-annotations seed keys: `app.kubernetes.io/part-of: <umbrella>`, `chartpress.dev/managed: "true"`.
- Shared-resource names: Secret `<umbrella>-shared-secrets`; NewRelic `ConfigMap <umbrella>-newrelic-config`, `Secret <umbrella>-newrelic-license`.
- No new third-party deps in this phase (engine uses only what `go.mod` already has).

## File Structure

- `internal/engine/types.go` — `Spec`, `Subchart`, `Rules`; `DefaultRules()`, `Normalize`, `Validate`.
- `internal/engine/engine.go` — `BuildChart`, `GenerateChart`, and the chart-load/rename/subchart helpers (refactored copies of the server's, so the package is self-contained).
- `internal/engine/rules.go` — the six rule transforms + workload selection + file toggles + descriptions, each a small `func(...) error` operating on `*chart.Chart`.
- `internal/engine/ingress.go` — ingress controller rendering (alb/nginx/traefik/gce + istio Gateway/VirtualService).
- `internal/engine/render_test.go` — the in-process render helper shared by golden tests.
- `internal/engine/*_test.go` — one test file per task area.

Each rule transform has one responsibility and is independently testable.

---

### Task 1: Package scaffold — Spec types, defaults, normalize, validate

**Files:**
- Create: `internal/engine/types.go`
- Test: `internal/engine/types_test.go`

**Interfaces:**
- Produces: `Spec`, `Subchart`, `Rules` structs; `DefaultRules() Rules`; `Normalize(Spec) Spec`; `Validate(Spec) error`; `var AllowedWorkloads = []string{"deployment","statefulset","daemonset"}`; `var AllowedIngress = []string{"alb","nginx","traefik","istio","gce","none"}`.

- [ ] **Step 1: Write the failing test**

```go
// internal/engine/types_test.go
package engine

import "testing"

func TestDefaultRules(t *testing.T) {
	r := DefaultRules()
	if r.Ingress != "alb" || !r.LinkedTemplates || !r.GenerateUmbrellaReadme ||
		!r.GenerateSubchartReadme || !r.IncludeDocs {
		t.Fatalf("unexpected defaults: %+v", r)
	}
	if r.CommonAnnotations || r.ResourceNamesMatchChartName || r.SharedSecretsConfig || r.SharedNewrelicConfig {
		t.Fatalf("non-default booleans should be false: %+v", r)
	}
}

func TestNormalizeLowercasesAndDefaultsIngress(t *testing.T) {
	got := Normalize(Spec{
		UmbrellaChartName: "  Demo-Platform ",
		Subcharts:         []Subchart{{Name: " API ", Workload: "Deployment"}},
		Rules:             Rules{Ingress: ""},
	})
	if got.UmbrellaChartName != "demo-platform" {
		t.Fatalf("umbrella not sanitized: %q", got.UmbrellaChartName)
	}
	if got.Subcharts[0].Name != "api" || got.Subcharts[0].Workload != "deployment" {
		t.Fatalf("subchart not normalized: %+v", got.Subcharts[0])
	}
	if got.Rules.Ingress != "alb" {
		t.Fatalf("empty ingress should default to alb, got %q", got.Rules.Ingress)
	}
}

func TestValidateRejectsBadInput(t *testing.T) {
	cases := []struct {
		name string
		spec Spec
	}{
		{"empty name", Spec{Subcharts: []Subchart{{Name: "api", Workload: "deployment"}}, Rules: DefaultRules()}},
		{"no subcharts", Spec{UmbrellaChartName: "demo", Rules: DefaultRules()}},
		{"bad workload", Spec{UmbrellaChartName: "demo", Subcharts: []Subchart{{Name: "api", Workload: "job"}}, Rules: DefaultRules()}},
		{"bad ingress", Spec{UmbrellaChartName: "demo", Subcharts: []Subchart{{Name: "api", Workload: "deployment"}}, Rules: Rules{Ingress: "kong"}}},
		{"bad name chars", Spec{UmbrellaChartName: "Demo_Platform", Subcharts: []Subchart{{Name: "api", Workload: "deployment"}}, Rules: DefaultRules()}},
	}
	for _, c := range cases {
		if err := Validate(c.spec); err == nil {
			t.Errorf("%s: expected error, got nil", c.name)
		}
	}
}

func TestValidateAcceptsGoodInput(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             DefaultRules(),
	})
	if err := Validate(spec); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run 'TestDefaultRules|TestNormalize|TestValidate' -v`
Expected: FAIL — `undefined: DefaultRules` (package/types not created yet).

- [ ] **Step 3: Write minimal implementation**

```go
// internal/engine/types.go
package engine

import (
	"fmt"
	"regexp"
	"strings"
)

// Spec is the chartpress generation spec — the body of /generate and the .spec
// of a ChartpressConfig manifest.
type Spec struct {
	UmbrellaChartName string     `json:"umbrellaChartName" yaml:"umbrellaChartName"`
	Description       string     `json:"description,omitempty" yaml:"description,omitempty"`
	Subcharts         []Subchart `json:"subcharts" yaml:"subcharts"`
	Rules             Rules      `json:"rules" yaml:"rules"`
}

type Subchart struct {
	Name        string `json:"name" yaml:"name"`
	Workload    string `json:"workload" yaml:"workload"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type Rules struct {
	Ingress                     string `json:"ingress" yaml:"ingress"`
	CommonAnnotations           bool   `json:"common_annotations" yaml:"common_annotations"`
	LinkedTemplates             bool   `json:"linked_templates" yaml:"linked_templates"`
	ResourceNamesMatchChartName bool   `json:"resource_names_match_chart_name" yaml:"resource_names_match_chart_name"`
	SharedSecretsConfig         bool   `json:"shared_secrets_config" yaml:"shared_secrets_config"`
	SharedNewrelicConfig        bool   `json:"shared_newrelic_config" yaml:"shared_newrelic_config"`
	GenerateUmbrellaReadme      bool   `json:"generate_umbrella_readme" yaml:"generate_umbrella_readme"`
	GenerateSubchartReadme      bool   `json:"generate_subchart_readme" yaml:"generate_subchart_readme"`
	IncludeDocs                 bool   `json:"include_docs" yaml:"include_docs"`
}

var (
	AllowedWorkloads = []string{"deployment", "statefulset", "daemonset"}
	AllowedIngress   = []string{"alb", "nginx", "traefik", "istio", "gce", "none"}
	nameRE           = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
)

// DefaultRules returns the locked rule defaults.
func DefaultRules() Rules {
	return Rules{
		Ingress:                "alb",
		LinkedTemplates:        true,
		GenerateUmbrellaReadme: true,
		GenerateSubchartReadme: true,
		IncludeDocs:            true,
	}
}

func sanitizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// Normalize trims/lowercases names and workloads and fills an empty ingress with
// the default. It does NOT fill omitted booleans (that is the decode layer's job
// in Phase 2); the engine receives explicit rules.
func Normalize(s Spec) Spec {
	s.UmbrellaChartName = sanitizeName(s.UmbrellaChartName)
	s.Description = strings.TrimSpace(s.Description)
	for i := range s.Subcharts {
		s.Subcharts[i].Name = sanitizeName(s.Subcharts[i].Name)
		s.Subcharts[i].Workload = sanitizeName(s.Subcharts[i].Workload)
		s.Subcharts[i].Description = strings.TrimSpace(s.Subcharts[i].Description)
	}
	s.Rules.Ingress = strings.ToLower(strings.TrimSpace(s.Rules.Ingress))
	if s.Rules.Ingress == "" {
		s.Rules.Ingress = "alb"
	}
	return s
}

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// Validate enforces the spec-level invariants (name regex, >=1 subchart, workload
// and ingress enums).
func Validate(s Spec) error {
	if !nameRE.MatchString(s.UmbrellaChartName) {
		return fmt.Errorf("umbrellaChartName %q must match %s", s.UmbrellaChartName, nameRE.String())
	}
	if len(s.Subcharts) == 0 {
		return fmt.Errorf("at least one subchart is required")
	}
	for _, sc := range s.Subcharts {
		if !nameRE.MatchString(sc.Name) {
			return fmt.Errorf("subchart name %q must match %s", sc.Name, nameRE.String())
		}
		if !contains(AllowedWorkloads, sc.Workload) {
			return fmt.Errorf("subchart %q has invalid workload %q (allowed: %v)", sc.Name, sc.Workload, AllowedWorkloads)
		}
	}
	if !contains(AllowedIngress, s.Rules.Ingress) {
		return fmt.Errorf("rules.ingress %q invalid (allowed: %v)", s.Rules.Ingress, AllowedIngress)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run 'TestDefaultRules|TestNormalize|TestValidate' -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/engine/types.go internal/engine/types_test.go
git commit -m "feat(engine): spec types, defaults, normalize, validate"
```

---

### Task 2: Render helper + BuildChart skeleton (load, rename, subcharts, umbrella description)

**Files:**
- Create: `internal/engine/engine.go`, `internal/engine/render_test.go`
- Test: `internal/engine/engine_test.go`

**Interfaces:**
- Consumes: `Spec` (Task 1).
- Produces: `BuildChart(spec Spec, templatesDir string) (*chart.Chart, error)`; `GenerateChart(spec Spec, templatesDir, outputRoot string) (string, error)`; test helper `renderChart(t *testing.T, ch *chart.Chart) map[string]string` (rendered-template-path → manifest text).

- [ ] **Step 1: Write the failing test**

```go
// internal/engine/render_test.go
package engine

import (
	"testing"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

// renderChart renders an in-memory chart (umbrella + subchart deps) with default
// values and returns a map of rendered-template-path -> manifest YAML.
func renderChart(t *testing.T, ch *chart.Chart) map[string]string {
	t.Helper()
	vals, err := chartutil.ToRenderValues(ch, chartutil.Values{}, chartutil.ReleaseOptions{
		Name:      "rel",
		Namespace: "default",
	}, nil)
	if err != nil {
		t.Fatalf("ToRenderValues: %v", err)
	}
	out, err := engine.Render(ch, vals)
	if err != nil {
		t.Fatalf("engine.Render: %v", err)
	}
	return out
}

// allManifests concatenates every rendered manifest into one string for substring
// assertions.
func allManifests(m map[string]string) string {
	var b string
	for _, v := range m {
		b += v + "\n"
	}
	return b
}
```

```go
// internal/engine/engine_test.go
package engine

import (
	"strings"
	"testing"
)

// testdataTemplates points the engine at the repo's real template sources.
const testdataTemplates = "../../templates"

func basicSpec() Spec {
	return Normalize(Spec{
		UmbrellaChartName: "demo",
		Description:       "demo umbrella",
		Subcharts: []Subchart{
			{Name: "api", Workload: "deployment"},
			{Name: "cache", Workload: "deployment"},
		},
		Rules: DefaultRules(),
	})
}

func TestBuildChartBasic(t *testing.T) {
	ch, err := BuildChart(basicSpec(), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	if ch.Metadata.Name != "demo" {
		t.Fatalf("umbrella name = %q, want demo", ch.Metadata.Name)
	}
	if ch.Metadata.Description != "demo umbrella" {
		t.Fatalf("umbrella description = %q", ch.Metadata.Description)
	}
	if len(ch.Dependencies()) != 2 {
		t.Fatalf("want 2 subcharts, got %d", len(ch.Dependencies()))
	}
	man := allManifests(renderChart(t, ch))
	if strings.Count(man, "kind: Deployment") != 2 {
		t.Fatalf("expected 2 Deployments, got:\n%s", man)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestBuildChartBasic -v`
Expected: FAIL — `undefined: BuildChart`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/engine/engine.go
package engine

import (
	"fmt"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
)

const placeholderName = "umbrella-chart" // the literal name inside templates/umbrella

// BuildChart assembles the umbrella chart in memory from spec. templatesDir must
// contain "umbrella" and "subchart" subdirectories.
func BuildChart(spec Spec, templatesDir string) (*chart.Chart, error) {
	umbrellaPath := filepath.Join(templatesDir, "umbrella")
	subchartPath := filepath.Join(templatesDir, "subchart")

	umbrella, err := loader.Load(umbrellaPath)
	if err != nil {
		return nil, fmt.Errorf("load umbrella: %w", err)
	}
	renameChart(umbrella, spec.UmbrellaChartName)
	umbrella.Metadata.Description = chartDescription(spec.Description, spec.UmbrellaChartName)

	for _, sc := range spec.Subcharts {
		sub, err := loader.Load(subchartPath)
		if err != nil {
			return nil, fmt.Errorf("load subchart template: %w", err)
		}
		if err := buildSubchart(sub, sc, spec); err != nil {
			return nil, fmt.Errorf("subchart %q: %w", sc.Name, err)
		}
		umbrella.AddDependency(sub)
		umbrella.Metadata.Dependencies = append(umbrella.Metadata.Dependencies, &chart.Dependency{
			Name:       sc.Name,
			Version:    sub.Metadata.Version,
			Repository: fmt.Sprintf("file://charts/%s", sc.Name),
		})
	}

	if err := applyUmbrellaRules(umbrella, spec); err != nil {
		return nil, err
	}
	return umbrella, nil
}

// GenerateChart builds and writes the chart under outputRoot/<name>, returning the
// chart directory.
func GenerateChart(spec Spec, templatesDir, outputRoot string) (string, error) {
	ch, err := BuildChart(spec, templatesDir)
	if err != nil {
		return "", err
	}
	if err := chartutil.SaveDir(ch, outputRoot); err != nil {
		return "", fmt.Errorf("save chart: %w", err)
	}
	return filepath.Join(outputRoot, spec.UmbrellaChartName), nil
}

func chartDescription(desc, name string) string {
	if strings.TrimSpace(desc) != "" {
		return desc
	}
	return fmt.Sprintf("%s chart generated by chartpress", name)
}

// renameChart replaces the placeholder umbrella name across metadata, templates,
// values, and files.
func renameChart(ch *chart.Chart, newName string) {
	old := ch.Metadata.Name
	ch.Metadata.Name = newName
	for _, t := range ch.Templates {
		t.Data = []byte(strings.ReplaceAll(string(t.Data), old, newName))
	}
	for _, f := range ch.Files {
		f.Data = []byte(strings.ReplaceAll(string(f.Data), old, newName))
	}
}

// buildSubchart renames the subchart and applies per-subchart rules. Rule-specific
// behavior is filled in by later tasks; for now it only renames.
func buildSubchart(sub *chart.Chart, sc Subchart, spec Spec) error {
	sub.Metadata.Name = sc.Name
	sub.Metadata.Description = chartDescription(sc.Description, sc.Name)
	replacePlaceholders(sub, map[string]string{
		placeholderName: spec.UmbrellaChartName,
		"component":     sc.Name,
	})
	return nil
}

func replacePlaceholders(ch *chart.Chart, repl map[string]string) {
	apply := func(b []byte) []byte {
		s := string(b)
		for old, nw := range repl {
			s = strings.ReplaceAll(s, old, nw)
		}
		return []byte(s)
	}
	for _, t := range ch.Templates {
		t.Data = apply(t.Data)
	}
	for _, f := range ch.Files {
		f.Data = apply(f.Data)
	}
}

// applyUmbrellaRules is the umbrella-level rule hook; later tasks add behavior.
func applyUmbrellaRules(ch *chart.Chart, spec Spec) error { return nil }
```

Note: `buildSubchart` already sets the subchart `Chart.yaml` description (covers Task 4's subchart half). Keep Task 4 for the umbrella-description test + the explicit assertion.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestBuildChartBasic -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/engine.go internal/engine/render_test.go internal/engine/engine_test.go
git commit -m "feat(engine): BuildChart/GenerateChart skeleton + in-process render test helper"
```

---

### Task 3: Per-subchart workload selection

**Files:**
- Modify: `internal/engine/engine.go` (call `applyWorkload` from `buildSubchart`)
- Create: `internal/engine/rules.go`
- Test: `internal/engine/workload_test.go`

**Interfaces:**
- Produces: `applyWorkload(sub *chart.Chart, workload string)` — for `statefulset`/`daemonset`, replaces the subchart's `templates/deployment.yaml` with `templates/<workload>.yaml` whose body is `{{ include "umbrella-chart.<workload>" . }}`; `deployment` is a no-op.

- [ ] **Step 1: Write the failing test**

```go
// internal/engine/workload_test.go
package engine

import (
	"strings"
	"testing"
)

func TestWorkloadSelection(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts: []Subchart{
			{Name: "api", Workload: "deployment"},
			{Name: "db", Workload: "statefulset"},
			{Name: "agent", Workload: "daemonset"},
		},
		Rules: DefaultRules(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	man := allManifests(renderChart(t, ch))
	for _, want := range []string{"kind: Deployment", "kind: StatefulSet", "kind: DaemonSet"} {
		if !strings.Contains(man, want) {
			t.Fatalf("missing %q in:\n%s", want, man)
		}
	}
	if strings.Count(man, "kind: Deployment") != 1 {
		t.Fatalf("expected exactly 1 Deployment, got:\n%s", man)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestWorkloadSelection -v`
Expected: FAIL — three Deployments (workload ignored), so `StatefulSet`/`DaemonSet` missing.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/engine/rules.go
package engine

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
)

// applyWorkload swaps the subchart's workload manifest. The subchart template ships
// templates/deployment.yaml = {{ include "umbrella-chart.deployment" . }}; for other
// workloads we drop it and emit templates/<workload>.yaml including the matching
// umbrella named template.
func applyWorkload(sub *chart.Chart, workload string) {
	if workload == "deployment" {
		return
	}
	sub.Templates = dropTemplate(sub.Templates, "templates/deployment.yaml")
	sub.Templates = append(sub.Templates, &chart.File{
		Name: fmt.Sprintf("templates/%s.yaml", workload),
		Data: []byte(fmt.Sprintf("{{ include \"umbrella-chart.%s\" . }}\n", workload)),
	})
}

func dropTemplate(files []*chart.File, name string) []*chart.File {
	out := files[:0]
	for _, f := range files {
		if f.Name != name {
			out = append(out, f)
		}
	}
	return out
}
```

Then wire it in `engine.go` `buildSubchart`, before `replacePlaceholders`:

```go
func buildSubchart(sub *chart.Chart, sc Subchart, spec Spec) error {
	sub.Metadata.Name = sc.Name
	sub.Metadata.Description = chartDescription(sc.Description, sc.Name)
	applyWorkload(sub, sc.Workload)
	replacePlaceholders(sub, map[string]string{
		placeholderName: spec.UmbrellaChartName,
		"component":     sc.Name,
	})
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestWorkloadSelection -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/rules.go internal/engine/engine.go internal/engine/workload_test.go
git commit -m "feat(engine): per-subchart workload template selection"
```

---

### Task 4: Descriptions → Chart.yaml (umbrella + subchart, with fallback)

**Files:**
- Test: `internal/engine/description_test.go`
- (Implementation already in `engine.go` `chartDescription` / `buildSubchart` from Task 2-3; this task adds the regression test and the empty-fallback assertion.)

**Interfaces:**
- Consumes: `BuildChart`, `chartDescription`.

- [ ] **Step 1: Write the failing test**

```go
// internal/engine/description_test.go
package engine

import "testing"

func TestDescriptionsLandInChartYaml(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Description:       "the umbrella",
		Subcharts: []Subchart{
			{Name: "api", Workload: "deployment", Description: "the api"},
			{Name: "cache", Workload: "deployment"}, // empty -> fallback
		},
		Rules: DefaultRules(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	if ch.Metadata.Description != "the umbrella" {
		t.Fatalf("umbrella desc = %q", ch.Metadata.Description)
	}
	deps := map[string]string{}
	for _, d := range ch.Dependencies() {
		deps[d.Metadata.Name] = d.Metadata.Description
	}
	if deps["api"] != "the api" {
		t.Fatalf("api desc = %q", deps["api"])
	}
	if deps["cache"] != "cache chart generated by chartpress" {
		t.Fatalf("cache fallback desc = %q", deps["cache"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails or passes**

Run: `go test ./internal/engine/ -run TestDescriptionsLandInChartYaml -v`
Expected: PASS (implemented in Tasks 2–3). If it FAILS, fix `chartDescription`/`buildSubchart` until green. (Test-after is acceptable here because the behavior was introduced as part of the skeleton; this task locks it with a dedicated regression test.)

- [ ] **Step 3: Commit**

```bash
git add internal/engine/description_test.go
git commit -m "test(engine): lock description -> Chart.yaml behavior with fallback"
```

---

### Task 5: File toggles — readmes and docs

**Files:**
- Modify: `internal/engine/rules.go` (add `applyFileToggles`, call from `BuildChart`/`buildSubchart`)
- Modify: `internal/engine/engine.go`
- Test: `internal/engine/filetoggles_test.go`

**Interfaces:**
- Produces: `dropFile(files []*chart.File, predicate func(name string) bool) []*chart.File`; umbrella drops `README.adoc` and `docs/...`; subchart drops `README.adoc` per the rules.

- [ ] **Step 1: Write the failing test**

```go
// internal/engine/filetoggles_test.go
package engine

import (
	"strings"
	"testing"
)

func hasFile(files []fileLike, name string) bool {
	for _, f := range files {
		if f.GetName() == name {
			return true
		}
	}
	return false
}

func TestFileTogglesStripReadmesAndDocs(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules: Rules{
			Ingress:                "alb",
			LinkedTemplates:        true,
			GenerateUmbrellaReadme: false,
			GenerateSubchartReadme: false,
			IncludeDocs:            false,
		},
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	for _, f := range ch.Files {
		if f.Name == "README.adoc" {
			t.Fatalf("umbrella README.adoc should be stripped")
		}
		if strings.HasPrefix(f.Name, "docs/") {
			t.Fatalf("docs/ should be stripped, found %s", f.Name)
		}
	}
	for _, d := range ch.Dependencies() {
		for _, f := range d.Files {
			if f.Name == "README.adoc" {
				t.Fatalf("subchart README.adoc should be stripped")
			}
		}
	}
}

func TestFileTogglesKeepByDefault(t *testing.T) {
	ch, err := BuildChart(basicSpec(), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	var hasReadme bool
	for _, f := range ch.Files {
		if f.Name == "README.adoc" {
			hasReadme = true
		}
	}
	if !hasReadme {
		t.Fatalf("umbrella README.adoc should be present by default")
	}
}
```

Add the tiny `fileLike` shim used above:

```go
// internal/engine/filetoggles_test.go (same file, top)
type fileLike interface{ GetName() string }
```

(If `*chart.File` lacks `GetName`, drop the `hasFile`/`fileLike` helper and assert directly on `f.Name` as the other tests do — keep only the direct-field assertions.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestFileToggles -v`
Expected: FAIL — README/docs still present.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/engine/rules.go (append)
import "strings" // ensure imported

func dropFile(files []*chart.File, pred func(name string) bool) []*chart.File {
	out := files[:0]
	for _, f := range files {
		if !pred(f.Name) {
			out = append(out, f)
		}
	}
	return out
}

func applyUmbrellaFileToggles(ch *chart.Chart, r Rules) {
	if !r.GenerateUmbrellaReadme {
		ch.Files = dropFile(ch.Files, func(n string) bool { return n == "README.adoc" })
	}
	if !r.IncludeDocs {
		ch.Files = dropFile(ch.Files, func(n string) bool { return strings.HasPrefix(n, "docs/") })
	}
}

func applySubchartFileToggles(sub *chart.Chart, r Rules) {
	if !r.GenerateSubchartReadme {
		sub.Files = dropFile(sub.Files, func(n string) bool { return n == "README.adoc" })
	}
}
```

Wire into `engine.go`: call `applySubchartFileToggles(sub, spec.Rules)` at the end of `buildSubchart`, and `applyUmbrellaFileToggles(ch, spec.Rules)` inside `applyUmbrellaRules`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestFileToggles -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/rules.go internal/engine/engine.go internal/engine/filetoggles_test.go
git commit -m "feat(engine): readme/docs file toggles"
```

---

### Task 6: `resource_names_match_chart_name`

**Files:**
- Modify: `internal/engine/rules.go` (`applyResourceNaming`)
- Modify: `internal/engine/engine.go` (call from `buildSubchart`)
- Test: `internal/engine/naming_test.go`

**Interfaces:**
- Produces: `applyResourceNaming(sub *chart.Chart, match bool)` — when `true`, rewrites the subchart's `<name>.fullname` named template (in `templates/_helpers.tpl`) to emit `{{ .Chart.Name }}`.

The subchart `_helpers.tpl` (after placeholder replacement) defines `<name>.fullname` as `{{- template "umbrella-chart.fullname" . }}-{{ .Chart.Name }}`. We replace its body with `{{ .Chart.Name }}`.

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestResourceNamesMatchChartName -v`
Expected: FAIL — names are release-prefixed (`rel-api-api`).

- [ ] **Step 3: Write minimal implementation**

```go
// internal/engine/rules.go (append)
import "regexp" // ensure imported

// applyResourceNaming rewrites the subchart fullname helper to emit just the chart
// name. The helper define line looks like:
//   {{- define "api.fullname" -}}
//   {{- template "umbrella-chart.fullname" . }}-{{ .Chart.Name }}
//   {{- end }}
func applyResourceNaming(sub *chart.Chart, match bool) {
	if !match {
		return
	}
	def := regexp.MustCompile(`(?s)(\{\{-?\s*define\s+"` + regexp.QuoteMeta(sub.Metadata.Name) + `\.fullname"\s*-?\}\}).*?(\{\{-?\s*end\s*-?\}\})`)
	for _, t := range sub.Templates {
		if t.Name == "templates/_helpers.tpl" {
			t.Data = def.ReplaceAll(t.Data, []byte("${1}\n{{ .Chart.Name }}\n${2}"))
		}
	}
}
```

Wire into `buildSubchart` after `applyWorkload`:

```go
	applyResourceNaming(sub, spec.Rules.ResourceNamesMatchChartName)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestResourceNamesMatchChartName -v`
Expected: PASS. If the regex misses (template whitespace variant), inspect `sub` helper text in the test and adjust the pattern until green.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/rules.go internal/engine/engine.go internal/engine/naming_test.go
git commit -m "feat(engine): resource_names_match_chart_name via fullname helper rewrite"
```

---

### Task 7: `common_annotations`

**Files:**
- Modify: `internal/engine/rules.go` (`applyCommonAnnotations`)
- Modify: `internal/engine/engine.go`
- Test: `internal/engine/annotations_test.go`

**Interfaces:**
- Produces: `applyCommonAnnotations(ch *chart.Chart, spec Spec)` — sets `ch.Values["global"].(map)["commonAnnotations"]` to a seeded map and rewrites the umbrella `umbrella-chart.annotations` named template to append `{{ toYaml .Values.global.commonAnnotations }}` so every resource carries them. (Subcharts inherit `global` from the umbrella at render time.)

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestCommonAnnotations -v`
Expected: FAIL — annotations absent.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/engine/rules.go (append)

// applyCommonAnnotations seeds global.commonAnnotations and makes the umbrella
// annotations helper merge them onto every resource.
func applyCommonAnnotations(ch *chart.Chart, spec Spec) {
	if !spec.Rules.CommonAnnotations {
		return
	}
	global, _ := ch.Values["global"].(map[string]interface{})
	if global == nil {
		global = map[string]interface{}{}
		ch.Values["global"] = global
	}
	global["commonAnnotations"] = map[string]interface{}{
		"app.kubernetes.io/part-of": spec.UmbrellaChartName,
		"chartpress.dev/managed":    "true",
	}
	// Append the merge to the umbrella.annotations named template body.
	for _, t := range ch.Templates {
		if t.Name == "templates/_helpers.tpl" {
			t.Data = appendToAnnotationsDefine(t.Data, ch.Metadata.Name)
		}
	}
}

func appendToAnnotationsDefine(data []byte, chartName string) []byte {
	open := "{{- define \"" + chartName + ".annotations\" -}}"
	s := string(data)
	idx := indexOf(s, open)
	if idx < 0 {
		return data
	}
	insertAt := idx + len(open)
	merge := "\n{{- with .Values.global.commonAnnotations }}\n{{ toYaml . }}\n{{- end }}"
	return []byte(s[:insertAt] + merge + s[insertAt:])
}

func indexOf(s, sub string) int { return strings.Index(s, sub) }
```

Wire into `applyUmbrellaRules`:

```go
func applyUmbrellaRules(ch *chart.Chart, spec Spec) error {
	applyUmbrellaFileToggles(ch, spec.Rules)
	applyCommonAnnotations(ch, spec)
	return nil
}
```

Note on the helper name: the umbrella `_helpers.tpl` defines `umbrella-chart.annotations`, and `renameChart` (Task 2) already rewrote `umbrella-chart` → the umbrella name, so the define is `<name>.annotations`. Use `ch.Metadata.Name` as shown.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestCommonAnnotations -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/rules.go internal/engine/engine.go internal/engine/annotations_test.go
git commit -m "feat(engine): common_annotations merged onto all resources"
```

---

### Task 8: `shared_secrets_config`

**Files:**
- Modify: `internal/engine/rules.go` (`applySharedSecrets`)
- Modify: `internal/engine/engine.go` (umbrella emits the Secret; subcharts get `envFrom`)
- Test: `internal/engine/sharedsecrets_test.go`

**Interfaces:**
- Produces: `applySharedSecretsUmbrella(ch *chart.Chart, spec Spec)` (emits umbrella `templates/shared-secrets.yaml` + seeds `global.sharedSecrets`); `applySharedSecretsSubchart(sub *chart.Chart, spec Spec)` (appends an `envFrom` secretRef to the subchart `values.yaml` so the deployment template mounts it).

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestSharedSecrets -v`
Expected: FAIL — no Secret, no envFrom.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/engine/rules.go (append)

func applySharedSecretsUmbrella(ch *chart.Chart, spec Spec) {
	if !spec.Rules.SharedSecretsConfig {
		return
	}
	name := spec.UmbrellaChartName + "-shared-secrets"
	tmpl := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: " + name +
		"\ntype: Opaque\nstringData:\n{{- range $k, $v := .Values.global.sharedSecrets.data }}\n  {{ $k }}: {{ $v | quote }}\n{{- end }}\n"
	ch.Templates = append(ch.Templates, &chart.File{Name: "templates/shared-secrets.yaml", Data: []byte(tmpl)})

	global, _ := ch.Values["global"].(map[string]interface{})
	if global == nil {
		global = map[string]interface{}{}
		ch.Values["global"] = global
	}
	global["sharedSecrets"] = map[string]interface{}{"data": map[string]interface{}{}}
}

func applySharedSecretsSubchart(sub *chart.Chart, spec Spec) {
	if !spec.Rules.SharedSecretsConfig {
		return
	}
	appendEnvFrom(sub, map[string]interface{}{
		"secretRef": map[string]interface{}{"name": spec.UmbrellaChartName + "-shared-secrets"},
	})
}

// appendEnvFrom adds an entry to the subchart's .Values.envFrom (the deployment
// template already ranges over it).
func appendEnvFrom(sub *chart.Chart, entry map[string]interface{}) {
	cur, _ := sub.Values["envFrom"].([]interface{})
	sub.Values["envFrom"] = append(cur, entry)
}
```

Wire: in `buildSubchart`, call `applySharedSecretsSubchart(sub, spec)`; in `applyUmbrellaRules`, call `applySharedSecretsUmbrella(ch, spec)`.

Note: confirm the umbrella `deployment.tpl` `envFrom` block renders `secretRef`. The shipped template ranges `.Values.envFrom` and handles `.secretRef`/`.configMapRef` — matches this shape.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestSharedSecrets -v`
Expected: PASS. If the deployment `envFrom` block needs a tweak to emit `secretRef` cleanly, adjust the umbrella template copy in this task (engine owns the template sources it ships).

- [ ] **Step 5: Commit**

```bash
git add internal/engine/rules.go internal/engine/engine.go internal/engine/sharedsecrets_test.go
git commit -m "feat(engine): shared_secrets_config umbrella Secret + envFrom"
```

---

### Task 9: `shared_newrelic_config`

**Files:**
- Modify: `internal/engine/rules.go` (`applySharedNewrelicUmbrella`, `applySharedNewrelicSubchart`)
- Modify: `internal/engine/engine.go`
- Test: `internal/engine/newrelic_test.go`

**Interfaces:**
- Produces: umbrella `templates/newrelic-config.yaml` (ConfigMap `<umbrella>-newrelic-config`) + `templates/newrelic-license.yaml` (Secret `<umbrella>-newrelic-license`); per-subchart `envFrom` configMapRef, a `NEW_RELIC_LICENSE_KEY` env via secretKeyRef, and `NEW_RELIC_APP_NAME` = subchart name.

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestSharedNewrelic -v`
Expected: FAIL.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/engine/rules.go (append)

func applySharedNewrelicUmbrella(ch *chart.Chart, spec Spec) {
	if !spec.Rules.SharedNewrelicConfig {
		return
	}
	cfg := spec.UmbrellaChartName + "-newrelic-config"
	lic := spec.UmbrellaChartName + "-newrelic-license"
	cm := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: " + cfg +
		"\ndata:\n  NEW_RELIC_ENABLED: \"true\"\n  NEW_RELIC_DISTRIBUTED_TRACING_ENABLED: \"true\"\n  NEW_RELIC_LABELS: \"app:" + spec.UmbrellaChartName + "\"\n"
	secret := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: " + lic +
		"\ntype: Opaque\nstringData:\n  NEW_RELIC_LICENSE_KEY: {{ .Values.global.newrelic.licenseKey | default \"\" | quote }}\n"
	ch.Templates = append(ch.Templates,
		&chart.File{Name: "templates/newrelic-config.yaml", Data: []byte(cm)},
		&chart.File{Name: "templates/newrelic-license.yaml", Data: []byte(secret)},
	)
	global, _ := ch.Values["global"].(map[string]interface{})
	if global == nil {
		global = map[string]interface{}{}
		ch.Values["global"] = global
	}
	global["newrelic"] = map[string]interface{}{"licenseKey": ""}
}

func applySharedNewrelicSubchart(sub *chart.Chart, spec Spec) {
	if !spec.Rules.SharedNewrelicConfig {
		return
	}
	appendEnvFrom(sub, map[string]interface{}{
		"configMapRef": map[string]interface{}{"name": spec.UmbrellaChartName + "-newrelic-config"},
	})
	// NEW_RELIC_LICENSE_KEY from the shared license secret + per-subchart app name.
	env, _ := sub.Values["env"].([]interface{})
	env = append(env,
		map[string]interface{}{
			"name": "NEW_RELIC_LICENSE_KEY",
			"valueFrom": map[string]interface{}{
				"secretKeyRef": map[string]interface{}{
					"name": spec.UmbrellaChartName + "-newrelic-license",
					"key":  "NEW_RELIC_LICENSE_KEY",
				},
			},
		},
		map[string]interface{}{"name": "NEW_RELIC_APP_NAME", "value": sub.Metadata.Name},
	)
	sub.Values["env"] = env
}
```

Note: the shipped `deployment.tpl` `.Values.env` block assumes `secretKeyRef`/`configMapKeyRef` with `.keys`. For these plain entries (`value:` and a direct `secretKeyRef`), this task **also** updates the engine's copy of `deployment.tpl` to additionally support simple `{name, value}` and `{name, valueFrom}` env entries (append a branch that emits `value:`/`valueFrom:` verbatim via `toYaml`). Write that template change here and assert via the test.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestSharedNewrelic -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/rules.go internal/engine/engine.go internal/engine/newrelic_test.go templates/umbrella/templates/deployment.tpl
git commit -m "feat(engine): shared_newrelic_config configmap+secret+env wiring"
```

---

### Task 10: `ingress` — alb / nginx / traefik / gce / none

**Files:**
- Create: `internal/engine/ingress.go`
- Modify: `internal/engine/engine.go` (call `applyIngress`)
- Test: `internal/engine/ingress_test.go`

**Interfaces:**
- Produces: `applyIngress(sub *chart.Chart, controller string)` — for `none`, removes the subchart `templates/ingress.yaml`; for `alb/nginx/traefik/gce`, replaces it with a controller-specific `Ingress` template (right `ingressClassName` + annotations), gated by `{{- if .Values.ingress.host }}`.

- [ ] **Step 1: Write the failing test**

```go
// internal/engine/ingress_test.go
package engine

import (
	"strings"
	"testing"
)

func ingressSpec(controller string) Spec {
	s := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             func() Rules { r := DefaultRules(); r.Ingress = controller; return r }(),
	})
	return s
}

func renderWithHost(t *testing.T, controller string) string {
	t.Helper()
	ch, err := BuildChart(ingressSpec(controller), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart(%s): %v", controller, err)
	}
	// set a host on the subchart so the ingress guard fires
	for _, d := range ch.Dependencies() {
		d.Values["ingress"] = map[string]interface{}{"host": "api.example.com", "path": "/"}
	}
	return allManifests(renderChart(t, ch))
}

func TestIngressClasses(t *testing.T) {
	cases := map[string]string{
		"alb":     "ingressClassName: alb",
		"nginx":   "ingressClassName: nginx",
		"traefik": "ingressClassName: traefik",
		"gce":     "ingressClassName: gce",
	}
	for ctrl, want := range cases {
		man := renderWithHost(t, ctrl)
		if !strings.Contains(man, "kind: Ingress") || !strings.Contains(man, want) {
			t.Fatalf("%s: expected Ingress with %q, got:\n%s", ctrl, want, man)
		}
	}
}

func TestIngressNoneOmitsIngress(t *testing.T) {
	man := renderWithHost(t, "none")
	if strings.Contains(man, "kind: Ingress") {
		t.Fatalf("ingress=none should emit no Ingress, got:\n%s", man)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestIngress -v`
Expected: FAIL — the shipped `ingress.yaml` includes the `aws/emissary` template, not class-based.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/engine/ingress.go
package engine

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
)

var ingressClassAnnotations = map[string]string{
	"alb":     "    alb.ingress.kubernetes.io/scheme: internet-facing\n    alb.ingress.kubernetes.io/target-type: ip\n",
	"nginx":   "    nginx.ingress.kubernetes.io/rewrite-target: /\n",
	"gce":     "    kubernetes.io/ingress.class: gce\n",
	"traefik": "",
}

func ingressTemplate(controller string) string {
	ann := ingressClassAnnotations[controller]
	return fmt.Sprintf(`{{- if .Values.ingress.host }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
  annotations:
%s    {{- include (print .Chart.Name ".annotations") . | nindent 4 }}
spec:
  ingressClassName: %s
  rules:
    - host: {{ .Values.ingress.host }}
      http:
        paths:
          - path: {{ default "/" .Values.ingress.path }}
            pathType: Prefix
            backend:
              service:
                name: {{ include (print .Chart.Name ".fullname") . }}
                port:
                  number: {{ .Values.service.port }}
{{- end }}
`, ann, controller)
}

// applyIngress replaces the subchart ingress manifest with a controller-specific
// one, or removes it for "none"/"istio" (istio handled separately in Task 11).
func applyIngress(sub *chart.Chart, controller string) {
	sub.Templates = dropTemplate(sub.Templates, "templates/ingress.yaml")
	switch controller {
	case "none", "istio":
		return
	default:
		sub.Templates = append(sub.Templates, &chart.File{
			Name: "templates/ingress.yaml",
			Data: []byte(ingressTemplate(controller)),
		})
	}
}
```

Wire into `buildSubchart`: `applyIngress(sub, spec.Rules.Ingress)`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestIngress -v`
Expected: PASS (TestIngressClasses, TestIngressNoneOmitsIngress).

- [ ] **Step 5: Commit**

```bash
git add internal/engine/ingress.go internal/engine/engine.go internal/engine/ingress_test.go
git commit -m "feat(engine): ingress controller selection (alb/nginx/traefik/gce/none)"
```

---

### Task 11: `ingress: istio` — Gateway + VirtualService

**Files:**
- Modify: `internal/engine/ingress.go` (`applyIstioIngress`)
- Modify: `internal/engine/engine.go`
- Test: `internal/engine/istio_test.go`

**Interfaces:**
- Produces: when `controller == "istio"`, emit subchart `templates/istio.yaml` containing a `Gateway` + `VirtualService` (gated by `{{- if .Values.ingress.host }}`).

- [ ] **Step 1: Write the failing test**

```go
// internal/engine/istio_test.go
package engine

import (
	"strings"
	"testing"
)

func TestIngressIstioGatewayAndVirtualService(t *testing.T) {
	man := renderWithHost(t, "istio")
	if !strings.Contains(man, "kind: Gateway") {
		t.Fatalf("expected istio Gateway, got:\n%s", man)
	}
	if !strings.Contains(man, "kind: VirtualService") {
		t.Fatalf("expected istio VirtualService, got:\n%s", man)
	}
	if strings.Contains(man, "kind: Ingress") {
		t.Fatalf("istio should not emit a plain Ingress")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestIngressIstio -v`
Expected: FAIL — no Gateway/VirtualService.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/engine/ingress.go (append)

func istioTemplate() string {
	return `{{- if .Values.ingress.host }}
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
spec:
  selector:
    istio: ingressgateway
  servers:
    - port:
        number: 80
        name: http
        protocol: HTTP
      hosts:
        - {{ .Values.ingress.host | quote }}
---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: {{ include (print .Chart.Name ".fullname") . }}
  labels:
    {{- include (print .Chart.Name ".labels") . | nindent 4 }}
spec:
  hosts:
    - {{ .Values.ingress.host | quote }}
  gateways:
    - {{ include (print .Chart.Name ".fullname") . }}
  http:
    - match:
        - uri:
            prefix: {{ default "/" .Values.ingress.path }}
      route:
        - destination:
            host: {{ include (print .Chart.Name ".fullname") . }}
            port:
              number: {{ .Values.service.port }}
{{- end }}
`
}
```

Update `applyIngress`'s `case "istio"` to emit the template instead of returning:

```go
	case "none":
		return
	case "istio":
		sub.Templates = append(sub.Templates, &chart.File{
			Name: "templates/istio.yaml",
			Data: []byte(istioTemplate()),
		})
		return
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestIngressIstio -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/ingress.go internal/engine/engine.go internal/engine/istio_test.go
git commit -m "feat(engine): ingress=istio emits Gateway + VirtualService"
```

---

### Task 12: `linked_templates: false` — inline self-contained subcharts

**Files:**
- Modify: `internal/engine/rules.go` (`applyInlining`)
- Modify: `internal/engine/engine.go`
- Test: `internal/engine/linked_test.go`

**Interfaces:**
- Produces: `applyInlining(sub *chart.Chart, umbrella *chart.Chart)` — when `linked_templates: false`, copies the umbrella's named-template `define` blocks (`umbrella-chart.*`) into the subchart's `_helpers.tpl` so the subchart's `{{ include "umbrella-chart.<kind>" . }}` stubs resolve **without** the umbrella partials. BuildChart passes the loaded umbrella's helper bodies.

Approach: the subchart manifests reference `umbrella-chart.deployment`, `umbrella-chart.service`, etc. With linking, those are defined in the umbrella's `.tpl` files (global to the render). To make a subchart self-contained, copy every `{{- define "umbrella-chart.* -}}...{{- end }}` block from the umbrella templates into the subchart's `templates/_helpers.tpl`. (Rendering still works either way; the test asserts the defines are physically present in the subchart so it survives extraction.)

- [ ] **Step 1: Write the failing test**

```go
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
```

The test never names the `chart.Chart` type explicitly (it ranges over `ch.Dependencies()`), so no extra import is needed.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestLinked -v`
Expected: FAIL — subchart helpers do not contain the umbrella defines.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/engine/rules.go (append)

// collectUmbrellaDefines returns the concatenated bodies of every umbrella .tpl
// named-template file (the files whose name ends in .tpl under templates/).
func collectUmbrellaDefines(umbrella *chart.Chart) string {
	var b strings.Builder
	for _, t := range umbrella.Templates {
		if strings.HasSuffix(t.Name, ".tpl") {
			b.Write(t.Data)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// applyInlining appends the umbrella named-template defines into the subchart's
// _helpers.tpl so the subchart resolves its includes standalone.
func applyInlining(sub *chart.Chart, defines string) {
	for _, t := range sub.Templates {
		if t.Name == "templates/_helpers.tpl" {
			t.Data = append(append([]byte{}, t.Data...), append([]byte("\n"), []byte(defines)...)...)
			return
		}
	}
	sub.Templates = append(sub.Templates, &chart.File{
		Name: "templates/_helpers.tpl",
		Data: []byte(defines),
	})
}
```

Wire into `BuildChart`: capture the umbrella defines once (`defines := collectUmbrellaDefines(umbrella)` — note: capture **before** any per-subchart edits, after `renameChart` is fine since defines still say `umbrella-chart.*` only if renameChart did not rewrite them; `renameChart` rewrites the *umbrella name* token, but the named templates are literally `umbrella-chart.deployment` — they DO get rewritten by renameChart). To keep the `umbrella-chart.*` define names intact for inlining, collect the defines from a **fresh load** of the umbrella template (before rename), or skip renaming inside `.tpl` define names. Implement by loading umbrella defines from a separate `loader.Load(umbrellaPath)` copy that is not renamed:

```go
	// inside BuildChart, before the subchart loop:
	var inlineDefines string
	if !spec.Rules.LinkedTemplates {
		pristine, err := loader.Load(umbrellaPath)
		if err != nil {
			return nil, fmt.Errorf("load umbrella for inlining: %w", err)
		}
		inlineDefines = collectUmbrellaDefines(pristine)
	}
```

and in `buildSubchart` (pass `inlineDefines` through), after `applyIngress`:

```go
	if !spec.Rules.LinkedTemplates {
		applyInlining(sub, inlineDefines)
	}
```

(Thread `inlineDefines string` as a parameter to `buildSubchart`.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestLinked -v`
Expected: PASS.

- [ ] **Step 5: Run the full engine suite + commit**

Run: `go test ./internal/engine/ -v`
Expected: all tasks' tests PASS.

```bash
git add internal/engine/rules.go internal/engine/engine.go internal/engine/linked_test.go
git commit -m "feat(engine): linked_templates=false inlines umbrella defines into subcharts"
```

---

## Self-Review

- **Spec coverage (design §4):** workload selection (T3), all six rules — `linked_templates` (T12), `resource_names_match_chart_name` (T6), `common_annotations` (T7), `shared_secrets_config` (T8), `shared_newrelic_config` (T9), `ingress` incl. istio (T10–T11) — readme/docs toggles (T5), descriptions (T2/T4), scaffold-only + Go-side mechanism (all). Spec/validation/normalize (T1). Golden render via in-process helm engine (T2). Covered.
- **Out of this plan (later phases):** the `*bool` decode-layer defaulting for omitted JSON booleans (Phase 2), `/generate`/`/text-to-config`/`/charts`, CRD edits, operator, frontend.
- **Type consistency:** `Spec`/`Subchart`/`Rules` field names are used identically across tasks; `BuildChart(spec, templatesDir)` and helper signatures (`applyWorkload`, `applyResourceNaming`, `applyCommonAnnotations`, `applySharedSecrets*`, `applySharedNewrelic*`, `applyIngress`, `applyInlining`) are referenced consistently.
- **Known execution risks (validate via the golden tests, adjust template text inline):** (1) Task 9 requires the engine's `deployment.tpl` copy to support simple `{name,value}`/`{name,valueFrom}` env entries; (2) Task 6 regex must match the actual `_helpers.tpl` whitespace; (3) Task 12 inlining keeps `umbrella-chart.*` define names by loading a pristine umbrella copy. Each has a fallback note in-task.

## Notes for Phase 2+ (not implemented here)

- Phase 2 decodes `/generate` JSON with `*bool` rule fields → fills omitted booleans with `DefaultRules()` before calling `engine.Normalize`/`engine.Validate`/`engine.BuildChart`. The server repoints `generateChart` to `engine.GenerateChart`.
- The operator (Phase 3) imports `internal/engine` directly and renders from the CR `spec`.
