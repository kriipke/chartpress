# Phase 3 — Operator + Object Storage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the cluster operator that watches `ChartpressConfig` CRs, renders the rich chart from `.spec` via `internal/engine`, uploads the zip to S3-compatible object storage, and writes `status`; plus the shared object-storage package and the deferred Phase-2 backend wiring that mints presigned `downloadUrl`s from `status.artifactKey`.

**Architecture:** A new `cmd/operator` binary (same container image as the backend, `command: [operator]`) runs a `client-go` dynamic SharedInformer + rate-limiting workqueue (no controller-runtime; 30s resync as the level-based safety net). A pure `Reconciler` decodes `.spec` → `engine.Spec`, normalizes/validates defensively, renders via `engine.GenerateChart` into a temp dir, zips it, uploads to `charts/<name>.zip` (overwrite), and writes the status subresource. A shared `internal/objectstore` (minio-go) sits behind an `Uploader` (operator) and `Presigner` (backend) interface so every test uses in-memory fakes — no real bucket. The API identity (`chartpress.dev/v1alpha1`, `chartpressconfigs`) is lifted into a single `internal/apis` package consumed by both the backend and the operator.

**Tech Stack:** Go 1.23, `k8s.io/client-go` v0.32.2 (`dynamic`, `dynamic/dynamicinformer`, `dynamic/fake`, `util/workqueue`, `tools/cache`), `k8s.io/apimachinery` v0.32.2 (`unstructured`), `github.com/minio/minio-go/v7` (NEW direct dep), Helm v3 Go SDK (deploy render test), `archive/zip`.

## Global Constraints

- **API identity (single source of truth):** group `chartpress.dev`, version `v1alpha1`, kind `ChartpressConfig`, resource `chartpressconfigs`. Lifted into `internal/apis` this phase; the backend's `internal/server/manifest.go` is repointed to it. Do **not** redefine the group anywhere else.
- **GVR:** `schema.GroupVersionResource{Group: "chartpress.dev", Version: "v1alpha1", Resource: "chartpressconfigs"}` = `apis.GVR`.
- **Field managers:** backend `chartpress-backend` (`apis.FieldManagerBackend`); operator `chartpress-operator` (`apis.FieldManagerOperator`) — distinct.
- **Finalizer:** `chartpress.dev/artifact-cleanup` (`apis.FinalizerArtifactCleanup`).
- **Object storage (BYO/external, S3-compatible):** `github.com/minio/minio-go/v7`. Config via env/Secret: `S3_ENDPOINT`, `S3_BUCKET`, `S3_REGION`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_USE_SSL`. **No bundled MinIO.** Key scheme `charts/<name>.zip`, latest-per-CR (overwrite), no history.
- **Watch mechanism (locked decision 1):** dynamic `SharedInformerFactory` (event-driven) → client-go rate-limiting workqueue → worker calls `Reconcile`. **30s resync** as the level-based safety net. No controller-runtime; no leader election; **one replica**.
- **Render path (locked decision 2):** `engine.GenerateChart(spec, templatesDir, tmpRoot)` writes the chart to an `os.MkdirTemp` dir; then zip that directory tree. Temp dir removed on `defer`.
- **Failed-retry semantics (locked decision 3):** stamp `status.observedGeneration = metadata.generation` **only on success** (phase `Ready`). On failure set phase `Failed` + `message` and leave `observedGeneration` stale, so the next reconcile retries; the workqueue rate-limiter backs off. Short-circuit condition: `status.phase == "Ready" && status.observedGeneration == metadata.generation`.
- **Status writes (locked decision 5):** dynamic client `UpdateStatus` on the status subresource (already present in the CRD). Finalizer edits go through `Update` on the main resource.
- **Scope/tests (locked decision 4):** one shared `internal/objectstore` for both operator (upload) and backend (presign). The backend `downloadUrl` presign wiring (deferred Phase-2 item) lands **this phase**: `summarize` mints a fresh presigned GET from `status.artifactKey` **only when `phase == "Ready"`**, empty otherwise. All S3 behind interfaces, tested with in-memory fakes — no real bucket, no testcontainer.
- **Templates baked into the image:** the operator reads chart templates from `CHARTPRESS_TEMPLATES_DIR` (default `templates`, baked at `/app/templates`). It contains `umbrella/` and `subchart/` subdirs.
- **Engine API consumed (do not modify):** `engine.Spec`/`Subchart`/`Rules`, `engine.Normalize`, `engine.Validate`, `engine.GenerateChart(spec engine.Spec, templatesDir, outputRoot string) (string, error)`, `engine.DefaultRules()`.
- **Keep green throughout:** `go build ./...` clean after every task; the Phase-1 engine tests and Phase-2 `internal/server` / `internal/crd` / `internal/deploy` tests stay green; run `go mod tidy` whenever a task adds a direct import.

## File Structure

New package `internal/apis/` (shared API identity — leaf, no internal deps):
- `apis.go` — group/version/kind/resource consts, `GVR`, finalizer, field-manager consts.

New package `internal/objectstore/` (shared S3 wrapper — leaf, only minio-go):
- `objectstore.go` — `Config`, `ConfigFromEnv`, `Uploader`/`Presigner` interfaces, minio-backed `Client` (`New`, `Upload`, `Remove`, `PresignGet`).

New package `internal/operator/` (the controller):
- `decode.go` — `decodeSpec(obj) (engine.Spec, error)`.
- `render.go` — `Renderer` interface, `chartRenderer` (GenerateChart → temp dir → `zipDir`).
- `reconcile.go` — `Reconciler`, `CRClient` interface, finalizer/short-circuit/state-machine.
- `client.go` — `dynamicCRClient` (Update/UpdateStatus over `dynamic.Interface` + `apis.GVR`).
- `controller.go` — informer + workqueue wiring, `NewController`, `Run`, `Start()`, `namespaceFromEnv`, `templatesDir`.

New binary:
- `cmd/operator/main.go` — `func main(){ operator.Start() }`.

Modified (Phase-2 code):
- `internal/server/manifest.go` — repoint consts to `internal/apis`.
- `internal/server/k8s.go` — add consumer-side `Presigner` interface (alongside `Applier`/`ChartLister`/`Drafter`).
- `internal/server/server.go` — add `Presigner` field; wire `objectstore.New` in `Start()`.
- `internal/server/charts.go` — `summarize(ctx, p, obj)` mints `downloadUrl` when `Ready`.
- `chart/templates/_helpers.tpl` — add `chartpress.s3env` partial.
- `chart/templates/backend-deployment.yaml` — add S3 env.
- `chart/values.yaml` — add `operator:` and `s3:` blocks.
- `Dockerfile` — build + copy the `operator` binary.

New chart files:
- `chart/templates/operator-deployment.yaml`, `chart/templates/operator-rbac.yaml`, `chart/templates/s3-secret.yaml`.

New/extended tests: `internal/apis/apis_test.go`, `internal/objectstore/objectstore_test.go`, `internal/operator/{render,reconcile,client}_test.go`, `internal/server/charts_test.go` (extend), `internal/deploy/operator_test.go`.

## Parallelization (for subagent-driven execution)

- **Wave 0 (sequential):** Task 1 (`internal/apis` + backend repoint) — unblocks the operator and locks the single source of truth.
- **Wave 1 (parallel worktrees, disjoint files):** Tasks 2–3 (`internal/objectstore`) ‖ Task 9 (`chart/` + `internal/deploy/operator_test.go`, pure-YAML render, independent of Go packages).
- **Wave 2 (parallel worktrees, after Wave 0 + objectstore):** Tasks 4→5→6→7 (`internal/operator` + `cmd/operator`, sequential within the track) ‖ Task 8 (`internal/server` presign wiring).
- **Wave 3 (sequential, integration):** Task 10 (`Dockerfile` + whole-tree `go build`/`go test`/`go mod tidy` + `helm lint`).

Run the spec+quality reviewer after each task; run reviewers in parallel (read-only). Do the final whole-branch review after Task 10.

---

### Task 1: Shared `internal/apis` package + repoint backend consts

**Files:**
- Create: `internal/apis/apis.go`
- Test: `internal/apis/apis_test.go`
- Modify: `internal/server/manifest.go`

**Interfaces:**
- Produces: package `apis` with `Group`, `Version`, `GroupVersion`, `Kind`, `Resource` consts; `var GVR schema.GroupVersionResource`; `FinalizerArtifactCleanup`, `FieldManagerBackend`, `FieldManagerOperator` consts.
- Consumed by: `internal/server` (this task), `internal/operator` (Tasks 4–7).

- [ ] **Step 1: Write the failing test**

```go
// internal/apis/apis_test.go
package apis

