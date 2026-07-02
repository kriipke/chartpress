// internal/server/spec.go
package server

import (
	"encoding/json"
	"io"

	"github.com/kriipke/chartpress/internal/engine"
)

// requestRules mirrors engine.Rules but with pointers so we can tell an omitted
// field from an explicit false. engine.Normalize fills the ingress default but
// never fills booleans, so the defaulting MUST happen here.
type requestRules struct {
	Ingress                     *string `json:"ingress"`
	CommonAnnotations           *bool   `json:"common_annotations"`
	LinkedTemplates             *bool   `json:"linked_templates"`
	ResourceNamesMatchChartName *bool   `json:"resource_names_match_chart_name"`
	SharedSecretsConfig         *bool   `json:"shared_secrets_config"`
	SharedNewrelicConfig        *bool   `json:"shared_newrelic_config"`
	GenerateUmbrellaReadme      *bool   `json:"generate_umbrella_readme"`
	GenerateSubchartReadme      *bool   `json:"generate_subchart_readme"`
	IncludeDocs                 *bool   `json:"include_docs"`
	GenerateHandoff             *bool   `json:"generate_handoff"`
}

type requestSpec struct {
	UmbrellaChartName string            `json:"umbrellaChartName"`
	Description       string            `json:"description"`
	Subcharts         []engine.Subchart `json:"subcharts"`
	Dependencies      []string          `json:"dependencies"`
	Rules             *requestRules     `json:"rules"`
}

// decodeSpec reads a Spec request body, fills omitted rule fields from
// engine.DefaultRules(), and returns a normalized engine.Spec. It does NOT
// validate — callers run engine.Validate.
func decodeSpec(r io.Reader) (engine.Spec, error) {
	var req requestSpec
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return engine.Spec{}, err
	}

	rules := engine.DefaultRules()
	if rr := req.Rules; rr != nil {
		if rr.Ingress != nil {
			rules.Ingress = *rr.Ingress
		}
		if rr.CommonAnnotations != nil {
			rules.CommonAnnotations = *rr.CommonAnnotations
		}
		if rr.LinkedTemplates != nil {
			rules.LinkedTemplates = *rr.LinkedTemplates
		}
		if rr.ResourceNamesMatchChartName != nil {
			rules.ResourceNamesMatchChartName = *rr.ResourceNamesMatchChartName
		}
		if rr.SharedSecretsConfig != nil {
			rules.SharedSecretsConfig = *rr.SharedSecretsConfig
		}
		if rr.SharedNewrelicConfig != nil {
			rules.SharedNewrelicConfig = *rr.SharedNewrelicConfig
		}
		if rr.GenerateUmbrellaReadme != nil {
			rules.GenerateUmbrellaReadme = *rr.GenerateUmbrellaReadme
		}
		if rr.GenerateSubchartReadme != nil {
			rules.GenerateSubchartReadme = *rr.GenerateSubchartReadme
		}
		if rr.IncludeDocs != nil {
			rules.IncludeDocs = *rr.IncludeDocs
		}
		// engine.Rules.GenerateHandoff is already a pointer (nil = enabled),
		// so the explicit value passes through unchanged.
		rules.GenerateHandoff = rr.GenerateHandoff
	}

	spec := engine.Spec{
		UmbrellaChartName: req.UmbrellaChartName,
		Description:       req.Description,
		Subcharts:         req.Subcharts,
		Dependencies:      req.Dependencies,
		Rules:             rules,
	}
	return engine.Normalize(spec), nil
}
