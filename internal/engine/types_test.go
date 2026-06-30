// internal/engine/types_test.go
package engine

import "testing"

func TestDefaultRules(t *testing.T) {
	r := DefaultRules()
	if r.Ingress != "alb" || !r.LinkedTemplates || !r.GenerateUmbrellaReadme ||
		!r.GenerateSubchartReadme || !r.IncludeDocs {
		t.Fatalf("unexpected defaults: %+v", r)
	}
	if r.CommonAnnotations || r.ResourceNamesMatchChartName || r.SharedSecretsConfig || r.SharedNewrelicConfig {
		t.Fatalf("non-default booleans should be false: %+v", r)
	}
}

func TestNormalizeLowercasesAndDefaultsIngress(t *testing.T) {
	got := Normalize(Spec{
		UmbrellaChartName: "  Demo-Platform ",
		Subcharts:         []Subchart{{Name: " API ", Workload: "Deployment"}},
		Rules:             Rules{Ingress: ""},
	})
	if got.UmbrellaChartName != "demo-platform" {
		t.Fatalf("umbrella not sanitized: %q", got.UmbrellaChartName)
	}
	if got.Subcharts[0].Name != "api" || got.Subcharts[0].Workload != "deployment" {
		t.Fatalf("subchart not normalized: %+v", got.Subcharts[0])
	}
	if got.Rules.Ingress != "alb" {
		t.Fatalf("empty ingress should default to alb, got %q", got.Rules.Ingress)
	}
}

func TestValidateRejectsBadInput(t *testing.T) {
	cases := []struct {
		name string
		spec Spec
	}{
		{"empty name", Spec{Subcharts: []Subchart{{Name: "api", Workload: "deployment"}}, Rules: DefaultRules()}},
		{"no subcharts", Spec{UmbrellaChartName: "demo", Rules: DefaultRules()}},
		{"bad workload", Spec{UmbrellaChartName: "demo", Subcharts: []Subchart{{Name: "api", Workload: "job"}}, Rules: DefaultRules()}},
		{"bad ingress", Spec{UmbrellaChartName: "demo", Subcharts: []Subchart{{Name: "api", Workload: "deployment"}}, Rules: Rules{Ingress: "kong"}}},
		{"bad name chars", Spec{UmbrellaChartName: "Demo_Platform", Subcharts: []Subchart{{Name: "api", Workload: "deployment"}}, Rules: DefaultRules()}},
		{"bad subchart name", Spec{UmbrellaChartName: "demo", Subcharts: []Subchart{{Name: "Bad_api", Workload: "deployment"}}, Rules: DefaultRules()}},
	}
	for _, c := range cases {
		if err := Validate(c.spec); err == nil {
			t.Errorf("%s: expected error, got nil", c.name)
		}
	}
}

func TestValidateAcceptsGoodInput(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             DefaultRules(),
	})
	if err := Validate(spec); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}
