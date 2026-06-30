// internal/server/texttoconfig_test.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
)

type fakeDrafter struct {
	spec engine.Spec
	err  error
}

func (d fakeDrafter) Draft(ctx context.Context, prompt string) (engine.Spec, error) {
	return d.spec, d.err
}

func TestTextToConfigReturnsSpec(t *testing.T) {
	drafted := engine.Normalize(engine.Spec{
		UmbrellaChartName: "shop",
		Subcharts:         []engine.Subchart{{Name: "web", Workload: "deployment"}},
		Rules:             engine.DefaultRules(),
	})
	srv := &Server{Drafter: fakeDrafter{spec: drafted}}
	req := httptest.NewRequest(http.MethodPost, "/text-to-config", strings.NewReader(`{"prompt":"an online shop"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got engine.Spec
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.UmbrellaChartName != "shop" || len(got.Subcharts) != 1 || got.Rules.Ingress != "alb" {
		t.Fatalf("got = %+v", got)
	}
}

func TestTextToConfigRejectsEmptyPrompt(t *testing.T) {
	srv := &Server{Drafter: fakeDrafter{}}
	req := httptest.NewRequest(http.MethodPost, "/text-to-config", strings.NewReader(`{"prompt":"  "}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
