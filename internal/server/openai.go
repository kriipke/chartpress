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

const draftSystemPrompt = `You draft a chartpress Spec for a Kubernetes Helm umbrella chart from a short app description.

Choose a kebab-case umbrellaChartName and a rules block. For each component the user describes that is THEIR OWN CODE, emit one subchart with a kebab-case name and a pattern.

Classifying the pattern (closed set — you MUST pick the nearest of these; never invent traits):
- api-microservice: REST/HTTP backend contacted directly.
- grpc-service: internal gRPC server.
- edge-gateway: the public entry / BFF / API gateway (usually the only externally-reachable component).
- web-frontend: SPA, SSR app, or static site.
- worker: pulls its work and serves nothing — "worker", "consumer", "executor", "processor", "job runner", email/image/async handlers.
- stream-processor: Kafka consumer group, CDC consumer, partition-coupled stream stage.
- scheduler: runs exactly one instance — "scheduler", "cron dispatcher", "migrator", "outbox relay", "leader-elected" controller.
- realtime-gateway: WebSocket/SSE/push server with long-lived connections.
- ml-inference: model server, embedding service, LLM wrapper (developer-written).
- webhook-ingest: receives third-party webhooks (Stripe/GitHub/Twilio).
- admin-dashboard: internal admin/ops UI.
- node-agent: per-node daemon (log shipper, node exporter, security agent).
Do NOT use "custom" — that value is reserved for humans. Always choose the closest real pattern.

Trait overrides (exposure, port, ingress, scaling, shared_env, workload): emit one ONLY when the user's text states the fact explicitly (e.g. "listens on port 3000", "must not run more than one", "over gRPC"). Otherwise leave it null and let the pattern's default apply. Never infer an override.

Guardrails:
- Dependency rule: self-hosted infrastructure the user did NOT write — a database, cache, message broker, search engine (Postgres, MySQL, Redis, Valkey, Kafka, RabbitMQ, MongoDB, Elasticsearch) — is NOT a subchart. List it in the top-level "dependencies" array using its lowercase key. Never emit a subchart for it.
- Sidecar rule: a sidecar (cloud-sql-proxy, envoy) is part of its owning component, never its own subchart. Ignore it at draft time.

Only emit fields defined by the schema.`

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

// nullableEnum builds a strict-mode nullable enum property: the model may emit
// one of the values OR null (null = "no explicit override, use the pattern
// default"). Strict structured output requires null to be a member of enum.
func nullableEnum(values []string) map[string]interface{} {
	enum := make([]interface{}, 0, len(values)+1)
	for _, v := range values {
		enum = append(enum, v)
	}
	enum = append(enum, nil)
	return map[string]interface{}{"type": []string{"string", "null"}, "enum": enum}
}

// specJSONSchema is the strict structured-output schema for a chartpress Spec.
// Strict mode requires additionalProperties:false and every property in
// required; "optional" trait overrides are modeled as nullable (null = unset).
func specJSONSchema() map[string]interface{} {
	boolProp := map[string]interface{}{"type": "boolean"}
	nullableBool := map[string]interface{}{"type": []string{"boolean", "null"}}
	nullableInt := map[string]interface{}{"type": []string{"integer", "null"}}
	return map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"umbrellaChartName", "description", "subcharts", "dependencies", "rules"},
		"properties": map[string]interface{}{
			"umbrellaChartName": map[string]interface{}{"type": "string"},
			"description":       map[string]interface{}{"type": "string"},
			"dependencies": map[string]interface{}{
				"type":        "array",
				"description": "Self-hosted infrastructure the user did not write, by registry key. Never a subchart.",
				"items":       map[string]interface{}{"type": "string", "enum": engine.DependencyKeys()},
			},
			"subcharts": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"name", "description", "pattern", "workload", "exposure", "port", "ingress", "scaling", "shared_env"},
					"properties": map[string]interface{}{
						"name":        map[string]interface{}{"type": "string"},
						"description": map[string]interface{}{"type": "string"},
						"pattern":     map[string]interface{}{"type": "string", "enum": engine.PatternIDs(true)},
						// Trait overrides — null unless the user's text states the fact.
						"workload":   nullableEnum(engine.AllowedWorkloads),
						"exposure":   nullableEnum(engine.AllowedExposures),
						"port":       nullableInt,
						"ingress":    nullableBool,
						"scaling":    nullableEnum(engine.AllowedScalings),
						"shared_env": nullableBool,
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
