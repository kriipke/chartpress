// internal/server/charts.go
package server

import (
	"encoding/json"
	"net/http"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type chartSummary struct {
	Name          string `json:"name"`
	Phase         string `json:"phase"`
	SubchartCount int    `json:"subchartCount"`
	LastGenerated string `json:"lastGenerated,omitempty"`
	Message       string `json:"message,omitempty"`
	DownloadURL   string `json:"downloadUrl,omitempty"`
}

// summarize maps a ChartpressConfig CR to the Charts-row shape. Phase defaults
// to "Pending" before the operator sets status. downloadUrl stays empty in
// Phase 2 — presigned URLs are minted in Phase 3 once status.artifactKey exists.
func summarize(obj unstructured.Unstructured) chartSummary {
	subs, _, _ := unstructured.NestedSlice(obj.Object, "spec", "subcharts")
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	if phase == "" {
		phase = "Pending"
	}
	msg, _, _ := unstructured.NestedString(obj.Object, "status", "message")
	lastGen, _, _ := unstructured.NestedString(obj.Object, "status", "lastGenerated")
	return chartSummary{
		Name:          obj.GetName(),
		Phase:         phase,
		SubchartCount: len(subs),
		LastGenerated: lastGen,
		Message:       msg,
	}
}

func (s *Server) handleCharts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET is allowed", http.StatusMethodNotAllowed)
		return
	}
	items, err := s.Lister.List(r.Context(), s.Namespace)
	if err != nil {
		http.Error(w, "failed to list charts: "+err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]chartSummary, 0, len(items))
	for _, it := range items {
		out = append(out, summarize(it))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleChartByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET is allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/charts/")
	if name == "" {
		http.Error(w, "missing chart name", http.StatusBadRequest)
		return
	}
	obj, err := s.Lister.Get(r.Context(), s.Namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			http.Error(w, "chart not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get chart: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summarize(*obj))
}
