package operator

import (
	"encoding/json"
	"fmt"

	"github.com/kriipke/chartpress/internal/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// decodeSpec extracts the CR's .spec and JSON round-trips it into a normalized
// engine.Spec. The backend persists a complete, defaulted rules block, so this is
// a re-hydration; Normalize is applied defensively. Validation is the reconciler's.
func decodeSpec(obj *unstructured.Unstructured) (engine.Spec, error) {
	raw, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return engine.Spec{}, fmt.Errorf("read .spec: %w", err)
	}
	if !found {
		return engine.Spec{}, fmt.Errorf("ChartpressConfig %q has no .spec", obj.GetName())
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return engine.Spec{}, err
	}
	var spec engine.Spec
	if err := json.Unmarshal(b, &spec); err != nil {
		return engine.Spec{}, err
	}
	return engine.Normalize(spec), nil
}
