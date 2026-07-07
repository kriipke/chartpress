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
	deleted   []string
	updateErr error
	statusErr error
	deleteErr error
}

func (f *fakeCRClient) Update(_ context.Context, _ string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	f.obj = obj.DeepCopy()
	return f.obj.DeepCopy(), nil
}

func (f *fakeCRClient) Delete(_ context.Context, _ string, name string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, name)
	return nil
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

func TestReconcileReapsExpiredAnonChart(t *testing.T) {
	fc := &fakeCRClient{}
	up := &fakeUploader{}
	r := &Reconciler{Client: fc, Renderer: fakeRenderer{zip: []byte("x")}, Uploader: up, Namespace: "chartpress-system", Now: fixedClock()}

	// Anonymous chart that expired an hour before the fixed clock.
	obj := crObj("abc123-demo", 1)
	obj.SetFinalizers([]string{apis.FinalizerArtifactCleanup})
	obj.SetAnnotations(map[string]string{
		apis.AnnotationExpiresAt: fixedClock()().Add(-time.Hour).Format(time.RFC3339),
	})
	// Already Ready — reaping must win over the Ready short-circuit.
	_ = unstructured.SetNestedField(obj.Object, phaseReady, "status", "phase")
	_ = unstructured.SetNestedField(obj.Object, int64(1), "status", "observedGeneration")

	if err := r.Reconcile(context.Background(), obj); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(fc.deleted) != 1 || fc.deleted[0] != "abc123-demo" {
		t.Fatalf("deleted = %v, want [abc123-demo]", fc.deleted)
	}
	if len(up.uploaded) != 0 {
		t.Fatalf("expired chart must not be rendered; uploads=%v", up.uploaded)
	}
}

func TestReconcileKeepsUnexpiredAnonChart(t *testing.T) {
	fc := &fakeCRClient{}
	up := &fakeUploader{}
	r := &Reconciler{Client: fc, Renderer: fakeRenderer{zip: []byte("PK")}, Uploader: up, Namespace: "chartpress-system", Now: fixedClock()}

	obj := crObj("abc123-demo", 1)
	obj.SetAnnotations(map[string]string{
		apis.AnnotationExpiresAt: fixedClock()().Add(time.Hour).Format(time.RFC3339),
	})
	if err := r.Reconcile(context.Background(), obj); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(fc.deleted) != 0 {
		t.Fatalf("unexpired chart must not be deleted; deleted=%v", fc.deleted)
	}
	if _, ok := up.uploaded["charts/abc123-demo.zip"]; !ok {
		t.Fatalf("unexpired chart should render; uploads=%v", up.uploaded)
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
