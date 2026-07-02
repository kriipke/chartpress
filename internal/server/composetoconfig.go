// internal/server/composetoconfig.go
package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kriipke/chartpress/internal/compose"
	"github.com/kriipke/chartpress/internal/engine"
)

// handleComposeToConfig drafts a Spec from a docker-compose file by deterministic
// mapping (no LLM — compose is authoritative for structure). It mirrors
// handleTextToConfig, but returns a {spec, notes} envelope: notes describe what
// couldn't be mapped cleanly (renamed services, unrecognized images kept as
// subcharts, dropped extra ports) so the UI can surface them for review. Like
// text-to-config it does NOT hard-validate — the draft is meant to be edited.
func (s *Server) handleComposeToConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST is allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Compose string `json:"compose"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Compose) == "" {
		http.Error(w, "compose file is required", http.StatusBadRequest)
		return
	}
	spec, notes, err := compose.ToSpec([]byte(body.Compose))
	if err != nil {
		// Parse failures and empty/serviceless files are user-fixable input
		// errors, surfaced verbatim like the rest of the API.
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Spec  engine.Spec `json:"spec"`
		Notes []string    `json:"notes"`
	}{Spec: engine.Normalize(spec), Notes: notes})
}
