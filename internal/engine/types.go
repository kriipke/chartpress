// internal/engine/types.go
package engine

import (
	"fmt"
	"regexp"
	"strings"
)

// Spec is the chartpress generation spec — the body of /generate and the .spec
// of a ChartpressConfig manifest.
type Spec struct {
	UmbrellaChartName string     `json:"umbrellaChartName" yaml:"umbrellaChartName"`
	Description       string     `json:"description,omitempty" yaml:"description,omitempty"`
	Subcharts         []Subchart `json:"subcharts" yaml:"subcharts"`
	Rules             Rules      `json:"rules" yaml:"rules"`
	// Dependencies are self-hosted infrastructure (databases, caches, brokers)
	// emitted as umbrella Chart.yaml dependencies: entries from the curated
	// registry — never as generated subcharts. Each is a registry key
	// (see dependencies.json); unknown keys become TODO stubs.
	Dependencies []string `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

type Subchart struct {
	Name        string `json:"name" yaml:"name"`
	Workload    string `json:"workload,omitempty" yaml:"workload,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Pattern + traits. The spec stores INTENT: zero values mean "unset —
	// resolve from the pattern" (bools use pointers because false is a valid
	// explicit choice; ""/0 are never valid enum/port values, so plain types
	// suffice for the rest). ResolveTraits applies the pattern's defaults
	// dependently; see resolve.go.
	Pattern   string `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Exposure  string `json:"exposure,omitempty" yaml:"exposure,omitempty"`
	Port      int    `json:"port,omitempty" yaml:"port,omitempty"`
	Ingress   *bool  `json:"ingress,omitempty" yaml:"ingress,omitempty"`
	Scaling   string `json:"scaling,omitempty" yaml:"scaling,omitempty"`
	SharedEnv *bool  `json:"shared_env,omitempty" yaml:"shared_env,omitempty"`
}

type Rules struct {
	Ingress                     string `json:"ingress" yaml:"ingress"`
	CommonAnnotations           bool   `json:"common_annotations" yaml:"common_annotations"`
	LinkedTemplates             bool   `json:"linked_templates" yaml:"linked_templates"`
	ResourceNamesMatchChartName bool   `json:"resource_names_match_chart_name" yaml:"resource_names_match_chart_name"`
	SharedSecretsConfig         bool   `json:"shared_secrets_config" yaml:"shared_secrets_config"`
	SharedNewrelicConfig        bool   `json:"shared_newrelic_config" yaml:"shared_newrelic_config"`
	GenerateUmbrellaReadme      bool   `json:"generate_umbrella_readme" yaml:"generate_umbrella_readme"`
	GenerateSubchartReadme      bool   `json:"generate_subchart_readme" yaml:"generate_subchart_readme"`
	IncludeDocs                 bool   `json:"include_docs" yaml:"include_docs"`
	// GenerateHandoff gates the generated HANDOFF.md. Pointer so every decode
	// path (JSON body, CLI YAML, CRD) treats an *omitted* field as true
	// without special-casing: nil means enabled.
	GenerateHandoff *bool `json:"generate_handoff,omitempty" yaml:"generate_handoff,omitempty"`
}

// HandoffEnabled reports whether the HANDOFF.md should be generated; an unset
// generate_handoff means yes (it's the product thesis).
func (r Rules) HandoffEnabled() bool {
	return r.GenerateHandoff == nil || *r.GenerateHandoff
}

var (
	AllowedWorkloads = []string{"deployment", "statefulset", "daemonset"}
	AllowedIngress   = []string{"alb", "nginx", "traefik", "istio", "gce", "none"}
	AllowedExposures = []string{"http", "grpc", "tcp", "none"}
	AllowedScalings  = []string{"auto", "fixed", "singleton"}
	nameRE           = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
)

// DefaultRules returns the locked rule defaults.
func DefaultRules() Rules {
	return Rules{
		Ingress:                "alb",
		LinkedTemplates:        true,
		GenerateUmbrellaReadme: true,
		GenerateSubchartReadme: true,
		IncludeDocs:            true,
	}
}

func sanitizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// Normalize trims/lowercases names, workloads, and trait enums, and fills an
// empty ingress with the default. It does NOT fill omitted booleans (that is
// the decode layer's job) and does NOT resolve traits (see ResolveTraits).
func Normalize(s Spec) Spec {
	s.UmbrellaChartName = sanitizeName(s.UmbrellaChartName)
	s.Description = strings.TrimSpace(s.Description)
	for i := range s.Subcharts {
		s.Subcharts[i].Name = sanitizeName(s.Subcharts[i].Name)
		s.Subcharts[i].Workload = sanitizeName(s.Subcharts[i].Workload)
		s.Subcharts[i].Description = strings.TrimSpace(s.Subcharts[i].Description)
		s.Subcharts[i].Pattern = sanitizeName(s.Subcharts[i].Pattern)
		s.Subcharts[i].Exposure = sanitizeName(s.Subcharts[i].Exposure)
		s.Subcharts[i].Scaling = sanitizeName(s.Subcharts[i].Scaling)
	}
	for i := range s.Dependencies {
		s.Dependencies[i] = sanitizeName(s.Dependencies[i])
	}
	s.Rules.Ingress = strings.ToLower(strings.TrimSpace(s.Rules.Ingress))
	if s.Rules.Ingress == "" {
		s.Rules.Ingress = "alb"
	}
	return s
}

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// Validate enforces the spec-level invariants: name regex, >=1 subchart, the
// ingress enum, and — via ResolveTraits — the pattern/trait invariants. By
// construction of dependent defaulting, only keys the user actually wrote can
// fail trait validation.
func Validate(s Spec) error {
	if !nameRE.MatchString(s.UmbrellaChartName) {
		return fmt.Errorf("umbrellaChartName %q must match %s", s.UmbrellaChartName, nameRE.String())
	}
	if len(s.Subcharts) == 0 {
		return fmt.Errorf("at least one subchart is required")
	}
	if !contains(AllowedIngress, s.Rules.Ingress) {
		return fmt.Errorf("rules.ingress %q invalid (allowed: %v)", s.Rules.Ingress, AllowedIngress)
	}
	seen := map[string]bool{}
	for _, sc := range s.Subcharts {
		if !nameRE.MatchString(sc.Name) {
			return fmt.Errorf("subchart name %q must match %s", sc.Name, nameRE.String())
		}
		// "global" is helm's reserved values key; a subchart by that name would
		// collide with the umbrella values' global block.
		if sc.Name == "global" {
			return fmt.Errorf("subchart name %q is reserved", sc.Name)
		}
		if seen[sc.Name] {
			return fmt.Errorf("duplicate subchart name %q", sc.Name)
		}
		seen[sc.Name] = true
		if _, err := ResolveTraits(sc, s.Rules); err != nil {
			return fmt.Errorf("subchart %q: %w", sc.Name, err)
		}
	}
	return nil
}
