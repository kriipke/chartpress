package operator

import (
	"context"
	"strings"
	"testing"

	"github.com/kriipke/chartpress/internal/apis"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestReadyMessage(t *testing.T) {
	if got := readyMessage(nil); got != "" {
		t.Fatalf("clean spec message = %q, want empty", got)
	}
	got := readyMessage([]string{"a", "b"})
	if !strings.HasPrefix(got, "warnings: ") || !strings.Contains(got, "a; b") {
		t.Fatalf("readyMessage = %q", got)
	}
}

// A spec whose resolved subcharts trip a warning (edge gateway alongside a
// second public subchart) surfaces it in the Ready status message.
func TestReconcileSurfacesWarnings(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apis.GroupVersion,
		"kind":       apis.Kind,
		"metadata": map[string]interface{}{
			"name": "warn", "namespace": "chartpress-system", "generation": int64(1),
		},
		"spec": map[string]interface{}{
			"umbrellaChartName": "warn",
			"subcharts": []interface{}{
				map[string]interface{}{"name": "bff", "pattern": "edge-gateway"},
				map[string]interface{}{"name": "orders", "pattern": "api-microservice"},
			},
			"rules": map[string]interface{}{"ingress": "alb"},
		},
	}}

	fc := &fakeCRClient{}
	r := &Reconciler{Client: fc, Renderer: fakeRenderer{zip: []byte("PK")}, Uploader: &fakeUploader{}, Namespace: "chartpress-system", Now: fixedClock()}
	if err := r.Reconcile(context.Background(), obj); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	msg, _, _ := unstructured.NestedString(fc.obj.Object, "status", "message")
	if !strings.Contains(msg, "only public entry") {
		t.Fatalf("Ready message should carry the edge-gateway warning, got %q", msg)
	}
}
