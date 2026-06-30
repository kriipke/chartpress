// internal/server/generate.go
package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/kriipke/chartpress/internal/engine"
)

type generateResponse struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Phase        string `json:"phase"`
	ManifestYAML string `json:"manifestYaml"`
}

// handleGenerate validates a Spec, wraps it into a ChartpressConfig CR,
// server-side-applies it, and returns immediately with phase Pending. Rendering
// happens asynchronously in the operator (Phase 3).
func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST is allowed", http.StatusMethodNotAllowed)
		return
	}

	spec, err := decodeSpec(r.Body)
	if err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := engine.Validate(spec); err != nil {
		http.Error(w, "invalid spec: "+err.Error(), http.StatusBadRequest)
		return
	}

	obj := wrapManifest(spec)
	if err := s.Applier.Apply(r.Context(), s.Namespace, obj); err != nil {
		log.Printf("[ERROR] apply ChartpressConfig %q: %v", spec.UmbrellaChartName, err)
		http.Error(w, "failed to apply ChartpressConfig: "+err.Error(), http.StatusInternalServerError)
		return
	}

	y, err := manifestYAML(obj)
	if err != nil {
		http.Error(w, "failed to serialize manifest: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(generateResponse{
		Name:         spec.UmbrellaChartName,
		Namespace:    s.Namespace,
		Phase:        "Pending",
		ManifestYAML: y,
	})
}
