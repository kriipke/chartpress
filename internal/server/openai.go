// internal/server/openai.go
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/kriipke/chartpress/internal/engine"
)

// openAIDrafter drafts a Spec via the OpenAI Responses API with strict
// JSON-schema structured output.
type openAIDrafter struct {
	apiKey     string
	model      string
	endpoint   string
	httpClient *http.Client
}

func newOpenAIDrafter() *openAIDrafter {
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4.1"
	}
	return &openAIDrafter{
		apiKey:     os.Getenv("OPENAI_API_KEY"),
		model:      model,
		endpoint:   "https://api.openai.com/v1/responses",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

const draftSystemPrompt = "You draft a chartpress Spec for a Kubernetes Helm umbrella chart from a short app description. " +
	"Choose a kebab-case umbrellaChartName, 1+ subcharts each with a kebab-case name and a workload of deployment, statefulset, or daemonset, " +
	"and a rules block. Only emit fields defined by the schema."

func (d *openAIDrafter) Draft(ctx context.Context, prompt string) (engine.Spec, error) {
	reqBody := map[string]interface{}{
		"model": d.model,
		"input": []map[string]string{
			{"role": "system", "content": draftSystemPrompt},
			{"role": "user", "content": prompt},
		},
		"text": map[string]interface{}{
			"format": map[string]interface{}{
				"type":   "json_schema",
				"name":   "chartpress_spec",
				"strict": true,
				"schema": specJSONSchema(),
			},
		},
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return engine.Spec{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.endpoint, bytes.NewReader(buf))
	if err != nil {
		return engine.Spec{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return engine.Spec{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return engine.Spec{}, fmt.Errorf("openai responses: status %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}

	var parsed struct {
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return engine.Spec{}, err
	}

	var jsonText string
	for _, o := range parsed.Output {
		if o.Type != "message" {
			continue
		}
		for _, c := range o.Content {
			if c.Type == "output_text" {
				jsonText += c.Text
			}
		}
	}
	if jsonText == "" {
		return engine.Spec{}, fmt.Errorf("openai responses: no output_text in response")
	}

	var spec engine.Spec
	if err := json.Unmarshal([]byte(jsonText), &spec); err != nil {
		return engine.Spec{}, fmt.Errorf("openai responses: parse spec: %w", err)
	}
	return spec, nil
}

// specJSONSchema is the strict structured-output schema for a chartpress Spec.
// Strict mode requires additionalProperties:false and every property in required.
func specJSONSchema() map[string]interface{} {
	boolProp := map[string]interface{}{"type": "boolean"}
	return map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"umbrellaChartName", "description", "subcharts", "rules"},
		"properties": map[string]interface{}{
			"umbrellaChartName": map[string]interface{}{"type": "string"},
			"description":       map[string]interface{}{"type": "string"},
			"subcharts": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"name", "workload", "description"},
					"properties": map[string]interface{}{
						"name":        map[string]interface{}{"type": "string"},
						"workload":    map[string]interface{}{"type": "string", "enum": engine.AllowedWorkloads},
						"description": map[string]interface{}{"type": "string"},
					},
				},
			},
			"rules": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"required": []string{
					"ingress", "common_annotations", "linked_templates",
					"resource_names_match_chart_name", "shared_secrets_config",
					"shared_newrelic_config", "generate_umbrella_readme",
					"generate_subchart_readme", "include_docs",
				},
				"properties": map[string]interface{}{
					"ingress":                         map[string]interface{}{"type": "string", "enum": engine.AllowedIngress},
					"common_annotations":              boolProp,
					"linked_templates":                boolProp,
					"resource_names_match_chart_name": boolProp,
					"shared_secrets_config":           boolProp,
					"shared_newrelic_config":          boolProp,
					"generate_umbrella_readme":        boolProp,
					"generate_subchart_readme":        boolProp,
					"include_docs":                    boolProp,
				},
			},
		},
	}
}
