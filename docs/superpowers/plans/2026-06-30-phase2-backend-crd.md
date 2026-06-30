# Phase 2 — Backend + CRD Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Repoint the backend from synchronous local chart-rendering to an asynchronous control-plane: `/generate` validates a Spec, wraps it into a `ChartpressConfig` CR, and server-side-applies it; `/charts` lists those CRs; `/text-to-config` drafts a Spec from a prompt via OpenAI; the CRD + examples migrate to the new (`chartpress.dev`, single-ingress, 3-workload) shape; the Helm chart gains backend RBAC + downward-API namespace.

**Architecture:** All chart generation logic now lives in `internal/engine` (Phase 1, done). The backend (`internal/server`) becomes a thin HTTP layer that (a) decodes a Spec with presence-tracked rule booleans, (b) normalizes/validates via the engine, (c) wraps into an unstructured `ChartpressConfig` and applies it with a `dynamic` client (real SSA in prod, `dynamic/fake` in tests), and (d) lists CRs for the Charts view. Object storage / presigned download URLs are **deferred to Phase 3** — `downloadUrl` is left empty this phase. The backend uses dependency-injected interfaces (`Applier`, `ChartLister`, `Drafter`) so every handler is testable with in-memory fakes and no apiserver.

**Tech Stack:** Go 1.23, `net/http`, `k8s.io/client-go/dynamic` (+ `dynamic/fake`), `k8s.io/apimachinery` unstructured (all already in `go.mod` at v0.32.2, promoted from indirect to direct), `sigs.k8s.io/yaml`, Helm v3 Go SDK for the chart-render test, OpenAI Responses API over `net/http`.

## Global Constraints

- **API group / version:** `chartpress.dev/v1alpha1`, kind `ChartpressConfig`, resource `chartpressconfigs`. The CRD currently on disk uses group `kriipke.dev` — Phase 2 **migrates it** to `chartpress.dev` (CRD, both example CRs, README, and the apply GVR). Copy these verbatim.
- **GVR:** `schema.GroupVersionResource{Group: "chartpress.dev", Version: "v1alpha1", Resource: "chartpressconfigs"}`.
- **Field manager for SSA:** `chartpress-backend`. Apply with `Force: true`.
- **Allowed workloads:** `deployment`, `statefulset`, `daemonset` (job/cronjob rejected everywhere).
- **Allowed ingress:** `alb`, `nginx`, `traefik`, `istio`, `gce`, `none` (single string enum — NOT a list).
- **Rule defaults (locked):** `ingress: "alb"`, `linked_templates: true`, `generate_umbrella_readme: true`, `generate_subchart_readme: true`, `include_docs: true`; every other boolean `false`. These come from `engine.DefaultRules()` — never hardcode them, call it.
- **Presence-tracking is mandatory:** the `/generate` (and `/text-to-config`) decode layer MUST use `*bool` / `*string` rule fields and fill only the omitted ones from `engine.DefaultRules()` BEFORE building `engine.Rules`. `engine.Normalize` fills the ingress default but NOT booleans. Losing this breaks the "minimal `{umbrellaChartName, subcharts}` body ≈ today's output" backward-compat guarantee.
- **`/generate` is async:** it returns `{name, namespace, phase:"Pending", manifestYaml}` after applying the CR. No local rendering, no zip, no `/download`. The pre-Phase-2 sync path (`generateChart`, `renameChart`, `newSubchart`, `loadChart`, `validateConfig`, `zipOutputDir`, `handleDownload`, the `Accept: application/zip` branch, and the old `Config`/`Subchart` types) is DELETED.
- **Object storage deferred:** no `minio-go`, no S3 config, no presigning this phase. `chartSummary.downloadUrl` stays empty (Phase 3 fills it once the operator writes `status.artifactKey`).
- **Namespace:** read from `POD_NAMESPACE` (downward API); default `"default"` when unset.
- **OpenAI:** Responses API, model from `OPENAI_MODEL` (default `gpt-4.1`), key from `OPENAI_API_KEY`, strict JSON-schema structured output. The real call sits behind the `Drafter` interface so handlers test against a fake.
- **Keep green throughout:** `go build ./...` clean and `go test ./internal/engine/` green after every task; run `go mod tidy` whenever a task adds a direct import.

## File Structure

`internal/server/` (split the current 399-line `server.go` by responsibility):

- `server.go` — `Server` struct (holds `Applier`, `Lister`, `Drafter`, `Namespace`), `NewServer`-style construction via `Start()`, `Handler()` mux, `cors` middleware, `getPort()`. Slim wiring only.
- `spec.go` — inbound Spec decode layer: `requestSpec`/`requestRules` (`*bool`/`*string`), `decodeSpec`.
- `manifest.go` — package constants (group/version/kind/fieldManager), `chartpressGVR`, `wrapManifest`, `manifestYAML`.
- `k8s.go` — `Applier`/`ChartLister`/`Drafter` interfaces, `dynamicApplier`, `dynamicLister`, `newDynamicClient`, `resolveNamespace`.
- `generate.go` — `handleGenerate` + `generateResponse`.
- `charts.go` — `handleCharts`, `handleChartByName`, `chartSummary`, `summarize`.
- `texttoconfig.go` — `handleTextToConfig`.
- `openai.go` — `openAIDrafter` (Responses API impl) + `newOpenAIDrafter`.

New test-only packages (each gets a one-line `doc.go` so `go build ./...` stays clean — a directory with only `_test.go` files breaks the build):

- `internal/crd/` — `doc.go` + `crd_test.go`: asserts the migrated CRD schema and that the example CRs are valid `engine.Spec`s.
- `internal/deploy/` — `doc.go` + `chart_test.go`: renders `chart/` via the Helm SDK and asserts the backend RBAC + downward-API env.

Modified outside Go: `crds/crd-helmchart.yaml`, `crds/helmchart-iot.yaml`, `crds/helmchart-ml.yaml`, `crds/README.md`, `chart/templates/backend-rbac.yaml` (new), `chart/templates/backend-deployment.yaml`, `chart/values.yaml`.

---

### Task 1: Spec decode layer (`*bool` defaulting)

**Files:**
- Create: `internal/server/spec.go`
- Test: `internal/server/spec_test.go`

**Interfaces:**
- Consumes: `engine.Spec`, `engine.Subchart`, `engine.Rules`, `engine.DefaultRules()`, `engine.Normalize` from `github.com/kriipke/chartpress/internal/engine`.
- Produces: `func decodeSpec(r io.Reader) (engine.Spec, error)` — JSON-decodes a request body, fills omitted rule fields from `engine.DefaultRules()` via presence-tracked pointers, returns a `engine.Normalize`d spec. Does NOT call `engine.Validate` (callers do).

- [ ] **Step 1: Write the failing test**

```go
// internal/server/spec_test.go
package server

import (
	"strings"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
)

func TestDecodeSpecMinimalBodyGetsDefaultRules(t *testing.T) {
	body := `{"umbrellaChartName":"demo","subcharts":[{"name":"api","workload":"deployment"}]}`
	spec, err := decodeSpec(strings.NewReader(body))
	if err != nil {
		t.Fatalf("decodeSpec: %v", err)
	}
	if spec.UmbrellaChartName != "demo" {
		t.Fatalf("name = %q, want demo", spec.UmbrellaChartName)
	}
	if len(spec.Subcharts) != 1 || spec.Subcharts[0].Workload != "deployment" {
		t.Fatalf("subcharts = %+v", spec.Subcharts)
	}
	want := engine.DefaultRules()
	if spec.Rules != want {
		t.Fatalf("rules = %+v, want defaults %+v", spec.Rules, want)
	}
}

func TestDecodeSpecHonorsExplicitFalseAndTrue(t *testing.T) {
	// linked_templates defaults true; explicit false must survive (not be re-defaulted).
	// common_annotations defaults false; explicit true must survive.
	body := `{"umbrellaChartName":"demo","subcharts":[{"name":"api","workload":"statefulset"}],
		"rules":{"linked_templates":false,"common_annotations":true,"ingress":"nginx"}}`
	spec, err := decodeSpec(strings.NewReader(body))
	if err != nil {
		t.Fatalf("decodeSpec: %v", err)
	}
	if spec.Rules.LinkedTemplates {
		t.Fatal("linked_templates: explicit false was lost")
	}
	if !spec.Rules.CommonAnnotations {
		t.Fatal("common_annotations: explicit true was lost")
	}
	if spec.Rules.Ingress != "nginx" {
		t.Fatalf("ingress = %q, want nginx", spec.Rules.Ingress)
	}
	// untouched booleans still take their defaults
	if !spec.Rules.IncludeDocs || !spec.Rules.GenerateUmbrellaReadme {
		t.Fatalf("omitted true-defaults were dropped: %+v", spec.Rules)
	}
}

func TestDecodeSpecNormalizesNames(t *testing.T) {
	body := `{"umbrellaChartName":"  Demo  ","subcharts":[{"name":"API","workload":"Deployment"}]}`
	spec, err := decodeSpec(strings.NewReader(body))
	if err != nil {
		t.Fatalf("decodeSpec: %v", err)
	}
	if spec.UmbrellaChartName != "demo" || spec.Subcharts[0].Name != "api" || spec.Subcharts[0].Workload != "deployment" {
		t.Fatalf("not normalized: %+v", spec)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/ -run TestDecodeSpec -v`
