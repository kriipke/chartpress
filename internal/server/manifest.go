// internal/server/manifest.go
package server

import (
	"encoding/json"

	"github.com/kriipke/chartpress/internal/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	sigsyaml "sigs.k8s.io/yaml"
)

const (
	apiGroup             = "chartpress.dev"
	apiVersionV1alpha1   = "chartpress.dev/v1alpha1"
	kindChartpressConfig = "ChartpressConfig"
	fieldManager         = "chartpress-backend"
)

var chartpressGVR = schema.GroupVersionResource{
	Group:    apiGroup,
	Version:  "v1alpha1",
	Resource: "chartpressconfigs",
}

// wrapManifest builds an unstructured ChartpressConfig CR from a (already
// normalized) spec. metadata.name == spec.UmbrellaChartName; .spec is the spec
// itself, JSON round-tripped so the persisted CR carries the engine field names.
func wrapManifest(spec engine.Spec) *unstructured.Unstructured {
	b, _ := json.Marshal(spec) // engine.Spec has no unmarshalable fields
	var specMap map[string]interface{}
	_ = json.Unmarshal(b, &specMap)

	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apiVersionV1alpha1,
		"kind":       kindChartpressConfig,
		"metadata": map[string]interface{}{
			"name": spec.UmbrellaChartName,
		},
		"spec": specMap,
	}}
}

// manifestYAML serializes the CR to YAML for the /generate response body.
func manifestYAML(obj *unstructured.Unstructured) (string, error) {
	b, err := sigsyaml.Marshal(obj.Object)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
