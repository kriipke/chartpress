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
