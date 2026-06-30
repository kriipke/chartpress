// internal/server/generate_test.go
package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// recordingApplier captures the last applied object.
type recordingApplier struct {
	ns  string
	obj *unstructured.Unstructured
	err error
}

func (a *recordingApplier) Apply(ctx context.Context, namespace string, obj *unstructured.Unstructured) error {
	a.ns, a.obj = namespace, obj
	return a.err
}

func newTestServer(a Applier) *Server {
	return &Server{Applier: a, Namespace: "chartpress-system"}
}

func TestGenerateAppliesCRAndReturnsPending(t *testing.T) {
	app := &recordingApplier{}
	srv := newTestServer(app)
	body := `{"umbrellaChartName":"demo","subcharts":[{"name":"api","workload":"deployment"}]}`
	req := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp generateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if resp.Name != "demo" || resp.Namespace != "chartpress-system" || resp.Phase != "Pending" {
		t.Fatalf("resp = %+v", resp)
	}
	if !strings.Contains(resp.ManifestYAML, "kind: ChartpressConfig") {
		t.Fatalf("manifestYaml missing kind:\n%s", resp.ManifestYAML)
	}
	if app.obj == nil || app.obj.GetName() != "demo" || app.ns != "chartpress-system" {
		t.Fatalf("applier got ns=%q obj=%v", app.ns, app.obj)
	}
}

func TestGenerateRejectsInvalidWorkload(t *testing.T) {
	srv := newTestServer(&recordingApplier{})
	body := `{"umbrellaChartName":"demo","subcharts":[{"name":"api","workload":"cronjob"}]}`
	req := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestGenerateRejectsNonPost(t *testing.T) {
	srv := newTestServer(&recordingApplier{})
	req := httptest.NewRequest(http.MethodGet, "/generate", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

// When the apiserver rejects the apply, the handler must surface a 500 rather
// than reporting success.
func TestGenerateReturns500WhenApplyFails(t *testing.T) {
	app := &recordingApplier{err: errors.New("apiserver rejected apply")}
	srv := newTestServer(app)
	body := `{"umbrellaChartName":"demo","subcharts":[{"name":"api","workload":"deployment"}]}`
	req := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", rec.Code, rec.Body.String())
	}
}
