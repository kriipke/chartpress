// internal/server/texttoconfig.go
package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kriipke/chartpress/internal/engine"
)

// handleTextToConfig drafts a Spec from a natural-language prompt via the
// Drafter (OpenAI in production). It returns the normalized spec for the form to
// pre-fill; the frontend overrides the name and the user edits before /generate
// validates. This endpoint does not hard-validate (the draft may need editing).
func (s *Server) handleTextToConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST is allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Prompt) == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}
	spec, err := s.Drafter.Draft(r.Context(), body.Prompt)
	if err != nil {
		http.Error(w, "failed to draft spec: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(engine.Normalize(spec))
}