Expected: FAIL — `undefined: decodeSpec` (build error).

- [ ] **Step 3: Write minimal implementation**

```go
// internal/server/spec.go
package server

import (
	"encoding/json"
	"io"

	"github.com/kriipke/chartpress/internal/engine"
)

// requestRules mirrors engine.Rules but with pointers so we can tell an omitted
// field from an explicit false. engine.Normalize fills the ingress default but
// never fills booleans, so the defaulting MUST happen here.
type requestRules struct {
	Ingress                     *string `json:"ingress"`
	CommonAnnotations           *bool   `json:"common_annotations"`
	LinkedTemplates             *bool   `json:"linked_templates"`
	ResourceNamesMatchChartName *bool   `json:"resource_names_match_chart_name"`
	SharedSecretsConfig         *bool   `json:"shared_secrets_config"`
	SharedNewrelicConfig        *bool   `json:"shared_newrelic_config"`
	GenerateUmbrellaReadme      *bool   `json:"generate_umbrella_readme"`
	GenerateSubchartReadme      *bool   `json:"generate_subchart_readme"`
	IncludeDocs                 *bool   `json:"include_docs"`
}

type requestSpec struct {
	UmbrellaChartName string            `json:"umbrellaChartName"`
	Description       string            `json:"description"`
	Subcharts         []engine.Subchart `json:"subcharts"`
	Rules             *requestRules     `json:"rules"`
}

// decodeSpec reads a Spec request body, fills omitted rule fields from
// engine.DefaultRules(), and returns a normalized engine.Spec. It does NOT
// validate — callers run engine.Validate.
func decodeSpec(r io.Reader) (engine.Spec, error) {
	var req requestSpec
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return engine.Spec{}, err
	}

	rules := engine.DefaultRules()
	if rr := req.Rules; rr != nil {
		if rr.Ingress != nil {
			rules.Ingress = *rr.Ingress
		}
		if rr.CommonAnnotations != nil {
			rules.CommonAnnotations = *rr.CommonAnnotations
		}
		if rr.LinkedTemplates != nil {
			rules.LinkedTemplates = *rr.LinkedTemplates
		}
		if rr.ResourceNamesMatchChartName != nil {
			rules.ResourceNamesMatchChartName = *rr.ResourceNamesMatchChartName
		}
		if rr.SharedSecretsConfig != nil {
			rules.SharedSecretsConfig = *rr.SharedSecretsConfig
		}
		if rr.SharedNewrelicConfig != nil {
			rules.SharedNewrelicConfig = *rr.SharedNewrelicConfig
		}
		if rr.GenerateUmbrellaReadme != nil {
			rules.GenerateUmbrellaReadme = *rr.GenerateUmbrellaReadme
		}
		if rr.GenerateSubchartReadme != nil {
			rules.GenerateSubchartReadme = *rr.GenerateSubchartReadme
		}
		if rr.IncludeDocs != nil {
			rules.IncludeDocs = *rr.IncludeDocs
		}
	}

	spec := engine.Spec{
		UmbrellaChartName: req.UmbrellaChartName,
		Description:       req.Description,
		Subcharts:         req.Subcharts,
		Rules:             rules,
	}
	return engine.Normalize(spec), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/server/ -run TestDecodeSpec -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/server/spec.go internal/server/spec_test.go
git commit -m "feat(server): spec decode layer with presence-tracked rule defaulting"
```

---

### Task 2: Manifest wrapping

**Files:**
- Create: `internal/server/manifest.go`
- Test: `internal/server/manifest_test.go`

**Interfaces:**
- Consumes: `engine.Spec`, `engine.Normalize`, `engine.DefaultRules` (test only).
- Produces:
  - consts `apiGroup="chartpress.dev"`, `apiVersionV1alpha1="chartpress.dev/v1alpha1"`, `kindChartpressConfig="ChartpressConfig"`, `fieldManager="chartpress-backend"`.
  - `var chartpressGVR schema.GroupVersionResource` (group `chartpress.dev`, version `v1alpha1`, resource `chartpressconfigs`).
  - `func wrapManifest(spec engine.Spec) *unstructured.Unstructured` — apiVersion/kind/metadata.name(=UmbrellaChartName)/spec(=the engine.Spec marshalled to a map).
  - `func manifestYAML(obj *unstructured.Unstructured) (string, error)`.

- [ ] **Step 1: Write the failing test**

```go
// internal/server/manifest_test.go
package server

import (
	"strings"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func sampleSpec() engine.Spec {
	return engine.Normalize(engine.Spec{
		UmbrellaChartName: "demo-platform",
		Description:       "Example platform chart",
		Subcharts:         []engine.Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             engine.DefaultRules(),
	})
}

func TestWrapManifestShape(t *testing.T) {
	obj := wrapManifest(sampleSpec())
	if obj.GetAPIVersion() != "chartpress.dev/v1alpha1" {
		t.Fatalf("apiVersion = %q", obj.GetAPIVersion())
	}
	if obj.GetKind() != "ChartpressConfig" {
		t.Fatalf("kind = %q", obj.GetKind())
	}
	if obj.GetName() != "demo-platform" {
		t.Fatalf("metadata.name = %q, want demo-platform", obj.GetName())
	}
	name, found, err := unstructured.NestedString(obj.Object, "spec", "umbrellaChartName")
	if err != nil || !found || name != "demo-platform" {
		t.Fatalf("spec.umbrellaChartName = %q found=%v err=%v", name, found, err)
	}
	subs, found, err := unstructured.NestedSlice(obj.Object, "spec", "subcharts")
	if err != nil || !found || len(subs) != 1 {
		t.Fatalf("spec.subcharts len=%d found=%v err=%v", len(subs), found, err)
	}
	ingress, _, _ := unstructured.NestedString(obj.Object, "spec", "rules", "ingress")
	if ingress != "alb" {
		t.Fatalf("spec.rules.ingress = %q, want alb", ingress)
	}
}

func TestManifestYAMLRoundTrips(t *testing.T) {
	y, err := manifestYAML(wrapManifest(sampleSpec()))
	if err != nil {
		t.Fatalf("manifestYAML: %v", err)
	}
	for _, want := range []string{
		"apiVersion: chartpress.dev/v1alpha1",
		"kind: ChartpressConfig",
		"name: demo-platform",
		"umbrellaChartName: demo-platform",
	} {
		if !strings.Contains(y, want) {
			t.Fatalf("manifest YAML missing %q:\n%s", want, y)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/ -run 'TestWrapManifest|TestManifestYAML' -v`
Expected: FAIL — `undefined: wrapManifest` / `undefined: manifestYAML`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/server/manifest.go
package server

import (
	"encoding/json"

	"github.com/kriipke/chartpress/internal/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	sigsyaml "sigs.k8s.io/yaml"
)

