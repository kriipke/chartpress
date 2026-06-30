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
