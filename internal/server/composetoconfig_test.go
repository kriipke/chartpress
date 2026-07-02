package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
)

func postCompose(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	srv := &Server{} // deterministic mapping needs no dependencies
	req := httptest.NewRequest(http.MethodPost, "/compose-to-config", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestComposeToConfigReturnsSpecAndNotes(t *testing.T) {
	compose := "services:\n  api:\n    build: .\n    ports: [\"8080:8080\"]\n  db:\n    image: postgres:16\n"
	body, _ := json.Marshal(map[string]string{"compose": compose})
	rec := postCompose(t, string(body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Spec  engine.Spec `json:"spec"`
		Notes []string    `json:"notes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Spec.Subcharts) != 1 || got.Spec.Subcharts[0].Name != "api" {
		t.Fatalf("subcharts = %+v", got.Spec.Subcharts)
	}
	if len(got.Spec.Dependencies) != 1 || got.Spec.Dependencies[0] != "postgresql" {
		t.Fatalf("dependencies = %v", got.Spec.Dependencies)
	}
	if got.Spec.Rules.Ingress != "alb" {
		t.Fatalf("rules not defaulted: %+v", got.Spec.Rules)
	}
	if got.Notes == nil {
		t.Fatalf("notes should be a (possibly empty) array, not null")
	}
}

func TestComposeToConfigRejectsEmpty(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"compose": "   "})
	if rec := postCompose(t, string(body)); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestComposeToConfigRejectsInvalid(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"compose": "services: [unclosed"})
	rec := postCompose(t, string(body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid docker-compose file") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestComposeToConfigRejectsNonPost(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/compose-to-config", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}