const (
	apiGroup             = "chartpress.dev"
	apiVersionV1alpha1   = "chartpress.dev/v1alpha1"
	kindChartpressConfig = "ChartpressConfig"
	fieldManager         = "chartpress-backend"
)

var chartpressGVR = schema.GroupVersionResource{
	Group:    apiGroup,
	Version:  "v1alpha1",
	Resource: "chartpressconfigs",
}

// wrapManifest builds an unstructured ChartpressConfig CR from a (already
// normalized) spec. metadata.name == spec.UmbrellaChartName; .spec is the spec
// itself, JSON round-tripped so the persisted CR carries the engine field names.
func wrapManifest(spec engine.Spec) *unstructured.Unstructured {
	b, _ := json.Marshal(spec) // engine.Spec has no unmarshalable fields
	var specMap map[string]interface{}
	_ = json.Unmarshal(b, &specMap)

	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apiVersionV1alpha1,
		"kind":       kindChartpressConfig,
		"metadata": map[string]interface{}{
			"name": spec.UmbrellaChartName,
		},
		"spec": specMap,
	}}
}

// manifestYAML serializes the CR to YAML for the /generate response body.
func manifestYAML(obj *unstructured.Unstructured) (string, error) {
	b, err := sigsyaml.Marshal(obj.Object)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go mod tidy && go test ./internal/server/ -run 'TestWrapManifest|TestManifestYAML' -v`
Expected: PASS (2 tests). `go mod tidy` promotes `k8s.io/apimachinery` and `sigs.k8s.io/yaml` to direct deps.

- [ ] **Step 5: Commit**

```bash
git add internal/server/manifest.go internal/server/manifest_test.go go.mod go.sum
git commit -m "feat(server): wrap spec into unstructured ChartpressConfig (chartpress.dev/v1alpha1)"
```

---

### Task 3: Kubernetes client seam (interfaces + dynamic impls)

**Files:**
- Create: `internal/server/k8s.go`
- Test: `internal/server/k8s_test.go`

**Interfaces:**
- Consumes: `chartpressGVR`, `fieldManager`, `wrapManifest` (test only), `engine.*` (test only).
- Produces:
  - `type Applier interface { Apply(ctx context.Context, namespace string, obj *unstructured.Unstructured) error }`
  - `type ChartLister interface { List(ctx context.Context, namespace string) ([]unstructured.Unstructured, error); Get(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) }`
  - `type Drafter interface { Draft(ctx context.Context, prompt string) (engine.Spec, error) }` (implemented in Task 6).
  - `type dynamicApplier struct{ client dynamic.Interface }` implementing `Applier` via SSA `.Apply` with `fieldManager`/`Force:true`.
  - `type dynamicLister struct{ client dynamic.Interface }` implementing `ChartLister`.
  - `func newDynamicClient() (dynamic.Interface, error)` (in-cluster config).
  - `func resolveNamespace() string` (`POD_NAMESPACE` or `"default"`).

- [ ] **Step 1: Write the failing test**

```go
// internal/server/k8s_test.go
package server

import (
	"context"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func newFakeDynamic(objs ...runtime.Object) *dynamicfake.FakeDynamicClient {
	scheme := runtime.NewScheme()
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{chartpressGVR: "ChartpressConfigList"},
		objs...,
	)
}

func specFor(name string) engine.Spec {
	return engine.Normalize(engine.Spec{
		UmbrellaChartName: name,
		Subcharts:         []engine.Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             engine.DefaultRules(),
	})
}

func TestDynamicApplierCreatesThenUpdates(t *testing.T) {
	fc := newFakeDynamic()
	a := &dynamicApplier{client: fc}
	ctx := context.Background()

	if err := a.Apply(ctx, "team-a", wrapManifest(specFor("demo"))); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	got, err := fc.Resource(chartpressGVR).Namespace("team-a").Get(ctx, "demo", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get after apply: %v", err)
	}
	if got.GetKind() != "ChartpressConfig" {
		t.Fatalf("kind = %q", got.GetKind())
	}
	desc, _, _ := unstructured.NestedString(got.Object, "spec", "description")
	if desc != "" {
		t.Fatalf("unexpected description %q", desc)
	}

	// Re-apply with a changed spec → SSA updates the same object.
	updated := specFor("demo")
	updated.Description = "now with a description"
	if err := a.Apply(ctx, "team-a", wrapManifest(updated)); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	got2, err := fc.Resource(chartpressGVR).Namespace("team-a").Get(ctx, "demo", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get after re-apply: %v", err)
	}
	desc2, _, _ := unstructured.NestedString(got2.Object, "spec", "description")
	if desc2 != "now with a description" {
		t.Fatalf("description after update = %q", desc2)
	}
}

func TestDynamicListerListAndGet(t *testing.T) {
	fc := newFakeDynamic(wrapManifest(specFor("a")), wrapManifest(specFor("b")))
	l := &dynamicLister{client: fc}
	ctx := context.Background()

	items, err := l.List(ctx, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("list len = %d, want 2", len(items))
	}

	one, err := l.Get(ctx, "", "a")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if one.GetName() != "a" {
		t.Fatalf("get name = %q", one.GetName())
	}
}

func TestResolveNamespace(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "")
	if got := resolveNamespace(); got != "default" {
		t.Fatalf("default ns = %q, want default", got)
	}
	t.Setenv("POD_NAMESPACE", "chartpress-system")
	if got := resolveNamespace(); got != "chartpress-system" {
		t.Fatalf("ns = %q, want chartpress-system", got)
	}
}
```

> Note: the example CRs in `newFakeDynamic(...)` are applied at namespace `""` (cluster-wide fake list). The fake's `NewSimpleDynamicClientWithCustomListKinds` registers the list kind so `.List` does not panic.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/ -run 'TestDynamic|TestResolveNamespace' -v`
Expected: FAIL — `undefined: dynamicApplier` / `dynamicLister` / `resolveNamespace`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/server/k8s.go
package server

