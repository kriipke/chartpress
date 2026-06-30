// internal/server/manifest.go
package server

import (
	"encoding/json"
	"fmt"

	"github.com/kriipke/chartpress/internal/apis"
	"github.com/kriipke/chartpress/internal/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	sigsyaml "sigs.k8s.io/yaml"
)

const (
	apiGroup             = apis.Group
	apiVersionV1alpha1   = apis.GroupVersion
	kindChartpressConfig = apis.Kind
	fieldManager         = apis.FieldManagerBackend
)

var chartpressGVR = apis.GVR

// wrapManifest builds an unstructured ChartpressConfig CR from a (already
// normalized) spec. metadata.name == spec.UmbrellaChartName; .spec is the spec
// itself, JSON round-tripped so the persisted CR carries the engine field names.
func wrapManifest(spec engine.Spec) *unstructured.Unstructured {
	// engine.Spec contains only JSON-native types, so these cannot fail in
	// practice; a failure means a non-serializable field was added — surface it
	// loudly rather than persisting a CR with a null spec.
	b, err := json.Marshal(spec)
	if err != nil {
		panic(fmt.Sprintf("wrapManifest: marshal engine.Spec: %v", err))
	}
	var specMap map[string]interface{}
	if err := json.Unmarshal(b, &specMap); err != nil {
		panic(fmt.Sprintf("wrapManifest: unmarshal spec to map: %v", err))
	}

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
