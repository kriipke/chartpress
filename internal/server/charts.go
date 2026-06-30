// internal/server/charts.go
package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

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

const presignExpiry = 15 * time.Minute

// summarize maps a ChartpressConfig CR to the Charts-row shape. When the operator
// has marked it Ready for the CURRENT generation and recorded an artifactKey, it
// mints a fresh presigned GET URL; any presign error is logged and leaves
// downloadUrl empty (the row still renders). downloadUrl stays empty for every
// non-Ready phase and for a Ready status left over from a superseded spec.
func summarize(ctx context.Context, p Presigner, obj unstructured.Unstructured) chartSummary {
	subs, _, _ := unstructured.NestedSlice(obj.Object, "spec", "subcharts")
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	if phase == "" {
		phase = "Pending"
	}
	msg, _, _ := unstructured.NestedString(obj.Object, "status", "message")
	lastGen, _, _ := unstructured.NestedString(obj.Object, "status", "lastGenerated")
	cs := chartSummary{
		Name:          obj.GetName(),
		Phase:         phase,
		SubchartCount: len(subs),
		LastGenerated: lastGen,
		Message:       msg,
	}
	// Only mint a URL when the Ready status reflects the spec the API server
	// currently holds. The operator stamps observedGeneration to
	// metadata.generation only on a successful render, so when a user reapplies a
	// changed spec the generation increments while the prior Ready status (and its
	// artifactKey) lingers until the operator reconciles. Without this gate /charts
	// would hand out a presigned URL for the stale artifact — indefinitely if the
	// operator is down. Mirrors the operator's own Ready short-circuit.
	observed, _, _ := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	if phase == "Ready" && observed == obj.GetGeneration() && p != nil {
		if key, _, _ := unstructured.NestedString(obj.Object, "status", "artifactKey"); key != "" {
			if url, err := p.PresignGet(ctx, key, presignExpiry); err != nil {
				log.Printf("[ERROR] presign %q: %v", key, err)
			} else {
				cs.DownloadURL = url
			}
		}
	}
	return cs
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
		out = append(out, summarize(r.Context(), s.Presigner, it))
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
	_ = json.NewEncoder(w).Encode(summarize(r.Context(), s.Presigner, *obj))
}