import "testing"

func TestGVRAndIdentity(t *testing.T) {
	if Group != "chartpress.dev" || Version != "v1alpha1" {
		t.Fatalf("group/version = %s/%s", Group, Version)
	}
	if GroupVersion != "chartpress.dev/v1alpha1" {
		t.Fatalf("groupVersion = %q", GroupVersion)
	}
	if Kind != "ChartpressConfig" || Resource != "chartpressconfigs" {
		t.Fatalf("kind/resource = %s/%s", Kind, Resource)
	}
	if GVR.Group != Group || GVR.Version != Version || GVR.Resource != Resource {
		t.Fatalf("GVR = %+v", GVR)
	}
}

func TestFinalizerAndFieldManagers(t *testing.T) {
	if FinalizerArtifactCleanup != "chartpress.dev/artifact-cleanup" {
		t.Fatalf("finalizer = %q", FinalizerArtifactCleanup)
	}
	if FieldManagerBackend != "chartpress-backend" || FieldManagerOperator != "chartpress-operator" {
		t.Fatalf("field managers = %s / %s", FieldManagerBackend, FieldManagerOperator)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/apis/ -v`
Expected: FAIL — package/symbols undefined (build error).

- [ ] **Step 3: Write minimal implementation**

```go
// internal/apis/apis.go
// Package apis is the single source of truth for the ChartpressConfig API
// identity, shared by the backend (apply/list) and the operator (watch/reconcile).
package apis

import "k8s.io/apimachinery/pkg/runtime/schema"

const (
	Group        = "chartpress.dev"
	Version      = "v1alpha1"
	GroupVersion = Group + "/" + Version
	Kind         = "ChartpressConfig"
	Resource     = "chartpressconfigs"

	FinalizerArtifactCleanup = "chartpress.dev/artifact-cleanup"
	FieldManagerBackend      = "chartpress-backend"
	FieldManagerOperator     = "chartpress-operator"
)

// GVR is the GroupVersionResource for ChartpressConfig CRs.
var GVR = schema.GroupVersionResource{Group: Group, Version: Version, Resource: Resource}
```

Repoint `internal/server/manifest.go` — replace the const/var block so values flow from `apis` (keep the local identifiers so the rest of `internal/server` is untouched):

```go
// internal/server/manifest.go  (replace the existing const + var chartpressGVR block)
import (
	// ...existing imports...
	"github.com/kriipke/chartpress/internal/apis"
)

const (
	apiGroup             = apis.Group
	apiVersionV1alpha1   = apis.GroupVersion
	kindChartpressConfig = apis.Kind
	fieldManager         = apis.FieldManagerBackend
)

var chartpressGVR = apis.GVR
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go mod tidy && go test ./internal/apis/ ./internal/server/ -v`
Expected: `internal/apis` PASS; `internal/server` still green (values unchanged).

- [ ] **Step 5: Commit**

```bash
git add internal/apis/ internal/server/manifest.go go.mod go.sum
git commit -m "feat(apis): shared ChartpressConfig identity; repoint backend consts"
```

---

### Task 2: `internal/objectstore` config + `ConfigFromEnv`

**Files:**
- Create: `internal/objectstore/objectstore.go` (Config + ConfigFromEnv only this task)
- Test: `internal/objectstore/objectstore_test.go`

**Interfaces:**
- Produces: `type Config struct { Endpoint, Bucket, Region, AccessKey, SecretKey string; UseSSL bool }`; `func ConfigFromEnv() Config`.

- [ ] **Step 1: Write the failing test**

```go
// internal/objectstore/objectstore_test.go
package objectstore

import "testing"

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("S3_ENDPOINT", "s3.example.com")
	t.Setenv("S3_BUCKET", "charts")
	t.Setenv("S3_REGION", "us-west-2")
	t.Setenv("S3_ACCESS_KEY", "AK")
	t.Setenv("S3_SECRET_KEY", "SK")
	t.Setenv("S3_USE_SSL", "true")

	c := ConfigFromEnv()
	if c.Endpoint != "s3.example.com" || c.Bucket != "charts" || c.Region != "us-west-2" ||
		c.AccessKey != "AK" || c.SecretKey != "SK" || !c.UseSSL {
		t.Fatalf("config = %+v", c)
	}
}

func TestConfigFromEnvUseSSLDefaultsFalse(t *testing.T) {
	t.Setenv("S3_USE_SSL", "")
	if ConfigFromEnv().UseSSL {
		t.Fatal("useSSL must be false when S3_USE_SSL is unset")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/objectstore/ -v`
Expected: FAIL — `undefined: ConfigFromEnv` / `Config`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/objectstore/objectstore.go
// Package objectstore wraps an S3-compatible bucket (AWS S3 / R2 / MinIO) behind
// an Uploader (operator) and a Presigner (backend), so callers test against fakes
// and never need a real bucket.
package objectstore

import (
	"os"
	"strings"
)

// Config holds S3-compatible connection settings (BYO/external bucket).
type Config struct {
	Endpoint  string
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// ConfigFromEnv reads the S3_* environment variables (set from the chart's S3
// Secret/values). UseSSL is true only when S3_USE_SSL == "true" (case-insensitive).
func ConfigFromEnv() Config {
	return Config{
		Endpoint:  os.Getenv("S3_ENDPOINT"),
		Bucket:    os.Getenv("S3_BUCKET"),
		Region:    os.Getenv("S3_REGION"),
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
		UseSSL:    strings.EqualFold(os.Getenv("S3_USE_SSL"), "true"),
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/objectstore/ -v`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/objectstore/objectstore.go internal/objectstore/objectstore_test.go
git commit -m "feat(objectstore): S3 Config + ConfigFromEnv"
```

---

### Task 3: `internal/objectstore` minio client (`Uploader` + `Presigner`)

**Files:**
- Modify: `internal/objectstore/objectstore.go` (add interfaces + `Client`)
- Test: `internal/objectstore/objectstore_test.go` (add presign + interface tests)

**Interfaces:**
- Produces:
  - `type Uploader interface { Upload(ctx context.Context, key string, r io.Reader, size int64) error; Remove(ctx context.Context, key string) error }`
  - `type Presigner interface { PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) }`
  - `type Client struct{ ... }` implementing both; `func New(cfg Config) (*Client, error)`.

- [ ] **Step 1: Write the failing test**

```go
// internal/objectstore/objectstore_test.go  (add to the existing file)
import (
	"context"
	"strings"
	"time"
)

// Compile-time proof the minio Client satisfies both consumer interfaces.
var (
	_ Uploader  = (*Client)(nil)
	_ Presigner = (*Client)(nil)
)

func TestNewRejectsMissingEndpointOrBucket(t *testing.T) {
	if _, err := New(Config{Bucket: "b"}); err == nil {
		t.Fatal("expected error when endpoint is empty")
	}
	if _, err := New(Config{Endpoint: "s3.example.com"}); err == nil {
		t.Fatal("expected error when bucket is empty")
	}
}

// PresignedGetObject computes the URL+signature locally (no network), so we can
// assert its shape against the real minio client without a bucket.
func TestPresignGetProducesSignedURL(t *testing.T) {
	c, err := New(Config{
		Endpoint: "s3.example.com", Bucket: "my-bucket", Region: "us-east-1",
		AccessKey: "AKIAEXAMPLE", SecretKey: "secretsecretsecret", UseSSL: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	u, err := c.PresignGet(context.Background(), "charts/demo.zip", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
	for _, want := range []string{
		"https://s3.example.com/my-bucket/charts/demo.zip",
		"X-Amz-Signature=",
		"X-Amz-Expires=900",
	} {
		if !strings.Contains(u, want) {
			t.Fatalf("presigned url %q missing %q", u, want)
		}
	}
}
```

> Note: for a non-AWS custom endpoint minio defaults to path-style URLs (`<endpoint>/<bucket>/<key>`). If a future minio version switches the default, adjust the host assertion to virtual-host style — the `X-Amz-*` assertions still hold.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/objectstore/ -run 'TestNew|TestPresignGet' -v`
Expected: FAIL — `undefined: New` / `Client` / `Uploader` / `Presigner`.

- [ ] **Step 3: Write minimal implementation**

First add the dependency:

```bash
go get github.com/minio/minio-go/v7
```

Append to `internal/objectstore/objectstore.go`:

```go
import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Uploader writes (and removes) chart archives. Used by the operator.
type Uploader interface {
	Upload(ctx context.Context, key string, r io.Reader, size int64) error
	Remove(ctx context.Context, key string) error
}

// Presigner mints time-limited GET URLs for chart archives. Used by the backend.
type Presigner interface {
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// Client is the minio-backed implementation of both Uploader and Presigner.
type Client struct {
	mc     *minio.Client
	bucket string
}

// New builds a minio client for the configured bucket. It does not connect;
// connection errors surface on the first Upload/Remove call (PresignGet is local).
func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("objectstore: endpoint is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("objectstore: bucket is required")
	}
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("objectstore: %w", err)
	}
	return &Client{mc: mc, bucket: cfg.Bucket}, nil
}

func (c *Client) Upload(ctx context.Context, key string, r io.Reader, size int64) error {
	_, err := c.mc.PutObject(ctx, c.bucket, key, r, size, minio.PutObjectOptions{ContentType: "application/zip"})
	return err
}

func (c *Client) Remove(ctx context.Context, key string) error {
	return c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}

func (c *Client) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, c.bucket, key, expiry, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go mod tidy && go test ./internal/objectstore/ -v && go build ./...`
Expected: PASS (all objectstore tests); build clean. `go mod tidy` promotes `github.com/minio/minio-go/v7` to a direct dep.

- [ ] **Step 5: Commit**

```bash
git add internal/objectstore/ go.mod go.sum
git commit -m "feat(objectstore): minio Uploader + Presigner client"
```

---

### Task 4: Operator spec decode + chart render to zip

**Files:**
- Create: `internal/operator/decode.go`
- Create: `internal/operator/render.go`
- Test: `internal/operator/render_test.go`

**Interfaces:**
- Consumes: `engine.Spec`/`Normalize`/`GenerateChart` (engine); `unstructured`.
- Produces:
  - `func decodeSpec(obj *unstructured.Unstructured) (engine.Spec, error)` — JSON round-trips `.spec` into a normalized `engine.Spec`.
  - `type Renderer interface { RenderZip(spec engine.Spec) ([]byte, error) }`
  - `type chartRenderer struct{ templatesDir string }` implementing `Renderer` (GenerateChart → temp dir → `zipDir`).

- [ ] **Step 1: Write the failing test**

```go
// internal/operator/render_test.go
package operator

import (
	"archive/zip"
	"bytes"
	"path/filepath"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
)

func zipEntries(t *testing.T, b []byte) map[string]bool {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	return names
}

func TestChartRendererProducesValidChartZip(t *testing.T) {
	spec := engine.Normalize(engine.Spec{
		UmbrellaChartName: "demo-platform",
		Subcharts:         []engine.Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             engine.DefaultRules(),
	})
	r := &chartRenderer{templatesDir: filepath.Join("..", "..", "templates")}

	b, err := r.RenderZip(spec)
	if err != nil {
		t.Fatalf("RenderZip: %v", err)
	}
	names := zipEntries(t, b)
	for _, want := range []string{
		"demo-platform/Chart.yaml",
		"demo-platform/charts/api/Chart.yaml",
	} {
		if !names[want] {
			t.Fatalf("zip missing %q; entries = %v", want, names)
		}
	}
}

func TestDecodeSpecFromUnstructured(t *testing.T) {
	obj := crObj("shop", 1) // helper defined in reconcile_test.go (Task 5)
	spec, err := decodeSpec(obj)
	if err != nil {
		t.Fatalf("decodeSpec: %v", err)
	}
	if spec.UmbrellaChartName != "shop" || len(spec.Subcharts) != 1 ||
		spec.Subcharts[0].Workload != "deployment" || spec.Rules.Ingress != "alb" {
		t.Fatalf("decoded spec = %+v", spec)
	}
}
```

> `crObj` is the shared test helper added in Task 5 (`reconcile_test.go`); both files are in `package operator`. Implement Task 4 and Task 5 in the same track so the helper exists when the suite runs. If running Task 4's test before Task 5, temporarily inline a minimal `crObj` and remove it when Task 5 lands.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/operator/ -run 'TestChartRenderer|TestDecodeSpec' -v`
Expected: FAIL — `undefined: chartRenderer` / `decodeSpec`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/operator/decode.go
package operator

import (
	"encoding/json"
	"fmt"

	"github.com/kriipke/chartpress/internal/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// decodeSpec extracts the CR's .spec and JSON round-trips it into a normalized
// engine.Spec. The backend persists a complete, defaulted rules block, so this is
// a re-hydration; Normalize is applied defensively. Validation is the reconciler's.
func decodeSpec(obj *unstructured.Unstructured) (engine.Spec, error) {
	raw, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return engine.Spec{}, fmt.Errorf("read .spec: %w", err)
	}
	if !found {
		return engine.Spec{}, fmt.Errorf("ChartpressConfig %q has no .spec", obj.GetName())
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return engine.Spec{}, err
	}
	var spec engine.Spec
	if err := json.Unmarshal(b, &spec); err != nil {
		return engine.Spec{}, err
	}
	return engine.Normalize(spec), nil
}
```

```go
// internal/operator/render.go
package operator

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kriipke/chartpress/internal/engine"
)

// Renderer turns a Spec into the chart archive bytes the operator uploads.
type Renderer interface {
	RenderZip(spec engine.Spec) ([]byte, error)
}

// chartRenderer renders via engine.GenerateChart (reusing Helm's on-disk SaveDir
// layout, including nested subcharts) into a temp dir, then zips the directory.
type chartRenderer struct {
	templatesDir string
}

func (r *chartRenderer) RenderZip(spec engine.Spec) ([]byte, error) {
	tmp, err := os.MkdirTemp("", "chartpress-render-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	chartDir, err := engine.GenerateChart(spec, r.templatesDir, tmp)
	if err != nil {
		return nil, err
	}
	return zipDir(chartDir)
}

// zipDir walks root and returns a zip whose entries are forward-slash paths
// relative to root's parent (so the archive root is the chart directory name).
func zipDir(root string) ([]byte, error) {
	parent := filepath.Dir(root)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(parent, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	})
	if walkErr != nil {
		_ = zw.Close()
		return nil, walkErr
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/operator/ -run 'TestChartRenderer|TestDecodeSpec' -v && go build ./...`
Expected: PASS; build clean.

- [ ] **Step 5: Commit**

```bash
git add internal/operator/decode.go internal/operator/render.go internal/operator/render_test.go
git commit -m "feat(operator): decode .spec and render chart to zip"
```

---

### Task 5: Operator reconcile state machine

**Files:**
- Create: `internal/operator/reconcile.go`
- Test: `internal/operator/reconcile_test.go`

**Interfaces:**
- Consumes: `apis.GVR`/`apis.FinalizerArtifactCleanup`; `decodeSpec` (T4); `Renderer` (T4); `objectstore.Uploader` (T3); `engine.Validate`.
- Produces:
  - `type CRClient interface { Update(ctx, ns, obj) (*unstructured.Unstructured, error); UpdateStatus(ctx, ns, obj) (*unstructured.Unstructured, error) }`
  - `type Reconciler struct { Client CRClient; Renderer Renderer; Uploader objectstore.Uploader; Namespace string; Now func() time.Time }`
  - `func (r *Reconciler) Reconcile(ctx context.Context, obj *unstructured.Unstructured) error`
  - phase consts; finalizer/key helpers; `crObj` test helper (shared with Task 4).

- [ ] **Step 1: Write the failing test**

```go
// internal/operator/reconcile_test.go
package operator

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/kriipke/chartpress/internal/apis"
	"github.com/kriipke/chartpress/internal/engine"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// crObj builds a minimal valid ChartpressConfig CR (shared across operator tests).
func crObj(name string, generation int64) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apis.GroupVersion,
		"kind":       apis.Kind,
		"metadata": map[string]interface{}{
			"name":       name,
			"namespace":  "chartpress-system",
			"generation": generation,
		},
		"spec": map[string]interface{}{
			"umbrellaChartName": name,
			"subcharts": []interface{}{
				map[string]interface{}{"name": "api", "workload": "deployment"},
			},
			"rules": map[string]interface{}{
				"ingress":                  "alb",
				"linked_templates":         true,
				"generate_umbrella_readme": true,
				"generate_subchart_readme": true,
				"include_docs":             true,
			},
		},
	}}
}

type fakeCRClient struct {
	obj       *unstructured.Unstructured
	statusLog []string
	updateErr error
	statusErr error
}

func (f *fakeCRClient) Update(_ context.Context, _ string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	f.obj = obj.DeepCopy()
	return f.obj.DeepCopy(), nil
}

func (f *fakeCRClient) UpdateStatus(_ context.Context, _ string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if f.statusErr != nil {
		return nil, f.statusErr
	}
	f.obj = obj.DeepCopy()
	phase, _, _ := unstructured.NestedString(f.obj.Object, "status", "phase")
	f.statusLog = append(f.statusLog, phase)
	return f.obj.DeepCopy(), nil
}

type fakeUploader struct {
	uploaded  map[string][]byte
	removed   []string
	uploadErr error
}

func (u *fakeUploader) Upload(_ context.Context, key string, r io.Reader, _ int64) error {
	if u.uploadErr != nil {
		return u.uploadErr
	}
	b, _ := io.ReadAll(r)
	if u.uploaded == nil {
		u.uploaded = map[string][]byte{}
	}
	u.uploaded[key] = b
	return nil
}

func (u *fakeUploader) Remove(_ context.Context, key string) error {
	u.removed = append(u.removed, key)
	return nil
}

type fakeRenderer struct {
	zip []byte
	err error
}

func (r fakeRenderer) RenderZip(engine.Spec) ([]byte, error) { return r.zip, r.err }

func fixedClock() func() time.Time {
	return func() time.Time { return time.Date(2026, 6, 30, 3, 10, 0, 0, time.UTC) }
}

func TestReconcileHappyPathSetsReady(t *testing.T) {
	fc := &fakeCRClient{}
	up := &fakeUploader{}
	r := &Reconciler{Client: fc, Renderer: fakeRenderer{zip: []byte("PK-zip-bytes")}, Uploader: up, Namespace: "chartpress-system", Now: fixedClock()}

	if err := r.Reconcile(context.Background(), crObj("demo", 1)); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if fc.obj == nil || !hasFinalizer(fc.obj, apis.FinalizerArtifactCleanup) {
		t.Fatal("finalizer was not added")
	}
	if len(fc.statusLog) < 2 || fc.statusLog[0] != phaseGenerating || fc.statusLog[len(fc.statusLog)-1] != phaseReady {
		t.Fatalf("status transitions = %v, want [Generating ... Ready]", fc.statusLog)
	}
	if _, ok := up.uploaded["charts/demo.zip"]; !ok {
		t.Fatalf("uploaded keys = %v, want charts/demo.zip", up.uploaded)
	}
	phase, _, _ := unstructured.NestedString(fc.obj.Object, "status", "phase")
	og, _, _ := unstructured.NestedInt64(fc.obj.Object, "status", "observedGeneration")
	ak, _, _ := unstructured.NestedString(fc.obj.Object, "status", "artifactKey")
	lg, _, _ := unstructured.NestedString(fc.obj.Object, "status", "lastGenerated")
	if phase != phaseReady || og != 1 || ak != "charts/demo.zip" || lg != "2026-06-30T03:10:00Z" {
		t.Fatalf("final status: phase=%s og=%d ak=%s lg=%s", phase, og, ak, lg)
	}
}

func TestReconcileShortCircuitsWhenCurrent(t *testing.T) {
	fc := &fakeCRClient{}
	up := &fakeUploader{}
	r := &Reconciler{Client: fc, Renderer: fakeRenderer{zip: []byte("x")}, Uploader: up, Namespace: "chartpress-system", Now: fixedClock()}

	obj := crObj("demo", 2)
	obj.SetFinalizers([]string{apis.FinalizerArtifactCleanup})
	_ = unstructured.SetNestedField(obj.Object, phaseReady, "status", "phase")
	_ = unstructured.SetNestedField(obj.Object, int64(2), "status", "observedGeneration")

	if err := r.Reconcile(context.Background(), obj); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(up.uploaded) != 0 || len(fc.statusLog) != 0 {
		t.Fatalf("expected no work; uploads=%v statusLog=%v", up.uploaded, fc.statusLog)
	}
}

func TestReconcileRenderFailureSetsFailedAndRequeues(t *testing.T) {
	fc := &fakeCRClient{}
	up := &fakeUploader{}
	r := &Reconciler{Client: fc, Renderer: fakeRenderer{err: io.ErrUnexpectedEOF}, Uploader: up, Namespace: "chartpress-system", Now: fixedClock()}

	err := r.Reconcile(context.Background(), crObj("demo", 1))
	if err == nil {
		t.Fatal("expected an error so the workqueue requeues with backoff")
	}
	phase, _, _ := unstructured.NestedString(fc.obj.Object, "status", "phase")
	msg, _, _ := unstructured.NestedString(fc.obj.Object, "status", "message")
	_, ogFound, _ := unstructured.NestedInt64(fc.obj.Object, "status", "observedGeneration")
	if phase != phaseFailed || msg == "" || ogFound {
		t.Fatalf("phase=%s msg=%q observedGenerationSet=%v (want Failed, msg set, og stale)", phase, msg, ogFound)
	}
}

func TestReconcileDeleteRemovesArtifactAndFinalizer(t *testing.T) {
	fc := &fakeCRClient{}
	up := &fakeUploader{}
	r := &Reconciler{Client: fc, Renderer: fakeRenderer{zip: []byte("x")}, Uploader: up, Namespace: "chartpress-system", Now: fixedClock()}

	obj := crObj("demo", 1)
	obj.SetFinalizers([]string{apis.FinalizerArtifactCleanup})
	now := metav1.Now()
	obj.SetDeletionTimestamp(&now)

	if err := r.Reconcile(context.Background(), obj); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(up.removed) != 1 || up.removed[0] != "charts/demo.zip" {
		t.Fatalf("removed = %v, want [charts/demo.zip]", up.removed)
	}
	if hasFinalizer(fc.obj, apis.FinalizerArtifactCleanup) {
		t.Fatal("finalizer should have been dropped after cleanup")
	}
}

func TestReconcileUploadFailureSetsFailed(t *testing.T) {
	fc := &fakeCRClient{}
	up := &fakeUploader{uploadErr: io.ErrClosedPipe}
	r := &Reconciler{Client: fc, Renderer: fakeRenderer{zip: []byte("x")}, Uploader: up, Namespace: "chartpress-system", Now: fixedClock()}

	if err := r.Reconcile(context.Background(), crObj("demo", 1)); err == nil {
		t.Fatal("expected upload error to requeue")
	}
	phase, _, _ := unstructured.NestedString(fc.obj.Object, "status", "phase")
	if phase != phaseFailed {
		t.Fatalf("phase = %s, want Failed", phase)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/operator/ -run TestReconcile -v`
Expected: FAIL — `undefined: Reconciler` / `hasFinalizer` / `phaseGenerating` etc.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/operator/reconcile.go
package operator

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/kriipke/chartpress/internal/apis"
	"github.com/kriipke/chartpress/internal/engine"
	"github.com/kriipke/chartpress/internal/objectstore"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	phasePending    = "Pending"
	phaseGenerating = "Generating"
	phaseReady      = "Ready"
	phaseFailed     = "Failed"
)

// CRClient writes ChartpressConfig CRs: Update for the main resource (finalizers),
// UpdateStatus for the status subresource.
type CRClient interface {
	Update(ctx context.Context, ns string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
	UpdateStatus(ctx context.Context, ns string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
}

// Reconciler is the level-based state machine for one ChartpressConfig.
type Reconciler struct {
	Client    CRClient
	Renderer  Renderer
	Uploader  objectstore.Uploader
	Namespace string
	Now       func() time.Time
}

func (r *Reconciler) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now().UTC()
}

func artifactKey(name string) string { return "charts/" + name + ".zip" }

// Reconcile drives one ChartpressConfig toward its spec: ensure the cleanup
// finalizer, run finalizer cleanup on delete, short-circuit when the current
// generation is already Ready, else render → upload → write Ready status.
// Returning a non-nil error requeues the key with rate-limited backoff.
func (r *Reconciler) Reconcile(ctx context.Context, obj *unstructured.Unstructured) error {
	name := obj.GetName()
	ns := obj.GetNamespace()
	if ns == "" {
		ns = r.Namespace
	}

	// Deletion: remove the artifact, then drop the finalizer.
	if obj.GetDeletionTimestamp() != nil {
		if !hasFinalizer(obj, apis.FinalizerArtifactCleanup) {
			return nil
		}
		if err := r.Uploader.Remove(ctx, artifactKey(name)); err != nil {
			return fmt.Errorf("remove artifact for %q: %w", name, err)
		}
		removeFinalizer(obj, apis.FinalizerArtifactCleanup)
		if _, err := r.Client.Update(ctx, ns, obj); err != nil {
			return fmt.Errorf("drop finalizer for %q: %w", name, err)
		}
		return nil
	}

	// Ensure the cleanup finalizer before generating any artifact.
	if !hasFinalizer(obj, apis.FinalizerArtifactCleanup) {
		addFinalizer(obj, apis.FinalizerArtifactCleanup)
		updated, err := r.Client.Update(ctx, ns, obj)
		if err != nil {
			return fmt.Errorf("add finalizer for %q: %w", name, err)
		}
		obj = updated
	}

	// Short-circuit: this generation already succeeded (observedGeneration is
	// stamped only on success, so phase==Ready && observed==generation is exact).
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	observed, _, _ := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	if phase == phaseReady && observed == obj.GetGeneration() {
		return nil
	}

	// Mark Generating.
	obj, err := r.writeStatus(ctx, ns, obj, map[string]interface{}{"phase": phaseGenerating})
	if err != nil {
		return fmt.Errorf("set Generating for %q: %w", name, err)
	}

	// Decode + validate (defense-in-depth; CRD already enforces structure).
	spec, derr := decodeSpec(obj)
	if derr == nil {
		derr = engine.Validate(spec)
	}
	if derr != nil {
		_, _ = r.writeStatus(ctx, ns, obj, failedStatus("invalid spec: "+derr.Error()))
		return nil // deterministic; resync re-checks if the spec changes
	}

	// Render.
	zipBytes, rerr := r.Renderer.RenderZip(spec)
	if rerr != nil {
		_, _ = r.writeStatus(ctx, ns, obj, failedStatus("render failed: "+rerr.Error()))
		return rerr
	}

	// Upload (overwrite charts/<name>.zip).
	key := artifactKey(name)
	if uerr := r.Uploader.Upload(ctx, key, bytes.NewReader(zipBytes), int64(len(zipBytes))); uerr != nil {
		_, _ = r.writeStatus(ctx, ns, obj, failedStatus("upload failed: "+uerr.Error()))
		return uerr
	}

	// Ready (stamp observedGeneration only here).
	_, err = r.writeStatus(ctx, ns, obj, map[string]interface{}{
		"phase":              phaseReady,
		"observedGeneration": obj.GetGeneration(),
		"artifactKey":        key,
		"lastGenerated":      r.now().Format(time.RFC3339),
		"message":            "",
	})
	if err != nil {
		return fmt.Errorf("set Ready for %q: %w", name, err)
	}
	return nil
}

// writeStatus replaces the managed status block and persists it via UpdateStatus,
// returning the server's copy (fresh resourceVersion) for the next write.
func (r *Reconciler) writeStatus(ctx context.Context, ns string, obj *unstructured.Unstructured, status map[string]interface{}) (*unstructured.Unstructured, error) {
	if err := unstructured.SetNestedMap(obj.Object, status, "status"); err != nil {
		return nil, err
	}
	return r.Client.UpdateStatus(ctx, ns, obj)
}

func failedStatus(msg string) map[string]interface{} {
	return map[string]interface{}{"phase": phaseFailed, "message": msg}
}

func hasFinalizer(obj *unstructured.Unstructured, f string) bool {
	for _, x := range obj.GetFinalizers() {
		if x == f {
			return true
		}
	}
	return false
}

func addFinalizer(obj *unstructured.Unstructured, f string) {
	if hasFinalizer(obj, f) {
		return
	}
	obj.SetFinalizers(append(obj.GetFinalizers(), f))
}

func removeFinalizer(obj *unstructured.Unstructured, f string) {
	cur := obj.GetFinalizers()
	out := cur[:0]
	for _, x := range cur {
		if x != f {
			out = append(out, x)
		}
	}
	obj.SetFinalizers(out)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/operator/ -v && go build ./...`
Expected: PASS (render + decode + all reconcile tests); build clean.

- [ ] **Step 5: Commit**

```bash
git add internal/operator/reconcile.go internal/operator/reconcile_test.go
git commit -m "feat(operator): level-based reconcile state machine + finalizer"
```

---

### Task 6: Operator dynamic CR client

**Files:**
- Create: `internal/operator/client.go`
- Test: `internal/operator/client_test.go`

**Interfaces:**
- Consumes: `apis.GVR`; `dynamic.Interface`.
- Produces: `type dynamicCRClient struct{ client dynamic.Interface }` implementing `CRClient` (Update/UpdateStatus); `func newDynamicCRClient(client dynamic.Interface) *dynamicCRClient`.

- [ ] **Step 1: Write the failing test**

```go
// internal/operator/client_test.go
package operator

import (
	"context"
	"testing"

	"github.com/kriipke/chartpress/internal/apis"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

var _ CRClient = (*dynamicCRClient)(nil)

func newFakeDynamic(objs ...runtime.Object) *dynamicfake.FakeDynamicClient {
	scheme := runtime.NewScheme()
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{apis.GVR: "ChartpressConfigList"},
		objs...,
	)
}

func TestDynamicCRClientUpdateAddsFinalizer(t *testing.T) {
	obj := crObj("demo", 1)
	c := newDynamicCRClient(newFakeDynamic(obj))

	obj.SetFinalizers([]string{apis.FinalizerArtifactCleanup})
	out, err := c.Update(context.Background(), "chartpress-system", obj)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !hasFinalizer(out, apis.FinalizerArtifactCleanup) {
		t.Fatalf("finalizer not persisted: %v", out.GetFinalizers())
	}
}

func TestDynamicCRClientUpdateStatus(t *testing.T) {
	obj := crObj("demo", 1)
	c := newDynamicCRClient(newFakeDynamic(obj))

	_ = unstructured.SetNestedField(obj.Object, phaseReady, "status", "phase")
	out, err := c.UpdateStatus(context.Background(), "chartpress-system", obj)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	phase, _, _ := unstructured.NestedString(out.Object, "status", "phase")
	if phase != phaseReady {
		t.Fatalf("status.phase = %q, want Ready", phase)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/operator/ -run TestDynamicCRClient -v`
Expected: FAIL — `undefined: dynamicCRClient` / `newDynamicCRClient`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/operator/client.go
package operator

import (
	"context"

	"github.com/kriipke/chartpress/internal/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// dynamicCRClient implements CRClient over a dynamic client for apis.GVR.
type dynamicCRClient struct{ client dynamic.Interface }

func newDynamicCRClient(client dynamic.Interface) *dynamicCRClient {
	return &dynamicCRClient{client: client}
}

func (c *dynamicCRClient) Update(ctx context.Context, ns string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.client.Resource(apis.GVR).Namespace(ns).Update(ctx, obj, metav1.UpdateOptions{})
}

func (c *dynamicCRClient) UpdateStatus(ctx context.Context, ns string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.client.Resource(apis.GVR).Namespace(ns).UpdateStatus(ctx, obj, metav1.UpdateOptions{})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/operator/ -v && go build ./...`
Expected: PASS; build clean.

- [ ] **Step 5: Commit**

```bash
git add internal/operator/client.go internal/operator/client_test.go
git commit -m "feat(operator): dynamic CR client (Update/UpdateStatus)"
```

---

### Task 7: Operator controller (informer + workqueue) + `Start()` + `cmd/operator`

**Files:**
- Create: `internal/operator/controller.go`
- Create: `cmd/operator/main.go`
- Test: `internal/operator/controller_test.go`

**Interfaces:**
- Consumes: `apis.GVR`; `Reconciler` (T5); `dynamicCRClient` (T6); `chartRenderer` (T4); `objectstore.New`/`ConfigFromEnv` (T2–T3); `dynamic`, `dynamicinformer`, `workqueue`, `cache`.
- Produces:
  - `func NewController(client dynamic.Interface, rec *Reconciler, namespace string, resync time.Duration) (*Controller, error)`
  - `func (c *Controller) Run(ctx context.Context, workers int) error`
  - `func Start()` (in-cluster wiring); `func namespaceFromEnv() string`; `func templatesDir() string`.

This task is integration wiring; its unit test is a construction/enqueue smoke test (the reconcile logic is already covered in Tasks 4–6). The rest is proven by `go build ./...` and the in-cluster path.

- [ ] **Step 1: Write the failing test**

```go
// internal/operator/controller_test.go
package operator

import (
	"testing"
	"time"

	"github.com/kriipke/chartpress/internal/apis"
)

func TestNewControllerEnqueuesEvents(t *testing.T) {
	obj := crObj("demo", 1)
	dyn := newFakeDynamic(obj)
	rec := &Reconciler{
		Client:    newDynamicCRClient(dyn),
		Renderer:  fakeRenderer{zip: []byte("x")},
		Uploader:  &fakeUploader{},
		Namespace: "chartpress-system",
		Now:       fixedClock(),
	}

	c, err := NewController(dyn, rec, "chartpress-system", 30*time.Second)
	if err != nil {
		t.Fatalf("NewController: %v", err)
	}
	// The seeded object should produce an enqueueable key.
	c.enqueue(obj)
	if c.queue.Len() == 0 {
		t.Fatal("expected the seeded CR to be enqueued")
	}
	key, _ := c.queue.Get()
	if key != "chartpress-system/demo" {
		t.Fatalf("queue key = %q, want chartpress-system/demo", key)
	}
	_ = apis.GVR // keep apis imported for clarity
}

func TestNamespaceAndTemplatesDirDefaults(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "")
	if namespaceFromEnv() != "default" {
		t.Fatalf("namespace default = %q", namespaceFromEnv())
	}
	t.Setenv("POD_NAMESPACE", "chartpress-system")
	if namespaceFromEnv() != "chartpress-system" {
		t.Fatalf("namespace = %q", namespaceFromEnv())
	}
	t.Setenv("CHARTPRESS_TEMPLATES_DIR", "")
	if templatesDir() != "templates" {
		t.Fatalf("templatesDir default = %q", templatesDir())
	}
	t.Setenv("CHARTPRESS_TEMPLATES_DIR", "/app/templates")
	if templatesDir() != "/app/templates" {
		t.Fatalf("templatesDir = %q", templatesDir())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/operator/ -run 'TestNewController|TestNamespaceAndTemplatesDir' -v`
Expected: FAIL — `undefined: NewController` / `namespaceFromEnv` / `templatesDir`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/operator/controller.go
package operator

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kriipke/chartpress/internal/apis"
	"github.com/kriipke/chartpress/internal/objectstore"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller wires a dynamic SharedInformer to a rate-limiting workqueue and a
// Reconciler. No controller-runtime, no leader election, one replica; the 30s
// resync re-delivers cached objects as the level-based safety net.
type Controller struct {
	informer cache.SharedIndexInformer
	queue    workqueue.TypedRateLimitingInterface[string]
	rec      *Reconciler
}

func NewController(client dynamic.Interface, rec *Reconciler, namespace string, resync time.Duration) (*Controller, error) {
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, resync, namespace, nil)
	informer := factory.ForResource(apis.GVR).Informer()
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())
	c := &Controller{informer: informer, queue: queue, rec: rec}

	if _, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.enqueue,
		UpdateFunc: func(_, newObj interface{}) { c.enqueue(newObj) },
		DeleteFunc: c.enqueue,
	}); err != nil {
		return nil, fmt.Errorf("add event handler: %w", err)
	}
	return c, nil
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Printf("[ERROR] key for object: %v", err)
		return
	}
	c.queue.Add(key)
}

// Run starts the informer and worker loops, blocking until ctx is cancelled.
func (c *Controller) Run(ctx context.Context, workers int) error {
	defer c.queue.ShutDown()
	go c.informer.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		return fmt.Errorf("informer cache sync failed")
	}
	for i := 0; i < workers; i++ {
		go wait.Until(func() {
			for c.processNext(ctx) {
			}
		}, time.Second, ctx.Done())
	}
	<-ctx.Done()
	return nil
}

func (c *Controller) processNext(ctx context.Context) bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)

	obj, exists, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil || !exists {
		c.queue.Forget(key)
		return true
	}
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		c.queue.Forget(key)
		return true
	}
	if err := c.rec.Reconcile(ctx, u.DeepCopy()); err != nil {
		log.Printf("[ERROR] reconcile %s: %v", key, err)
		c.queue.AddRateLimited(key)
		return true
	}
	c.queue.Forget(key)
	return true
}

// Start wires the in-cluster operator and runs it (production entrypoint).
func Start() {
	log.Println("[INFO] starting chartpress operator")
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("[FATAL] in-cluster config: %v", err)
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("[FATAL] dynamic client: %v", err)
	}
	store, err := objectstore.New(objectstore.ConfigFromEnv())
	if err != nil {
		log.Fatalf("[FATAL] object store: %v", err)
	}
	ns := namespaceFromEnv()
	rec := &Reconciler{
		Client:    newDynamicCRClient(dyn),
		Renderer:  &chartRenderer{templatesDir: templatesDir()},
		Uploader:  store,
		Namespace: ns,
	}
	ctrl, err := NewController(dyn, rec, ns, 30*time.Second)
	if err != nil {
		log.Fatalf("[FATAL] controller: %v", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Printf("[INFO] operator watching namespace %q", ns)
	if err := ctrl.Run(ctx, 1); err != nil {
		log.Fatalf("[FATAL] run: %v", err)
	}
}

func namespaceFromEnv() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	return "default"
}

func templatesDir() string {
	if d := os.Getenv("CHARTPRESS_TEMPLATES_DIR"); d != "" {
		return d
	}
	return "templates"
}
```

```go
// cmd/operator/main.go
package main

import "github.com/kriipke/chartpress/internal/operator"

func main() {
	operator.Start()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/operator/ -v && go build ./...`
Expected: PASS (controller smoke + namespace/templates defaults); build clean (incl. `cmd/operator`).

- [ ] **Step 5: Commit**

```bash
git add internal/operator/controller.go internal/operator/controller_test.go cmd/operator/main.go
git commit -m "feat(operator): informer+workqueue controller, Start(), cmd/operator"
```

---

### Task 8: Backend presign wiring (`downloadUrl` from `status.artifactKey`)

**Files:**
- Modify: `internal/server/k8s.go` (add `Presigner` interface)
- Modify: `internal/server/server.go` (add `Presigner` field; wire `objectstore.New` in `Start()`)
- Modify: `internal/server/charts.go` (`summarize` mints `downloadUrl`)
- Test: `internal/server/charts_test.go` (extend)

**Interfaces:**
- Consumes: `objectstore.New`/`ConfigFromEnv` (Start only); `ChartLister` (existing).
- Produces:
  - `type Presigner interface { PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) }` (consumer-side, alongside `Applier`/`ChartLister`/`Drafter`).
  - `Server.Presigner Presigner` field.
  - `func summarize(ctx context.Context, p Presigner, obj unstructured.Unstructured) chartSummary` (new signature).

- [ ] **Step 1: Write the failing test**

```go
// internal/server/charts_test.go  (add to the existing file)
import (
	"context"
	"time"
)

type fakePresigner struct {
	url       string
	err       error
	lastKey   string
	callCount int
}

func (p *fakePresigner) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	p.callCount++
	p.lastKey = key
	return p.url, p.err
}

func crReady(name, artifactKey string) unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apiVersionV1alpha1,
		"kind":       kindChartpressConfig,
		"metadata":   map[string]interface{}{"name": name},
		"spec": map[string]interface{}{
			"umbrellaChartName": name,
			"subcharts":         []interface{}{map[string]interface{}{"name": "api", "workload": "deployment"}},
		},
		"status": map[string]interface{}{"phase": "Ready", "artifactKey": artifactKey},
	}}
}

func TestChartsMintsDownloadURLWhenReady(t *testing.T) {
	pre := &fakePresigner{url: "https://s3.example.com/charts/demo.zip?sig=abc"}
	srv := &Server{
		Lister:    &fakeLister{items: []unstructured.Unstructured{crReady("demo", "charts/demo.zip")}},
		Presigner: pre,
		Namespace: "ns",
	}
	req := httptest.NewRequest(http.MethodGet, "/charts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var got []chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].DownloadURL != "https://s3.example.com/charts/demo.zip?sig=abc" {
		t.Fatalf("downloadUrl = %q (got %+v)", got[0].DownloadURL, got)
	}
	if pre.lastKey != "charts/demo.zip" {
		t.Fatalf("presigned key = %q, want charts/demo.zip", pre.lastKey)
	}
}

func TestChartsNoDownloadURLWhenNotReady(t *testing.T) {
	pre := &fakePresigner{url: "https://should-not-be-used"}
	srv := &Server{
		Lister: &fakeLister{items: []unstructured.Unstructured{
			crWith("pending-one", "Generating", "", 1, ""), // existing helper; no artifactKey
		}},
		Presigner: pre,
		Namespace: "ns",
	}
	req := httptest.NewRequest(http.MethodGet, "/charts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var got []chartSummary
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got) != 1 || got[0].DownloadURL != "" {
		t.Fatalf("downloadUrl must be empty when not Ready; got %+v", got)
	}
	if pre.callCount != 0 {
		t.Fatalf("presigner must not be called for non-Ready charts; calls=%d", pre.callCount)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/ -run 'TestChartsMints|TestChartsNoDownload' -v`
Expected: FAIL — `Server` has no `Presigner` field / `summarize` signature mismatch.

- [ ] **Step 3: Write minimal implementation**

Add the interface to `internal/server/k8s.go` (next to `Drafter`):

```go
import "time" // add to the import block

// Presigner mints a presigned GET URL for a stored chart archive (S3/R2/MinIO).
// Implemented by *objectstore.Client; injected so handlers test against a fake.
type Presigner interface {
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
}
```

Add the field to `Server` in `internal/server/server.go`:

```go
type Server struct {
	Applier   Applier
	Lister    ChartLister
	Drafter   Drafter
	Presigner Presigner
	Namespace string
}
```

Wire it in `Start()` (after building the dynamic client; before constructing `srv`, or set on `srv` after). S3 may be unconfigured in dev — degrade gracefully to no download URLs:

```go
	srv := &Server{
		Applier:   &dynamicApplier{client: client},
		Lister:    &dynamicLister{client: client},
		Namespace: resolveNamespace(),
	}
	if store, err := objectstore.New(objectstore.ConfigFromEnv()); err != nil {
		log.Printf("[WARN] object storage not configured, downloads disabled: %v", err)
	} else {
		srv.Presigner = store
	}
```

(Add `"github.com/kriipke/chartpress/internal/objectstore"` to `server.go` imports.)

Update `internal/server/charts.go` — new `summarize` signature + presign, and pass through from the handlers:

```go
import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const presignExpiry = 15 * time.Minute

// summarize maps a ChartpressConfig CR to the Charts-row shape. When the operator
// has marked it Ready and recorded an artifactKey, it mints a fresh presigned GET
// URL; any presign error is logged and leaves downloadUrl empty (the row still
// renders). downloadUrl stays empty for every non-Ready phase.
func summarize(ctx context.Context, p Presigner, obj unstructured.Unstructured) chartSummary {
	subs, _, _ := unstructured.NestedSlice(obj.Object, "spec", "subcharts")
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	if phase == "" {
		phase = "Pending"
	}
	msg, _, _ := unstructured.NestedString(obj.Object, "status", "message")
	lastGen, _, _ := unstructured.NestedString(obj.Object, "status", "lastGenerated")
	cs := chartSummary{
		Name:          obj.GetName(),
		Phase:         phase,
		SubchartCount: len(subs),
		LastGenerated: lastGen,
		Message:       msg,
	}
	if phase == "Ready" && p != nil {
		if key, _, _ := unstructured.NestedString(obj.Object, "status", "artifactKey"); key != "" {
			if url, err := p.PresignGet(ctx, key, presignExpiry); err != nil {
				log.Printf("[ERROR] presign %q: %v", key, err)
			} else {
				cs.DownloadURL = url
			}
		}
	}
	return cs
}
```

Update the two handlers in `charts.go` to pass `r.Context(), s.Presigner`:

```go
	// in handleCharts:
	for _, it := range items {
		out = append(out, summarize(r.Context(), s.Presigner, it))
	}
	// ...
	// in handleChartByName:
	_ = json.NewEncoder(w).Encode(summarize(r.Context(), s.Presigner, *obj))
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/server/ -v && go build ./...`
Expected: PASS — new presign tests AND existing `TestChartsListMapsFields`/`TestChartByName*` still green (they construct `Server` without `Presigner`, so `downloadUrl` stays empty as before).

- [ ] **Step 5: Commit**

```bash
git add internal/server/k8s.go internal/server/server.go internal/server/charts.go internal/server/charts_test.go go.mod go.sum
git commit -m "feat(server): mint presigned downloadUrl from status.artifactKey when Ready"
```

---

### Task 9: Helm chart — operator Deployment + RBAC + S3 config; backend S3 env

**Files:**
- Create: `chart/templates/operator-deployment.yaml`
- Create: `chart/templates/operator-rbac.yaml`
- Create: `chart/templates/s3-secret.yaml`
- Modify: `chart/templates/_helpers.tpl` (add `chartpress.s3env`)
- Modify: `chart/templates/backend-deployment.yaml` (add S3 env)
- Modify: `chart/values.yaml` (add `operator:` + `s3:`)
- Test: `internal/deploy/operator_test.go`

**Interfaces:**
- Consumes: the existing `renderChart(t)` harness in `internal/deploy/chart_test.go` (renders `chart/` with default values, release `chartpress`, ns `chartpress-system`).

- [ ] **Step 1: Write the failing test**

```go
// internal/deploy/operator_test.go
package deploy

import (
	"strings"
	"testing"
)

func TestOperatorDeploymentRendered(t *testing.T) {
	man := renderChart(t)
	for _, want := range []string{
		"name: chartpress-operator",
		"command: [\"/app/operator\"]",
		"image: \"ghcr.io/kriipke/chartpress/api:",
		"serviceAccountName: chartpress-operator",
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("operator deployment missing %q", want)
		}
	}
}

func TestOperatorRBACGrantsStatusAndFinalizers(t *testing.T) {
	man := renderChart(t)
	for _, want := range []string{
		"name: chartpress-operator",
		"chartpressconfigs/status",
		"chartpressconfigs/finalizers",
		`verbs: ["get", "list", "watch", "update", "patch"]`,
	} {
		if !strings.Contains(man, want) {
			t.Fatalf("operator RBAC missing %q", want)
		}
	}
}

func TestS3EnvOnOperatorAndBackend(t *testing.T) {
	man := renderChart(t)
	for _, want := range []string{
		"name: S3_ENDPOINT",
		"name: S3_BUCKET",
		"name: S3_ACCESS_KEY",
		"name: S3_SECRET_KEY",
	} {
		if strings.Count(man, want) < 2 {
			t.Fatalf("expected %q on BOTH operator and backend (>=2 occurrences)", want)
		}
	}
	// The S3 Secret is rendered by default (s3.create defaults true).
	if !strings.Contains(man, "name: chartpress-s3") {
		t.Fatal("chartpress-s3 Secret not rendered")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/deploy/ -run 'TestOperator|TestS3Env' -v`
Expected: FAIL — operator/S3 strings absent from the rendered chart.

- [ ] **Step 3: Write minimal implementation**

Append the S3 env partial to `chart/templates/_helpers.tpl`:

```gotemplate
{{- define "chartpress.s3env" -}}
- name: S3_ENDPOINT
  value: {{ .Values.s3.endpoint | quote }}
- name: S3_BUCKET
  value: {{ .Values.s3.bucket | quote }}
- name: S3_REGION
  value: {{ .Values.s3.region | quote }}
- name: S3_USE_SSL
  value: {{ .Values.s3.useSSL | quote }}
- name: S3_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      name: {{ .Values.s3.existingSecret | default "chartpress-s3" }}
      key: {{ .Values.s3.accessKeyKey }}
- name: S3_SECRET_KEY
  valueFrom:
    secretKeyRef:
      name: {{ .Values.s3.existingSecret | default "chartpress-s3" }}
      key: {{ .Values.s3.secretKeyKey }}
{{- end -}}
```

Create `chart/templates/operator-deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chartpress-operator
  labels:
    app: chartpress-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: chartpress-operator
  template:
    metadata:
      labels:
        app: chartpress-operator
    spec:
      serviceAccountName: {{ .Values.operator.serviceAccount.name }}
      containers:
        - name: operator
          image: "{{ .Values.backend.image.repository }}:{{ .Values.backend.image.tag }}"
          command: ["/app/operator"]
          env:
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            {{- include "chartpress.s3env" . | nindent 12 }}
          # /app/templates is baked into the image; the operator reads it via
          # CHARTPRESS_TEMPLATES_DIR (default "templates", run from WORKDIR /app).
```

Create `chart/templates/operator-rbac.yaml`:

```yaml
{{- if .Values.operator.rbac.create }}
# The operator watches ChartpressConfig CRs and writes their status + finalizers.
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ .Values.operator.serviceAccount.name }}
  namespace: {{ .Release.Namespace }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: chartpress-operator
  namespace: {{ .Release.Namespace }}
rules:
  - apiGroups: ["chartpress.dev"]
    resources: ["chartpressconfigs"]
    verbs: ["get", "list", "watch", "update", "patch"]
  - apiGroups: ["chartpress.dev"]
    resources: ["chartpressconfigs/status"]
    verbs: ["get", "update", "patch"]
  - apiGroups: ["chartpress.dev"]
    resources: ["chartpressconfigs/finalizers"]
    verbs: ["update"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: chartpress-operator
  namespace: {{ .Release.Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: chartpress-operator
subjects:
  - kind: ServiceAccount
    name: {{ .Values.operator.serviceAccount.name }}
    namespace: {{ .Release.Namespace }}
{{- end }}
```

Create `chart/templates/s3-secret.yaml`:

```yaml
{{- if and .Values.s3.create (not .Values.s3.existingSecret) }}
# BYO bucket credentials. Operator uses them to upload; backend to presign GETs.
# Set s3.existingSecret to reference a pre-existing Secret instead.
apiVersion: v1
kind: Secret
metadata:
  name: chartpress-s3
  namespace: {{ .Release.Namespace }}
type: Opaque
stringData:
  {{ .Values.s3.accessKeyKey }}: {{ .Values.s3.accessKey | quote }}
  {{ .Values.s3.secretKeyKey }}: {{ .Values.s3.secretKey | quote }}
{{- end }}
```

Modify `chart/templates/backend-deployment.yaml` — add the S3 env after the OpenAI block (inside the container's `env:`, before `volumeMounts:`):

```yaml
            {{- end }}
            {{- include "chartpress.s3env" . | nindent 12 }}
          volumeMounts:
```

(The `{{- end }}` shown is the existing close of the `if .Values.backend.openai.apiKeySecret.name` block; insert the `include` line immediately after it.)

Add to `chart/values.yaml`:

```yaml
operator:
  serviceAccount:
    name: chartpress-operator
  rbac:
    create: true

s3:
  # Render the chartpress-s3 Secret from accessKey/secretKey below (BYO bucket).
  # Set existingSecret to reference a pre-existing Secret instead (overrides create).
  create: true
  existingSecret: ""
  endpoint: s3.amazonaws.com
  bucket: chartpress-charts
  region: us-east-1
  useSSL: true
  accessKeyKey: access-key
  secretKeyKey: secret-key
  accessKey: ""
  secretKey: ""
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/deploy/ -v`
Expected: PASS — new operator/S3 tests AND the existing backend RBAC / downward-API / nginx tests still green.

- [ ] **Step 5: Commit**

```bash
git add chart/templates/operator-deployment.yaml chart/templates/operator-rbac.yaml chart/templates/s3-secret.yaml chart/templates/_helpers.tpl chart/templates/backend-deployment.yaml chart/values.yaml internal/deploy/operator_test.go
git commit -m "feat(chart): operator Deployment + RBAC, S3 Secret/values, backend S3 env"
```

---

### Task 10: Packaging — Dockerfile operator binary + whole-tree green

**Files:**
- Modify: `Dockerfile`

This task has no new unit test; it is validated by building the operator binary, the whole Go tree, the full test suite, and `helm lint`/`helm template`.

- [ ] **Step 1: Verify the gap (operator binary not in the image)**

Run: `grep -n 'cmd/operator\|/app/operator' Dockerfile`
Expected: no matches — the image does not yet build or copy the operator.

- [ ] **Step 2: Modify the Dockerfile**

Add the operator build (after the `chartpress-server` build) and include it in the runtime copy:

```dockerfile
RUN CGO_ENABLED=0 go build -o chartpress ./cmd/chartpress
RUN CGO_ENABLED=0 go build -o chartpress-server ./cmd/server
RUN CGO_ENABLED=0 go build -o operator ./cmd/operator
```

```dockerfile
# Copy the built binaries from the builder stage
COPY --from=builder /app/chartpress /app/chartpress-server /app/operator ./
```

(The runtime `command: ["/app/operator"]` from the operator Deployment runs this binary; `chartpress-server` stays the default `CMD` for the backend. `templates/` is already copied for the operator's renderer.)

- [ ] **Step 3: Build the operator binary + whole tree**

Run:
```bash
go build -o /tmp/chartpress-operator ./cmd/operator && echo "operator builds"
go build ./...
go mod tidy
```
Expected: operator builds; whole tree builds; `go mod tidy` is a no-op (or only tidies — no new direct deps beyond minio-go).

- [ ] **Step 4: Run the full suite + chart lint**

Run:
```bash
go test ./...
helm lint chart/
helm template chartpress chart/ --namespace chartpress-system >/dev/null && echo "helm template ok"
```
Expected: all Go packages PASS (`internal/apis`, `internal/objectstore`, `internal/operator`, `internal/server`, `internal/engine`, `internal/crd`, `internal/deploy`); `helm lint` reports 0 failures; `helm template` renders without error.

- [ ] **Step 5: Commit**

```bash
git add Dockerfile go.mod go.sum
git commit -m "build(docker): build and ship the operator binary in the shared image"
```

---

## Self-Review

**1. Spec coverage (design §6, §8, §10 operator + S3 parts; §5 backend `/charts` downloadUrl):**
- §6 operator binary (same image, `command:[operator]`) → Tasks 7, 9, 10. client-go informer/watch, one replica, no leader election → Task 7. Level-based reconcile (observedGeneration≠generation → Generating → render → zip → upload → Ready) → Task 5. Finalizer `chartpress.dev/artifact-cleanup` add + delete cleanup → Task 5. Status writes via subresource → Tasks 5/6. Templates baked, read by operator → Tasks 4/7/9/10.
- §8 minio-go behind interfaces; operator upload / backend presign; config from Secret/values; key `charts/<name>.zip` overwrite; no bundled MinIO → Tasks 2, 3 (objectstore), 5 (key), 9 (Secret/values).
- §5/§10 backend `downloadUrl` from `status.artifactKey` only when Ready (deferred Phase-2 item) → Task 8; backend S3 read creds → Task 9.
- §10 operator Deployment + SA + Role/RoleBinding (get/list/watch + update/patch on chartpressconfigs AND /status, + finalizer update) → Task 9. Backend gets S3 creds → Task 9.
- §12 testing: reconcile state machine (fake client + fake uploader) → Task 5; finalizer/delete path → Task 5; backend presign→downloadUrl (fake presigner) → Task 8.

**2. Out of scope (deferred, per the prompt §14):** frontend wizard + Charts browser (Phase 4); leader election / multi-replica; artifact history/versioning; bundled MinIO; full `status.conditions[]`. None are introduced here.

**3. Placeholder scan:** no TBD/TODO; every code step shows complete code; every command shows expected output. ✅

**4. Type consistency:**
- `apis.GVR` / `apis.GroupVersion` / `apis.Kind` / `apis.FinalizerArtifactCleanup` / `apis.FieldManager*` used identically in Tasks 1, 5, 6, 7.
- `objectstore.Uploader` (`Upload`+`Remove`) / `objectstore.Presigner` (`PresignGet`) defined in Task 3, consumed by `Reconciler.Uploader` (Task 5) and the server `Presigner` interface shape (Task 8).
- `Renderer.RenderZip(engine.Spec) ([]byte, error)` defined Task 4, implemented by `chartRenderer`, faked in Task 5, wired in Task 7.
- `CRClient` (`Update`/`UpdateStatus`) defined Task 5, implemented by `dynamicCRClient` Task 6, wired Task 7.
- phase consts `phaseGenerating/phaseReady/phaseFailed` consistent across Tasks 5–7; the server uses the literal `"Ready"` (Task 8) matching what the operator writes.
- `summarize(ctx, p, obj)` new signature (Task 8) updated at both call sites in `charts.go`.
- `crObj` / `fakeRenderer` / `fakeUploader` / `fixedClock` / `newFakeDynamic` shared helpers live in `package operator` test files (Tasks 4–7).

**Known execution risks (validate via the tests, adjust inline):**
1. **dynamic/fake `UpdateStatus`** (Task 6): the fake updates the object's status generically; the test asserts on the returned object to avoid subresource-tracker variance across client-go patch releases.
2. **Typed workqueue API** (Task 7): `workqueue.NewTypedRateLimitingQueue[string]` + `DefaultTypedControllerRateLimiter[string]` are the v0.32 idiom; if a symbol differs, fall back to the non-generic `workqueue.NewRateLimitingQueue` with `interface{}` keys (logic unchanged).
3. **minio path-style presign host** (Task 3): the assertion expects path-style for a non-AWS endpoint; switch to virtual-host if a future minio default changes (the `X-Amz-*` assertions are stable).
4. **Helm `nindent` column** (Task 9): the `include "chartpress.s3env" . | nindent 12` indentation must match the container `env:` list column; the deploy render test catches a miscount.

## Notes for Phase 4 (not implemented here)

- The frontend Charts browser consumes `GET /charts` → `downloadUrl` (now populated for `Ready` charts) and the choose/prompt/rich-form wizard. With the operator in place the pipeline is end-to-end testable in-cluster; the UI that surfaces it is Phase 4.
