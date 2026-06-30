// internal/server/charts_test.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// fakeLister serves canned CRs.
type fakeLister struct {
	items []unstructured.Unstructured
}

func (f *fakeLister) List(ctx context.Context, namespace string) ([]unstructured.Unstructured, error) {
	return f.items, nil
}
func (f *fakeLister) Get(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	for i := range f.items {
		if f.items[i].GetName() == name {
			return &f.items[i], nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Group: apiGroup, Resource: "chartpressconfigs"}, name)
}

func crWith(name, phase, message string, subcharts int, lastGen string) unstructured.Unstructured {
	subs := make([]interface{}, subcharts)
	for i := range subs {
		subs[i] = map[string]interface{}{"name": "s", "workload": "deployment"}
	}
	status := map[string]interface{}{}
	if phase != "" {
		status["phase"] = phase
	}
	if message != "" {
		status["message"] = message
	}
	if lastGen != "" {
		status["lastGenerated"] = lastGen
	}
	return unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apiVersionV1alpha1,
		"kind":       kindChartpressConfig,
		"metadata":   map[string]interface{}{"name": name},
		"spec":       map[string]interface{}{"umbrellaChartName": name, "subcharts": subs},
		"status":     status,
	}}
}

func TestChartsListMapsFields(t *testing.T) {
	srv := &Server{Lister: &fakeLister{items: []unstructured.Unstructured{
		crWith("ready-one", "Ready", "", 3, "2026-06-30T03:10:00Z"),
		crWith("fresh-one", "", "", 1, ""), // empty status → Pending
		crWith("bad-one", "Failed", "render blew up", 2, ""),
	}}, Namespace: "ns"}

	req := httptest.NewRequest(http.MethodGet, "/charts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got []chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	byName := map[string]chartSummary{}
	for _, c := range got {
		byName[c.Name] = c
	}
	if c := byName["ready-one"]; c.Phase != "Ready" || c.SubchartCount != 3 || c.LastGenerated != "2026-06-30T03:10:00Z" || c.DownloadURL != "" {
		t.Fatalf("ready-one = %+v (downloadUrl must be empty when the Server has no Presigner)", c)
	}
	if c := byName["fresh-one"]; c.Phase != "Pending" {
		t.Fatalf("empty status should map to Pending, got %q", c.Phase)
	}
	if c := byName["bad-one"]; c.Phase != "Failed" || c.Message != "render blew up" {
		t.Fatalf("bad-one = %+v", c)
	}
}

func TestChartByNameFound(t *testing.T) {
	srv := &Server{Lister: &fakeLister{items: []unstructured.Unstructured{
		crWith("demo", "Generating", "", 2, ""),
	}}}
	req := httptest.NewRequest(http.MethodGet, "/charts/demo", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "demo" || got.Phase != "Generating" || got.SubchartCount != 2 {
		t.Fatalf("got = %+v", got)
	}
}

func TestChartByNameNotFound(t *testing.T) {
	srv := &Server{Lister: &fakeLister{}}
	req := httptest.NewRequest(http.MethodGet, "/charts/missing", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

type fakePresigner struct {
	url       string
	err       error
	lastKey   string
	callCount int
}

func (p *fakePresigner) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	p.callCount++
	p.lastKey = key
	return p.url, p.err
}

func crReady(name, artifactKey string) unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": apiVersionV1alpha1,
		"kind":       kindChartpressConfig,
		"metadata":   map[string]interface{}{"name": name},
		"spec": map[string]interface{}{
			"umbrellaChartName": name,
			"subcharts":         []interface{}{map[string]interface{}{"name": "api", "workload": "deployment"}},
		},
		"status": map[string]interface{}{"phase": "Ready", "artifactKey": artifactKey},
	}}
}

func TestChartsMintsDownloadURLWhenReady(t *testing.T) {
	pre := &fakePresigner{url: "https://s3.example.com/charts/demo.zip?sig=abc"}
	srv := &Server{
		Lister:    &fakeLister{items: []unstructured.Unstructured{crReady("demo", "charts/demo.zip")}},
		Presigner: pre,
		Namespace: "ns",
	}
	req := httptest.NewRequest(http.MethodGet, "/charts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var got []chartSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].DownloadURL != "https://s3.example.com/charts/demo.zip?sig=abc" {
		t.Fatalf("downloadUrl = %q (got %+v)", got[0].DownloadURL, got)
	}
	if pre.lastKey != "charts/demo.zip" {
		t.Fatalf("presigned key = %q, want charts/demo.zip", pre.lastKey)
	}
}

func TestChartsNoDownloadURLWhenNotReady(t *testing.T) {
	pre := &fakePresigner{url: "https://should-not-be-used"}
	srv := &Server{
		Lister: &fakeLister{items: []unstructured.Unstructured{
			crWith("pending-one", "Generating", "", 1, ""), // existing helper; no artifactKey
		}},
		Presigner: pre,
		Namespace: "ns",
	}
	req := httptest.NewRequest(http.MethodGet, "/charts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var got []chartSummary
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got) != 1 || got[0].DownloadURL != "" {
		t.Fatalf("downloadUrl must be empty when not Ready; got %+v", got)
	}
	if pre.callCount != 0 {
		t.Fatalf("presigner must not be called for non-Ready charts; calls=%d", pre.callCount)
	}
}