import (
	"context"
	"os"

	"github.com/kriipke/chartpress/internal/engine"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// Applier server-side-applies a ChartpressConfig CR.
type Applier interface {
	Apply(ctx context.Context, namespace string, obj *unstructured.Unstructured) error
}

// ChartLister reads ChartpressConfig CRs for the Charts view.
type ChartLister interface {
	List(ctx context.Context, namespace string) ([]unstructured.Unstructured, error)
	Get(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error)
}

// Drafter turns a natural-language prompt into a Spec (implemented in openai.go).
type Drafter interface {
	Draft(ctx context.Context, prompt string) (engine.Spec, error)
}

type dynamicApplier struct{ client dynamic.Interface }

func (a *dynamicApplier) Apply(ctx context.Context, namespace string, obj *unstructured.Unstructured) error {
	_, err := a.client.Resource(chartpressGVR).Namespace(namespace).Apply(
		ctx, obj.GetName(), obj, metav1.ApplyOptions{FieldManager: fieldManager, Force: true},
	)
	return err
}

type dynamicLister struct{ client dynamic.Interface }

func (l *dynamicLister) List(ctx context.Context, namespace string) ([]unstructured.Unstructured, error) {
	list, err := l.client.Resource(chartpressGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (l *dynamicLister) Get(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	return l.client.Resource(chartpressGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// newDynamicClient builds an in-cluster dynamic client (production wiring).
func newDynamicClient() (dynamic.Interface, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(cfg)
}

// resolveNamespace reads POD_NAMESPACE (downward API), defaulting to "default".
func resolveNamespace() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	return "default"
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go mod tidy && go test ./internal/server/ -run 'TestDynamic|TestResolveNamespace' -v`
Expected: PASS (3 tests). `go mod tidy` promotes `k8s.io/client-go` to direct.

- [ ] **Step 5: Commit**

```bash
git add internal/server/k8s.go internal/server/k8s_test.go go.mod go.sum
git commit -m "feat(server): dynamic-client Applier/ChartLister seam + namespace resolution"
```

---

### Task 4: Server struct, `/generate` (async), remove dead sync code

**Files:**
- Rewrite: `internal/server/server.go` (replace the whole file)
- Create: `internal/server/generate.go`
- Test: `internal/server/generate_test.go`

**Interfaces:**
- Consumes: `decodeSpec` (T1), `wrapManifest`/`manifestYAML` (T2), `Applier`/`ChartLister`/`Drafter`/`resolveNamespace`/`newDynamicClient` (T3), `engine.Validate`.
- Produces:
  - `type Server struct { Applier Applier; Lister ChartLister; Drafter Drafter; Namespace string }`
  - `func (s *Server) Handler() http.Handler` (registers `/generate` now; `/charts` added in T5, `/text-to-config` in T6).
  - `func (s *Server) cors(next http.HandlerFunc) http.HandlerFunc`
  - `func Start()` (production wiring; sets Applier+Lister+Namespace; Drafter wired in T6).
  - `func getPort() string`
  - `type generateResponse struct { Name, Namespace, Phase, ManifestYAML string }` with json tags `name,namespace,phase,manifestYaml`.
  - `func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request)`.

- [ ] **Step 1: Write the failing test**

```go
// internal/server/generate_test.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// recordingApplier captures the last applied object.
type recordingApplier struct {
	ns  string
	obj *unstructured.Unstructured
	err error
}

func (a *recordingApplier) Apply(ctx context.Context, namespace string, obj *unstructured.Unstructured) error {
	a.ns, a.obj = namespace, obj
	return a.err
}

func newTestServer(a Applier) *Server {
	return &Server{Applier: a, Namespace: "chartpress-system"}
}

func TestGenerateAppliesCRAndReturnsPending(t *testing.T) {
	app := &recordingApplier{}
	srv := newTestServer(app)
	body := `{"umbrellaChartName":"demo","subcharts":[{"name":"api","workload":"deployment"}]}`
	req := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp generateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if resp.Name != "demo" || resp.Namespace != "chartpress-system" || resp.Phase != "Pending" {
		t.Fatalf("resp = %+v", resp)
	}
	if !strings.Contains(resp.ManifestYAML, "kind: ChartpressConfig") {
		t.Fatalf("manifestYaml missing kind:\n%s", resp.ManifestYAML)
	}
	if app.obj == nil || app.obj.GetName() != "demo" || app.ns != "chartpress-system" {
		t.Fatalf("applier got ns=%q obj=%v", app.ns, app.obj)
	}
}

func TestGenerateRejectsInvalidWorkload(t *testing.T) {
	srv := newTestServer(&recordingApplier{})
	body := `{"umbrellaChartName":"demo","subcharts":[{"name":"api","workload":"cronjob"}]}`
	req := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestGenerateRejectsNonPost(t *testing.T) {
	srv := newTestServer(&recordingApplier{})
	req := httptest.NewRequest(http.MethodGet, "/generate", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/ -run TestGenerate -v`
Expected: FAIL — `undefined: Server` / `generateResponse` (and the old package-level `handleGenerate` will conflict until server.go is rewritten).

- [ ] **Step 3: Write minimal implementation**

Replace the ENTIRE contents of `internal/server/server.go` with:

```go
// internal/server/server.go
package server

import (
	"log"
	"net/http"
	"os"
)

// Server is the chartpress backend HTTP layer. Its dependencies are interfaces
// so handlers can be tested with in-memory fakes (no apiserver, no OpenAI).
type Server struct {
	Applier   Applier
	Lister    ChartLister
	Drafter   Drafter
	Namespace string
}

// Handler builds the HTTP mux. Routes are registered by their owning task:
// /generate (this task), /charts + /charts/ (Task 5), /text-to-config (Task 6).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/generate", s.cors(s.handleGenerate))
	return mux
}

// cors sets permissive CORS headers and short-circuits preflight requests.
func (s *Server) cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

// Start wires production dependencies and serves. The dynamic client backs both
// apply and list; the namespace comes from the downward API.
func Start() {
	log.Println("[INFO] starting chartpress backend")
	client, err := newDynamicClient()
	if err != nil {
		log.Fatalf("[FATAL] kube client: %v", err)
	}
	srv := &Server{
		Applier:   &dynamicApplier{client: client},
		Lister:    &dynamicLister{client: client},
		Namespace: resolveNamespace(),
	}
	port := getPort()
	log.Printf("[INFO] listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv.Handler()))
}

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}
```

Create `internal/server/generate.go`:

```go
// internal/server/generate.go
package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/kriipke/chartpress/internal/engine"
)

type generateResponse struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Phase        string `json:"phase"`
	ManifestYAML string `json:"manifestYaml"`
}

// handleGenerate validates a Spec, wraps it into a ChartpressConfig CR,
// server-side-applies it, and returns immediately with phase Pending. Rendering
// happens asynchronously in the operator (Phase 3).
func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST is allowed", http.StatusMethodNotAllowed)
		return
	}

	spec, err := decodeSpec(r.Body)
	if err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := engine.Validate(spec); err != nil {
		http.Error(w, "invalid spec: "+err.Error(), http.StatusBadRequest)
		return
	}

	obj := wrapManifest(spec)
	if err := s.Applier.Apply(r.Context(), s.Namespace, obj); err != nil {
		log.Printf("[ERROR] apply ChartpressConfig %q: %v", spec.UmbrellaChartName, err)
		http.Error(w, "failed to apply ChartpressConfig: "+err.Error(), http.StatusInternalServerError)
		return
	}

	y, err := manifestYAML(obj)
	if err != nil {
		http.Error(w, "failed to serialize manifest: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(generateResponse{
		Name:         spec.UmbrellaChartName,
		Namespace:    s.Namespace,
		Phase:        "Pending",
		ManifestYAML: y,
	})
}
```

The rewrite of `server.go` deletes `generateChart`, `renameChart`, `newSubchart`, `loadChart`, `validateConfig`, `zipOutputDir`, `handleDownload`, the old `Config`/`Subchart` structs, and the `Accept: application/zip` branch. Confirm none survive:

Run: `grep -nE 'zipOutputDir|handleDownload|renameChart|newSubchart|loadChart|validateConfig|application/zip' internal/server/*.go`
Expected: no matches (empty output).

- [ ] **Step 4: Run test to verify it passes**

Run: `go build ./... && go test ./internal/server/ -v && go test ./internal/engine/`
Expected: build clean; all server tests PASS; engine still `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/server/server.go internal/server/generate.go internal/server/generate_test.go
git commit -m "feat(server): async /generate applies ChartpressConfig CR; remove local render/zip/download"
```

---

### Task 5: `/charts` and `/charts/<name>`

**Files:**
- Create: `internal/server/charts.go`
- Modify: `internal/server/server.go` (register `/charts` and `/charts/` in `Handler()`)
- Test: `internal/server/charts_test.go`

**Interfaces:**
- Consumes: `ChartLister` (T3), the `Server` struct + `cors` (T4).
- Produces:
  - `type chartSummary struct { Name string; Phase string; SubchartCount int; LastGenerated string; Message string; DownloadURL string }` with json tags `name,phase,subchartCount,lastGenerated,message,downloadUrl` (last three `omitempty`).
  - `func summarize(obj unstructured.Unstructured) chartSummary` — phase defaults to `"Pending"` when `status.phase` is empty; `downloadUrl` is always empty in Phase 2.
  - `func (s *Server) handleCharts(w, r)` (GET list) and `func (s *Server) handleChartByName(w, r)` (GET one; 404 on NotFound).

- [ ] **Step 1: Write the failing test**

```go
// internal/server/charts_test.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// fakeLister serves canned CRs.
type fakeLister struct {
	items []unstructured.Unstructured
}

func (f *fakeLister) List(ctx context.Context, namespace string) ([]unstructured.Unstructured, error) {
	return f.items, nil
}
func (f *fakeLister) Get(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	for i := range f.items {
		if f.items[i].GetName() == name {
			return &f.items[i], nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Group: apiGroup, Resource: "chartpressconfigs"}, name)
}

func crWith(name, phase, message string, subcharts int, lastGen string) unstructured.Unstructured {
	subs := make([]interface{}, subcharts)
	for i := range subs {
		subs[i] = map[string]interface{}{"name": "s", "workload": "deployment"}
	}
	status := map[string]interface{}{}
	if phase != "" {
		status["phase"] = phase
	}
	if message != "" {
		status["message"] = message
	}
	if lastGen != "" {
		status["lastGenerated"] = lastGen
	}
	return unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apiVersionV1alpha1,
		"kind":       kindChartpressConfig,
		"metadata":   map[string]interface{}{"name": name},
		"spec":       map[string]interface{}{"umbrellaChartName": name, "subcharts": subs},
		"status":     status,
	}}
}

func TestChartsListMapsFields(t *testing.T) {
	srv := &Server{Lister: &fakeLister{items: []unstructured.Unstructured{
		crWith("ready-one", "Ready", "", 3, "2026-06-30T03:10:00Z"),
		crWith("fresh-one", "", "", 1, ""), // empty status → Pending
		crWith("bad-one", "Failed", "render blew up", 2, ""),
	}}, Namespace: "ns"}

	req := httptest.NewRequest(http.MethodGet, "/charts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got []chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	byName := map[string]chartSummary{}
	for _, c := range got {
		byName[c.Name] = c
	}
	if c := byName["ready-one"]; c.Phase != "Ready" || c.SubchartCount != 3 || c.LastGenerated != "2026-06-30T03:10:00Z" || c.DownloadURL != "" {
		t.Fatalf("ready-one = %+v (downloadUrl must be empty in Phase 2)", c)
	}
	if c := byName["fresh-one"]; c.Phase != "Pending" {
		t.Fatalf("empty status should map to Pending, got %q", c.Phase)
	}
	if c := byName["bad-one"]; c.Phase != "Failed" || c.Message != "render blew up" {
		t.Fatalf("bad-one = %+v", c)
	}
}

func TestChartByNameFound(t *testing.T) {
	srv := &Server{Lister: &fakeLister{items: []unstructured.Unstructured{
		crWith("demo", "Generating", "", 2, ""),
	}}}
	req := httptest.NewRequest(http.MethodGet, "/charts/demo", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "demo" || got.Phase != "Generating" || got.SubchartCount != 2 {
		t.Fatalf("got = %+v", got)
	}
}

func TestChartByNameNotFound(t *testing.T) {
	srv := &Server{Lister: &fakeLister{}}
	req := httptest.NewRequest(http.MethodGet, "/charts/missing", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/ -run TestChart -v`
Expected: FAIL — `undefined: chartSummary` and `/charts` route not registered (404/connection).

- [ ] **Step 3: Write minimal implementation**

Create `internal/server/charts.go`:

```go
// internal/server/charts.go
package server

import (
	"encoding/json"
	"net/http"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type chartSummary struct {
	Name          string `json:"name"`
	Phase         string `json:"phase"`
	SubchartCount int    `json:"subchartCount"`
	LastGenerated string `json:"lastGenerated,omitempty"`
	Message       string `json:"message,omitempty"`
	DownloadURL   string `json:"downloadUrl,omitempty"`
}

// summarize maps a ChartpressConfig CR to the Charts-row shape. Phase defaults
// to "Pending" before the operator sets status. downloadUrl stays empty in
// Phase 2 — presigned URLs are minted in Phase 3 once status.artifactKey exists.
func summarize(obj unstructured.Unstructured) chartSummary {
	subs, _, _ := unstructured.NestedSlice(obj.Object, "spec", "subcharts")
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	if phase == "" {
		phase = "Pending"
	}
	msg, _, _ := unstructured.NestedString(obj.Object, "status", "message")
	lastGen, _, _ := unstructured.NestedString(obj.Object, "status", "lastGenerated")
	return chartSummary{
		Name:          obj.GetName(),
		Phase:         phase,
		SubchartCount: len(subs),
		LastGenerated: lastGen,
		Message:       msg,
	}
}

func (s *Server) handleCharts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET is allowed", http.StatusMethodNotAllowed)
		return
	}
	items, err := s.Lister.List(r.Context(), s.Namespace)
	if err != nil {
		http.Error(w, "failed to list charts: "+err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]chartSummary, 0, len(items))
	for _, it := range items {
		out = append(out, summarize(it))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleChartByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET is allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/charts/")
	if name == "" {
		http.Error(w, "missing chart name", http.StatusBadRequest)
		return
	}
	obj, err := s.Lister.Get(r.Context(), s.Namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			http.Error(w, "chart not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get chart: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summarize(*obj))
}
```

Add the routes to `Handler()` in `internal/server/server.go` (insert after the `/generate` line):

```go
	mux.HandleFunc("/charts", s.cors(s.handleCharts))
	mux.HandleFunc("/charts/", s.cors(s.handleChartByName))
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go build ./... && go test ./internal/server/ -v`
Expected: build clean; all server tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/server/charts.go internal/server/server.go internal/server/charts_test.go
git commit -m "feat(server): GET /charts and /charts/<name> list ChartpressConfig CRs"
```

---

### Task 6: `/text-to-config` + OpenAI Responses drafter

**Files:**
- Create: `internal/server/texttoconfig.go`
- Create: `internal/server/openai.go`
- Modify: `internal/server/server.go` (register `/text-to-config`; wire `Drafter` in `Start()`)
- Test: `internal/server/texttoconfig_test.go`
- Test: `internal/server/openai_test.go`

**Interfaces:**
- Consumes: `Drafter` interface (T3), `Server`/`cors` (T4), `engine.Spec`/`engine.Normalize`.
- Produces:
  - `func (s *Server) handleTextToConfig(w, r)` — body `{"prompt":"..."}`; returns the drafted (normalized) `engine.Spec` as JSON; 400 on empty prompt; 502 on drafter error.
  - `type openAIDrafter struct { apiKey, model, endpoint string; httpClient *http.Client }` implementing `Drafter`.
  - `func newOpenAIDrafter() *openAIDrafter` — model from `OPENAI_MODEL` (default `gpt-4.1`), key from `OPENAI_API_KEY`, endpoint `https://api.openai.com/v1/responses`.
  - `func specJSONSchema() map[string]interface{}` — the strict structured-output schema.

- [ ] **Step 1: Write the failing test**

```go
// internal/server/texttoconfig_test.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
)

type fakeDrafter struct {
	spec engine.Spec
	err  error
}

func (d fakeDrafter) Draft(ctx context.Context, prompt string) (engine.Spec, error) {
	return d.spec, d.err
}

func TestTextToConfigReturnsSpec(t *testing.T) {
	drafted := engine.Normalize(engine.Spec{
		UmbrellaChartName: "shop",
		Subcharts:         []engine.Subchart{{Name: "web", Workload: "deployment"}},
		Rules:             engine.DefaultRules(),
	})
	srv := &Server{Drafter: fakeDrafter{spec: drafted}}
	req := httptest.NewRequest(http.MethodPost, "/text-to-config", strings.NewReader(`{"prompt":"an online shop"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got engine.Spec
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.UmbrellaChartName != "shop" || len(got.Subcharts) != 1 || got.Rules.Ingress != "alb" {
		t.Fatalf("got = %+v", got)
	}
}

func TestTextToConfigRejectsEmptyPrompt(t *testing.T) {
	srv := &Server{Drafter: fakeDrafter{}}
	req := httptest.NewRequest(http.MethodPost, "/text-to-config", strings.NewReader(`{"prompt":"  "}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
```

```go
// internal/server/openai_test.go
package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIDrafterParsesResponsesOutput(t *testing.T) {
	// Stand in for the OpenAI Responses API: echo a spec back as output_text.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("auth header = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"model":"gpt-4.1"`) {
			t.Errorf("request missing model: %s", body)
		}
		if !strings.Contains(string(body), "json_schema") {
			t.Errorf("request missing structured-output schema: %s", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"output": []interface{}{
				map[string]interface{}{
					"type": "message",
					"content": []interface{}{
						map[string]interface{}{
							"type": "output_text",
							"text": `{"umbrellaChartName":"shop","description":"d","subcharts":[{"name":"web","workload":"deployment","description":""}],"rules":{"ingress":"nginx","common_annotations":false,"linked_templates":true,"resource_names_match_chart_name":false,"shared_secrets_config":false,"shared_newrelic_config":false,"generate_umbrella_readme":true,"generate_subchart_readme":true,"include_docs":true}}`,
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	d := &openAIDrafter{apiKey: "test-key", model: "gpt-4.1", endpoint: srv.URL, httpClient: srv.Client()}
	spec, err := d.Draft(context.Background(), "an online shop")
	if err != nil {
		t.Fatalf("Draft: %v", err)
	}
	if spec.UmbrellaChartName != "shop" || spec.Rules.Ingress != "nginx" || len(spec.Subcharts) != 1 {
		t.Fatalf("spec = %+v", spec)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/ -run 'TestTextToConfig|TestOpenAIDrafter' -v`
Expected: FAIL — `undefined: openAIDrafter` and `/text-to-config` not routed.

- [ ] **Step 3: Write minimal implementation**

Create `internal/server/texttoconfig.go`:

```go
// internal/server/texttoconfig.go
package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kriipke/chartpress/internal/engine"
)

// handleTextToConfig drafts a Spec from a natural-language prompt via the
// Drafter (OpenAI in production). It returns the normalized spec for the form to
// pre-fill; the frontend overrides the name and the user edits before /generate
// validates. This endpoint does not hard-validate (the draft may need editing).
func (s *Server) handleTextToConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST is allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Prompt) == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}
	spec, err := s.Drafter.Draft(r.Context(), body.Prompt)
	if err != nil {
		http.Error(w, "failed to draft spec: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(engine.Normalize(spec))
}
```

Create `internal/server/openai.go`:

```go
// internal/server/openai.go
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/kriipke/chartpress/internal/engine"
)

// openAIDrafter drafts a Spec via the OpenAI Responses API with strict
// JSON-schema structured output.
type openAIDrafter struct {
	apiKey     string
	model      string
	endpoint   string
	httpClient *http.Client
}

func newOpenAIDrafter() *openAIDrafter {
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4.1"
	}
	return &openAIDrafter{
		apiKey:     os.Getenv("OPENAI_API_KEY"),
		model:      model,
		endpoint:   "https://api.openai.com/v1/responses",
		httpClient: http.DefaultClient,
	}
}

const draftSystemPrompt = "You draft a chartpress Spec for a Kubernetes Helm umbrella chart from a short app description. " +
	"Choose a kebab-case umbrellaChartName, 1+ subcharts each with a kebab-case name and a workload of deployment, statefulset, or daemonset, " +
	"and a rules block. Only emit fields defined by the schema."

func (d *openAIDrafter) Draft(ctx context.Context, prompt string) (engine.Spec, error) {
	reqBody := map[string]interface{}{
		"model": d.model,
		"input": []map[string]string{
			{"role": "system", "content": draftSystemPrompt},
			{"role": "user", "content": prompt},
		},
		"text": map[string]interface{}{
			"format": map[string]interface{}{
				"type":   "json_schema",
				"name":   "chartpress_spec",
				"strict": true,
				"schema": specJSONSchema(),
			},
		},
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return engine.Spec{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.endpoint, bytes.NewReader(buf))
	if err != nil {
		return engine.Spec{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return engine.Spec{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return engine.Spec{}, fmt.Errorf("openai responses: status %d", resp.StatusCode)
	}

	var parsed struct {
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return engine.Spec{}, err
	}

	var jsonText string
	for _, o := range parsed.Output {
		if o.Type != "message" {
			continue
		}
		for _, c := range o.Content {
			if c.Type == "output_text" {
				jsonText += c.Text
			}
		}
	}
	if jsonText == "" {
		return engine.Spec{}, fmt.Errorf("openai responses: no output_text in response")
	}

	var spec engine.Spec
	if err := json.Unmarshal([]byte(jsonText), &spec); err != nil {
		return engine.Spec{}, fmt.Errorf("openai responses: parse spec: %w", err)
	}
	return spec, nil
}

// specJSONSchema is the strict structured-output schema for a chartpress Spec.
// Strict mode requires additionalProperties:false and every property in required.
func specJSONSchema() map[string]interface{} {
	boolProp := map[string]interface{}{"type": "boolean"}
	return map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"umbrellaChartName", "description", "subcharts", "rules"},
		"properties": map[string]interface{}{
			"umbrellaChartName": map[string]interface{}{"type": "string"},
			"description":       map[string]interface{}{"type": "string"},
			"subcharts": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"name", "workload", "description"},
					"properties": map[string]interface{}{
						"name":        map[string]interface{}{"type": "string"},
						"workload":    map[string]interface{}{"type": "string", "enum": engine.AllowedWorkloads},
						"description": map[string]interface{}{"type": "string"},
					},
				},
			},
			"rules": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"required": []string{
					"ingress", "common_annotations", "linked_templates",
					"resource_names_match_chart_name", "shared_secrets_config",
					"shared_newrelic_config", "generate_umbrella_readme",
					"generate_subchart_readme", "include_docs",
				},
				"properties": map[string]interface{}{
					"ingress":                         map[string]interface{}{"type": "string", "enum": engine.AllowedIngress},
					"common_annotations":              boolProp,
					"linked_templates":                boolProp,
					"resource_names_match_chart_name": boolProp,
					"shared_secrets_config":           boolProp,
					"shared_newrelic_config":          boolProp,
					"generate_umbrella_readme":        boolProp,
					"generate_subchart_readme":        boolProp,
					"include_docs":                    boolProp,
				},
			},
		},
	}
}
```

Add the route to `Handler()` in `server.go` (after the `/charts/` line):

```go
	mux.HandleFunc("/text-to-config", s.cors(s.handleTextToConfig))
```

Wire the production drafter in `Start()` — change the `srv := &Server{...}` literal to include:

```go
		Drafter:   newOpenAIDrafter(),
```

- [ ] **Step 2 (re-affirm): Run test to verify it fails** — already covered above; proceed.

- [ ] **Step 4: Run test to verify it passes**

Run: `go build ./... && go test ./internal/server/ -v && go test ./internal/engine/`
Expected: build clean; all server tests PASS; engine `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/server/texttoconfig.go internal/server/openai.go internal/server/server.go internal/server/texttoconfig_test.go internal/server/openai_test.go
git commit -m "feat(server): /text-to-config drafts a Spec via OpenAI Responses API"
```

---

### Task 7: CRD migration + example CRs + README

**Files:**
- Modify: `crds/crd-helmchart.yaml`
- Modify: `crds/helmchart-iot.yaml`
- Modify: `crds/helmchart-ml.yaml`
- Modify: `crds/README.md`
- Create: `internal/crd/doc.go`
- Create: `internal/crd/crd_test.go`

**Interfaces:**
- Consumes: `engine.Spec`, `engine.Normalize`, `engine.Validate`, `engine.AllowedWorkloads`, `engine.AllowedIngress`.
- Produces: a test package asserting the on-disk CRD schema and example CRs match the locked model.

- [ ] **Step 1: Write the failing test**

Create `internal/crd/doc.go`:

```go
// Package crd holds tests asserting the shipped CRD definition and example CRs
// match the engine's Spec contract (group chartpress.dev, single-ingress enum,
// 3-workload enum, descriptions).
package crd
```

Create `internal/crd/crd_test.go`:

```go
package crd

import (
	"os"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
	sigsyaml "sigs.k8s.io/yaml"
)

const crdPath = "../../crds/crd-helmchart.yaml"

func readYAML(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]interface{}
	if err := sigsyaml.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return m
}

func nested(t *testing.T, m map[string]interface{}, keys ...string) interface{} {
	t.Helper()
	var cur interface{} = m
	for _, k := range keys {
		mp, ok := cur.(map[string]interface{})
		if !ok {
			t.Fatalf("path %v: %q is not a map", keys, k)
		}
		cur = mp[k]
	}
	return cur
}

func TestCRDGroupAndNameMigrated(t *testing.T) {
	m := readYAML(t, crdPath)
	if g := nested(t, m, "spec", "group"); g != "chartpress.dev" {
		t.Fatalf("spec.group = %v, want chartpress.dev", g)
	}
	if n := nested(t, m, "metadata", "name"); n != "chartpressconfigs.chartpress.dev" {
		t.Fatalf("metadata.name = %v, want chartpressconfigs.chartpress.dev", n)
	}
}

func TestCRDSchemaShape(t *testing.T) {
	m := readYAML(t, crdPath)
	versions := nested(t, m, "spec", "versions").([]interface{})
	schema := versions[0].(map[string]interface{})["schema"].(map[string]interface{})
	specProps := nested(t, schema, "openAPIV3Schema", "properties", "spec", "properties").(map[string]interface{})

	// spec.description present
	if _, ok := specProps["description"]; !ok {
		t.Fatal("spec.description property missing")
	}
	// subcharts[].description present, workload enum is exactly the 3 allowed
	subItems := nested(t, specProps, "subcharts", "items", "properties").(map[string]interface{})
	if _, ok := subItems["description"]; !ok {
		t.Fatal("subcharts[].description property missing")
	}
	wEnum := toStrings(t, nested(t, subItems, "workload", "enum"))
	assertSetEqual(t, wEnum, engine.AllowedWorkloads, "workload enum")

	// rules.ingress is a single string enum; possible_ingresses is gone
	rulesProps := nested(t, specProps, "rules", "properties").(map[string]interface{})
	if _, gone := rulesProps["possible_ingresses"]; gone {
		t.Fatal("rules.possible_ingresses must be removed")
	}
	if typ := nested(t, rulesProps, "ingress", "type"); typ != "string" {
		t.Fatalf("rules.ingress type = %v, want string", typ)
	}
	iEnum := toStrings(t, nested(t, rulesProps, "ingress", "enum"))
	assertSetEqual(t, iEnum, engine.AllowedIngress, "ingress enum")
}

func TestExampleCRsAreValidSpecs(t *testing.T) {
	for _, path := range []string{"../../crds/helmchart-iot.yaml", "../../crds/helmchart-ml.yaml"} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var cr struct {
			APIVersion string      `json:"apiVersion"`
			Spec       engine.Spec `json:"spec"`
		}
		if err := sigsyaml.Unmarshal(b, &cr); err != nil {
			t.Fatalf("unmarshal %s: %v", path, err)
		}
		if cr.APIVersion != "chartpress.dev/v1alpha1" {
			t.Fatalf("%s apiVersion = %q, want chartpress.dev/v1alpha1", path, cr.APIVersion)
		}
		if err := engine.Validate(engine.Normalize(cr.Spec)); err != nil {
			t.Fatalf("%s is not a valid spec: %v", path, err)
		}
	}
}

func toStrings(t *testing.T, v interface{}) []string {
	t.Helper()
	raw, ok := v.([]interface{})
	if !ok {
		t.Fatalf("enum is not a list: %T", v)
	}
	out := make([]string, len(raw))
	for i, e := range raw {
		out[i] = e.(string)
	}
	return out
}

func assertSetEqual(t *testing.T, got, want []string, label string) {
	t.Helper()
	gm := map[string]bool{}
	for _, g := range got {
		gm[g] = true
	}
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
	for _, w := range want {
		if !gm[w] {
			t.Fatalf("%s missing %q (got %v)", label, w, got)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/crd/ -v`
Expected: FAIL — current CRD has group `kriipke.dev`, `possible_ingresses`, a 5-value workload enum, and no descriptions; examples use `kriipke.dev` + `possible_ingresses`.

- [ ] **Step 3: Write minimal implementation**

Replace `crds/crd-helmchart.yaml` with:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: chartpressconfigs.chartpress.dev
spec:
  group: chartpress.dev
  names:
    kind: ChartpressConfig
    plural: chartpressconfigs
    singular: chartpressconfig
    shortNames:
      - cpress
  scope: Namespaced
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                umbrellaChartName:
                  type: string
                  description: "Name of the umbrella Helm chart (kebab-case)."
                description:
                  type: string
                  description: "Human-readable description for the umbrella chart."
                subcharts:
                  type: array
                  description: "List of subcharts and their workload types."
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                        description: "The name of the subchart (kebab-case)."
                      workload:
                        type: string
                        enum: ["deployment", "statefulset", "daemonset"]
                        description: "The workload type for the subchart."
                      description:
                        type: string
                        description: "Human-readable description for this subchart."
                rules:
                  type: object
                  properties:
                    ingress:
                      type: string
                      enum: ["alb", "nginx", "traefik", "istio", "gce", "none"]
                      description: "Platform-wide ingress controller (single)."
                    common_annotations:
                      type: boolean
                      description: "Enable or disable common annotations."
                    linked_templates:
                      type: boolean
                      description: "Should subchart templates link shared umbrella partials?"
                    resource_names_match_chart_name:
                      type: boolean
                      description: "Should resource names match the chart name (no release prefix)?"
                    shared_secrets_config:
                      type: boolean
                      description: "Enable a shared umbrella Secret wired into every subchart."
                    shared_newrelic_config:
                      type: boolean
                      description: "Enable shared New Relic config + license wired into every subchart."
                    generate_umbrella_readme:
                      type: boolean
                      description: "Generate a README for the umbrella chart."
                    generate_subchart_readme:
                      type: boolean
                      description: "Generate READMEs for subcharts."
                    include_docs:
                      type: boolean
                      description: "Include the docs/ directory in the output."
              required: ["umbrellaChartName", "subcharts", "rules"]
            status:
              type: object
              description: "Operator-owned status (written by the chartpress operator)."
              properties:
                phase:
                  type: string
                  enum: ["Pending", "Generating", "Ready", "Failed"]
                  description: "Lifecycle phase of the generated chart."
                observedGeneration:
                  type: integer
                  description: "metadata.generation last reconciled by the operator."
                artifactKey:
                  type: string
                  description: "Object-storage key of the generated chart archive."
                lastGenerated:
                  type: string
                  description: "RFC3339 timestamp of the last successful generation."
                message:
                  type: string
                  description: "Human-readable status detail (error text when Failed)."
      subresources:
        status: {}
```

Replace `crds/helmchart-iot.yaml` with:

```yaml
apiVersion: chartpress.dev/v1alpha1
kind: ChartpressConfig
metadata:
  name: iot-hub
spec:
  umbrellaChartName: iot-hub
  description: "IoT ingestion and device-management platform."
  subcharts:
    - name: device-manager
      workload: deployment
      description: "Device registry and lifecycle API."
    - name: mqtt-broker
      workload: statefulset
      description: "MQTT message broker."
    - name: timescaledb
      workload: statefulset
      description: "Time-series database for telemetry."
  rules:
    ingress: nginx
    common_annotations: true
    linked_templates: true
    resource_names_match_chart_name: true
    shared_secrets_config: false
    shared_newrelic_config: false
    generate_umbrella_readme: false
    generate_subchart_readme: true
    include_docs: true
```

Replace `crds/helmchart-ml.yaml` with:

```yaml
apiVersion: chartpress.dev/v1alpha1
kind: ChartpressConfig
metadata:
  name: ml-platform
spec:
  umbrellaChartName: ml-platform
  description: "Model training and serving platform."
  subcharts:
    - name: trainer
      workload: deployment
      description: "Batch training jobs runner."
    - name: predictor
      workload: deployment
      description: "Online inference service."
    - name: model-db
      workload: statefulset
      description: "Model + metadata store."
    - name: worker
      workload: deployment
      description: "Async task worker."
  rules:
    ingress: istio
    common_annotations: false
    linked_templates: false
    resource_names_match_chart_name: false
    shared_secrets_config: true
    shared_newrelic_config: true
    generate_umbrella_readme: true
    generate_subchart_readme: true
    include_docs: false
```

In `crds/README.md`, apply these edits (the test does not assert the README; update it for accuracy):
- The `## CRD Example` block: change `apiVersion: kriipke.dev/v1alpha1` → `apiVersion: chartpress.dev/v1alpha1`; replace the `rules.possible_ingresses:` list with `ingress: alb`; add `description:` to the umbrella spec and to each subchart.
- `### spec.rules` field reference: replace the `possible_ingresses` bullet with `ingress: Single platform-wide ingress controller (one of alb, nginx, traefik, istio, gce, none).`
- Add a `### spec.description` and `subcharts[].description` mention, and a short `### status` subsection listing phase/observedGeneration/artifactKey/lastGenerated/message.
- The `kubectl apply` group reference and the prose "No such controller ships in this repository yet" line: leave a note that the operator arrives in a later phase.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/crd/ -v && go build ./...`
Expected: PASS (3 tests); build clean.

- [ ] **Step 5: Commit**

```bash
git add crds/ internal/crd/
git commit -m "feat(crd): migrate ChartpressConfig to chartpress.dev, single ingress, 3 workloads, descriptions + status"
```

---

### Task 8: Backend RBAC + downward-API namespace (Helm chart)

**Files:**
- Create: `chart/templates/backend-rbac.yaml`
- Modify: `chart/templates/backend-deployment.yaml`
- Modify: `chart/values.yaml`
- Create: `internal/deploy/doc.go`
- Create: `internal/deploy/chart_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks (renders `chart/` via the Helm Go SDK, mirroring `internal/engine/render_test.go`).
- Produces: a backend ServiceAccount + Role + RoleBinding granting `create/get/list/watch` on `chartpressconfigs.chartpress.dev`, the backend Deployment running as that SA with `POD_NAMESPACE` from the downward API and `OPENAI_MODEL`/`OPENAI_API_KEY` env.

- [ ] **Step 1: Write the failing test**

Create `internal/deploy/doc.go`:

```go
// Package deploy holds tests that render the chartpress Helm chart and assert
// deploy-time wiring (backend RBAC, downward-API namespace).
package deploy
```

Create `internal/deploy/chart_test.go`:

```go
package deploy

import (
	"strings"
	"testing"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

// renderChart loads and renders chart/ with its default values, returning all
// rendered manifests concatenated.
func renderChart(t *testing.T) string {
	t.Helper()
	ch, err := loader.Load("../../chart")
	if err != nil {
		t.Fatalf("load chart: %v", err)
	}
	vals, err := chartutil.ToRenderValues(ch, chartutil.Values{}, chartutil.ReleaseOptions{
		Name:      "chartpress",
		Namespace: "chartpress-system",
	}, nil)
	if err != nil {
		t.Fatalf("ToRenderValues: %v", err)
	}
	out, err := engine.Render(ch, vals)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	var b strings.Builder
	for _, v := range out {
		b.WriteString(v)
		b.WriteString("\n---\n")
	}
	return b.String()
}

func TestBackendRBACGrantsChartpressConfigs(t *testing.T) {
	man := renderChart(t)
	for _, want := range []string{
		"kind: Role",
		"kind: RoleBinding",
		"kind: ServiceAccount",
		"chartpress.dev",
		"chartpressconfigs",
		"create",
		"watch",
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("rendered chart missing %q", want)
		}
	}
}

func TestBackendDeploymentHasDownwardNamespaceAndSA(t *testing.T) {
	man := renderChart(t)
	for _, want := range []string{
		"serviceAccountName: chartpress-backend",
		"POD_NAMESPACE",
		"fieldRef",
		"metadata.namespace",
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("backend deployment missing %q", want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/deploy/ -v`
Expected: FAIL — no RBAC template; deployment lacks the SA + downward env.

- [ ] **Step 3: Write minimal implementation**

Create `chart/templates/backend-rbac.yaml`:

```yaml
{{- if .Values.backend.rbac.create }}
# The backend creates/reads ChartpressConfig CRs in its own namespace so the
# web app can submit specs and browse generated charts.
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ .Values.backend.serviceAccount.name }}
  namespace: {{ .Release.Namespace }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: chartpress-backend
  namespace: {{ .Release.Namespace }}
rules:
  - apiGroups: ["chartpress.dev"]
    resources: ["chartpressconfigs"]
    verbs: ["create", "get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: chartpress-backend
  namespace: {{ .Release.Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: chartpress-backend
subjects:
  - kind: ServiceAccount
    name: {{ .Values.backend.serviceAccount.name }}
    namespace: {{ .Release.Namespace }}
{{- end }}
```

Replace `chart/templates/backend-deployment.yaml` with (adds `serviceAccountName`, `POD_NAMESPACE` downward env, and OpenAI env):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chartpress-backend
  labels:
    app: chartpress-backend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: chartpress-backend
  template:
    metadata:
      labels:
        app: chartpress-backend
    spec:
      serviceAccountName: {{ .Values.backend.serviceAccount.name }}
      containers:
        - name: backend
          image: "{{ .Values.backend.image.repository }}:{{ .Values.backend.image.tag }}"
          ports:
            - containerPort: 8080
          env:
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: OPENAI_MODEL
              value: {{ .Values.backend.openai.model | quote }}
            {{- if .Values.backend.openai.apiKeySecret.name }}
            - name: OPENAI_API_KEY
              valueFrom:
                secretKeyRef:
                  name: {{ .Values.backend.openai.apiKeySecret.name }}
                  key: {{ .Values.backend.openai.apiKeySecret.key }}
            {{- end }}
          volumeMounts:
            - name: chartpress-config
              mountPath: /app/chartpress.yaml
              subPath: chartpress.yaml
            # /app/templates is intentionally NOT mounted: the chart templates
            # are baked into the image and an emptyDir mount would hide them.
      volumes:
        - name: chartpress-config
          configMap:
            name: chartpress-config
```

> Note: the old `output-volume` emptyDir is removed — the async backend no longer writes local chart output (rendering moves to the Phase 3 operator).

Add to `chart/values.yaml` under the `backend:` block (keep existing `image`/`service` keys):

```yaml
  serviceAccount:
    name: chartpress-backend
  rbac:
    create: true
  openai:
    model: gpt-4.1
    apiKeySecret:
      name: ""   # set to a Secret name to inject OPENAI_API_KEY; empty omits the env
      key: api-key
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/deploy/ -v && go build ./...`
Expected: PASS (2 tests); build clean.

- [ ] **Step 5: Commit**

```bash
git add chart/templates/backend-rbac.yaml chart/templates/backend-deployment.yaml chart/values.yaml internal/deploy/
git commit -m "feat(chart): backend RBAC for chartpressconfigs + downward-API POD_NAMESPACE + OpenAI env"
```

---

## Final verification (after all tasks)

- [ ] `go build ./...` — clean.
- [ ] `go test ./...` — all packages green (engine 22-test suite untouched; new server/crd/deploy suites pass).
- [ ] `grep -rn 'kriipke.dev' crds/` — no matches (group fully migrated).
- [ ] `grep -rnE 'zipOutputDir|handleDownload|possible_ingresses' internal/ crds/` — no matches.

## Self-Review (completed during planning)

- **Spec coverage:** §3.1 spec + `*bool` defaulting → Task 1; §3.2 manifest wrap → Task 2; §5 `/generate` SSA apply → Tasks 3+4; §5 `/charts`+`/charts/<name>` → Task 5; §5 `/text-to-config` (OpenAI Responses, gpt-4.1, strict schema) → Task 6; §9 CRD migration + examples + README → Task 7; §10 backend RBAC + `POD_NAMESPACE` → Task 8; §7 validation (engine.Validate reused in /generate) → Task 4; §8 object storage → **deferred to Phase 3 per locked decision** (downloadUrl empty, Task 5). Out-of-scope (operator/finalizer/S3 upload, frontend) correctly excluded.
- **Type consistency:** `decodeSpec→engine.Spec`, `wrapManifest→*unstructured.Unstructured`, `manifestYAML(*unstructured)→string`, `Applier.Apply(ctx,ns,obj)`, `ChartLister.List/Get`, `Drafter.Draft(ctx,prompt)→engine.Spec`, `generateResponse{name,namespace,phase,manifestYaml}`, `chartSummary{name,phase,subchartCount,lastGenerated,message,downloadUrl}` are used identically across the tasks that define and consume them.
- **Placeholder scan:** no TBD/TODO; every code step is complete and runnable.
- **Cross-task wiring note:** `Server.Drafter` is set in `Start()` only in Task 6; before Task 6 it is nil but the `/text-to-config` handler (its only consumer) does not exist yet, so no nil deref. `Handler()` gains one route per task (Task 4 `/generate`, Task 5 `/charts`+`/charts/`, Task 6 `/text-to-config`).
