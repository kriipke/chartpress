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
