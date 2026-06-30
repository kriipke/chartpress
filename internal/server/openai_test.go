// internal/server/openai_test.go
package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIDrafterParsesResponsesOutput(t *testing.T) {
	// Stand in for the OpenAI Responses API: echo a spec back as output_text.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/" {
			t.Errorf("path = %q, want /", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("auth header = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"model":"gpt-4.1"`) {
			t.Errorf("request missing model: %s", body)
		}
		if !strings.Contains(string(body), "json_schema") {
			t.Errorf("request missing structured-output schema: %s", body)
		}
		// The structured-output request must stay strict (regression guard).
		for _, marker := range []string{`"strict":true`, `"additionalProperties":false`, `"deployment"`} {
			if !strings.Contains(string(body), marker) {
				t.Errorf("request missing strict-schema marker %q: %s", marker, body)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"output": []interface{}{
				map[string]interface{}{
					"type": "message",
					"content": []interface{}{
						map[string]interface{}{
							"type": "output_text",
							"text": `{"umbrellaChartName":"shop","description":"d","subcharts":[{"name":"web","workload":"deployment","description":""}],"rules":{"ingress":"nginx","common_annotations":false,"linked_templates":true,"resource_names_match_chart_name":false,"shared_secrets_config":false,"shared_newrelic_config":false,"generate_umbrella_readme":true,"generate_subchart_readme":true,"include_docs":true}}`,
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	d := &openAIDrafter{apiKey: "test-key", model: "gpt-4.1", endpoint: srv.URL, httpClient: srv.Client()}
	spec, err := d.Draft(context.Background(), "an online shop")
	if err != nil {
		t.Fatalf("Draft: %v", err)
	}
	if spec.UmbrellaChartName != "shop" || spec.Rules.Ingress != "nginx" || len(spec.Subcharts) != 1 {
		t.Fatalf("spec = %+v", spec)
	}
}
