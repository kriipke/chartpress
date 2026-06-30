// internal/server/k8s_test.go
package server

import (
	"context"
	"strings"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"
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

// The dynamic fake's basic object tracker cannot round-trip a server-side apply
// of an Unstructured object, so we capture the apply action via a reactor and
// assert the applier issues a correctly-targeted SSA patch (ApplyPatchType) —
// the behavioral contract dynamicApplier owns. Higher-level upsert behavior is
// exercised against the handler with a fake Applier in generate_test.go.
func TestDynamicApplierIssuesServerSideApply(t *testing.T) {
	fc := newFakeDynamic()

	var (
		gotType     types.PatchType
		gotName     string
		gotNS       string
		gotResource string
		gotBody     []byte
	)
	fc.PrependReactor("patch", "chartpressconfigs", func(action clienttesting.Action) (bool, runtime.Object, error) {
		pa := action.(clienttesting.PatchAction)
		gotType = pa.GetPatchType()
		gotName = pa.GetName()
		gotNS = pa.GetNamespace()
		gotResource = pa.GetResource().Resource
		gotBody = pa.GetPatch()
		return true, wrapManifest(specFor("demo")), nil
	})

	a := &dynamicApplier{client: fc}
	if err := a.Apply(context.Background(), "team-a", wrapManifest(specFor("demo"))); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if gotType != types.ApplyPatchType {
		t.Fatalf("patch type = %q, want server-side apply (%q)", gotType, types.ApplyPatchType)
	}
	if gotResource != "chartpressconfigs" || gotNS != "team-a" || gotName != "demo" {
		t.Fatalf("apply targeted resource=%q ns=%q name=%q, want chartpressconfigs/team-a/demo", gotResource, gotNS, gotName)
	}
	if !strings.Contains(string(gotBody), "umbrellaChartName") {
		t.Fatalf("apply body missing the spec: %s", gotBody)
	}
}

// Apply must surface apiserver errors rather than silently retrying with a
// non-SSA write (the production code has no Get+Update fallback).
func TestDynamicApplierSurfacesApplyError(t *testing.T) {
	fc := newFakeDynamic()
	fc.PrependReactor("patch", "chartpressconfigs", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInvalid(
			schema.GroupKind{Group: apiGroup, Kind: kindChartpressConfig}, "demo", nil)
	})

	a := &dynamicApplier{client: fc}
	if err := a.Apply(context.Background(), "team-a", wrapManifest(specFor("demo"))); err == nil {
		t.Fatal("expected apply error to be surfaced, got nil")
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
