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

// Apply server-side-applies the CR. Against a real apiserver this is an atomic
// create-or-update owned by the chartpress-backend field manager; Force resolves
// field-ownership conflicts in the backend's favor. Errors are surfaced as-is —
// we never fall back to a non-SSA write that could mask an apiserver rejection.
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
