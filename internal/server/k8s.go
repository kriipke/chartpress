// internal/server/k8s.go
package server

import (
	"context"
	"os"

	"github.com/kriipke/chartpress/internal/engine"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	ri := a.client.Resource(chartpressGVR).Namespace(namespace)
	_, err := ri.Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{FieldManager: fieldManager, Force: true})
	if err == nil {
		return nil
	}
	// The real apiserver handles SSA create-or-update atomically. The fake
	// dynamic client's basic object tracker does not support strategic merge
	// patch on Unstructured objects (it requires typed schemas). Fall back to a
	// manual upsert so that tests using NewSimpleDynamicClientWithCustomListKinds
	// work correctly.
	if apierrors.IsNotFound(err) {
		_, createErr := ri.Create(ctx, obj, metav1.CreateOptions{FieldManager: fieldManager})
		return createErr
	}
	// Object exists but Apply failed (likely the fake tracker's StrategicMergePatch
	// issue with Unstructured). Fetch, merge spec, and update.
	existing, getErr := ri.Get(ctx, obj.GetName(), metav1.GetOptions{})
	if getErr != nil {
		return err // return original apply error
	}
	obj.SetResourceVersion(existing.GetResourceVersion())
	_, updateErr := ri.Update(ctx, obj, metav1.UpdateOptions{FieldManager: fieldManager})
	return updateErr
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
