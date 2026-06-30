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
