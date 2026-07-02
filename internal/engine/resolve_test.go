// internal/engine/resolve_test.go
package engine

import (
	"strings"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func TestResolveOmittedPatternIsBackCompat(t *testing.T) {
	// A pre-pattern spec entry must resolve to the api-microservice shape.
	rt, err := ResolveTraits(Subchart{Name: "api", Workload: "deployment"}, DefaultRules())
	if err != nil {
		t.Fatalf("ResolveTraits: %v", err)
	}
	want := ResolvedTraits{Pattern: "api-microservice", Workload: "deployment", Exposure: "http", Port: 8080, Ingress: true, Scaling: "auto", SharedEnv: true}
	if rt != want {
		t.Fatalf("resolved = %+v, want %+v", rt, want)
	}
}

func TestResolvePatternDefaults(t *testing.T) {
	cases := []struct {
		pattern string
		want    ResolvedTraits
	}{
		{"worker", ResolvedTraits{Pattern: "worker", Workload: "deployment", Exposure: "none", Port: 0, Ingress: false, Scaling: "auto", SharedEnv: true}},
		{"scheduler", ResolvedTraits{Pattern: "scheduler", Workload: "deployment", Exposure: "none", Port: 0, Ingress: false, Scaling: "singleton", SharedEnv: true}},
		{"grpc-service", ResolvedTraits{Pattern: "grpc-service", Workload: "deployment", Exposure: "grpc", Port: 50051, Ingress: false, Scaling: "auto", SharedEnv: true}},
		{"node-agent", ResolvedTraits{Pattern: "node-agent", Workload: "daemonset", Exposure: "none", Port: 0, Ingress: false, Scaling: "fixed", SharedEnv: false}},
		{"web-frontend", ResolvedTraits{Pattern: "web-frontend", Workload: "deployment", Exposure: "http", Port: 8080, Ingress: true, Scaling: "auto", SharedEnv: false}},
	}
	for _, c := range cases {
		rt, err := ResolveTraits(Subchart{Name: "x", Pattern: c.pattern}, DefaultRules())
		if err != nil {
			t.Fatalf("%s: %v", c.pattern, err)
		}
		if rt != c.want {
			t.Fatalf("%s resolved = %+v, want %+v", c.pattern, rt, c.want)
		}
	}
}

// Dependent defaulting: a default can never cause a validation error — only
// keys the user actually wrote can fail.
func TestResolveDependentDefaulting(t *testing.T) {
	// api-microservice + exposure:tcp — the inherited ingress:true default is
	// clamped to false, silently.
	rt, err := ResolveTraits(Subchart{Name: "x", Exposure: "tcp"}, DefaultRules())
	if err != nil {
		t.Fatalf("tcp override should not error: %v", err)
	}
	if rt.Ingress {
		t.Fatalf("defaulted ingress must clamp to false for tcp exposure")
	}
	if rt.Port != 8080 {
		t.Fatalf("tcp port default = %d, want 8080", rt.Port)
	}

	// custom + exposure:none stays valid: ingress default clamps, port absent.
	rt, err = ResolveTraits(Subchart{Name: "x", Pattern: "custom", Exposure: "none"}, DefaultRules())
	if err != nil {
		t.Fatalf("custom+none should not error: %v", err)
	}
	if rt.Ingress || rt.Port != 0 {
		t.Fatalf("custom+none resolved = %+v", rt)
	}

	// rules.ingress none clamps every defaulted ingress to false.
	rules := DefaultRules()
	rules.Ingress = "none"
	rt, err = ResolveTraits(Subchart{Name: "x"}, rules)
	if err != nil {
		t.Fatalf("rules.ingress none should not error: %v", err)
	}
	if rt.Ingress {
		t.Fatalf("ingress must clamp to false when rules.ingress is none")
	}

	// daemonset override clamps the pattern's scaling default (auto) to fixed.
	rt, err = ResolveTraits(Subchart{Name: "x", Workload: "daemonset"}, DefaultRules())
	if err != nil {
		t.Fatalf("daemonset override: %v", err)
	}
	if rt.Scaling != "fixed" {
		t.Fatalf("daemonset scaling = %q, want fixed", rt.Scaling)
	}
}

func TestResolveExplicitKeysFail(t *testing.T) {
	cases := []struct {
		name string
		sc   Subchart
		want string
	}{
		{"unknown pattern", Subchart{Pattern: "databass"}, "unknown pattern"},
		{"bad exposure", Subchart{Exposure: "udp"}, "invalid exposure"},
		{"bad scaling", Subchart{Scaling: "many"}, "invalid scaling"},
		{"bad port", Subchart{Port: 70000}, "out of range"},
		{"explicit ingress on none", Subchart{Exposure: "none", Ingress: boolPtr(true)}, "requires exposure"},
		{"singleton daemonset", Subchart{Workload: "daemonset", Scaling: "singleton"}, "daemonset"},
		{"auto daemonset", Subchart{Workload: "daemonset", Scaling: "auto"}, "daemonset"},
	}
	for _, c := range cases {
		if _, err := ResolveTraits(c.sc, DefaultRules()); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: err = %v, want containing %q", c.name, err, c.want)
		}
	}

	// Explicit ingress:true with rules.ingress none errors (the same key would
	// silently clamp if defaulted).
	rules := DefaultRules()
	rules.Ingress = "none"
	if _, err := ResolveTraits(Subchart{Ingress: boolPtr(true)}, rules); err == nil {
		t.Fatalf("explicit ingress with rules.ingress none must error")
	}
}

func TestResolveOverridesWin(t *testing.T) {
	rt, err := ResolveTraits(Subchart{Name: "pricing", Pattern: "grpc-service", Ingress: boolPtr(true), Port: 9000}, DefaultRules())
	if err != nil {
		t.Fatalf("ResolveTraits: %v", err)
	}
	if !rt.Ingress || rt.Port != 9000 {
		t.Fatalf("overrides not applied: %+v", rt)
	}
}

func TestWarnings(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts: []Subchart{
			{Name: "bff", Pattern: "edge-gateway"},
			{Name: "orders", Pattern: "api-microservice"}, // also ingress:true
			{Name: "admin", Pattern: "admin-dashboard"},
		},
		Rules: DefaultRules(),
	})
	warns := Warnings(spec)
	joined := strings.Join(warns, "\n")
	if !strings.Contains(joined, "edge gateway") {
		t.Fatalf("expected edge-gateway warning, got %v", warns)
	}
	if !strings.Contains(joined, "admin") || !strings.Contains(joined, "protect") {
		t.Fatalf("expected admin-dashboard warning, got %v", warns)
	}

	// A lone gateway warns about nothing.
	solo := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts: []Subchart{
			{Name: "bff", Pattern: "edge-gateway"},
			{Name: "worker", Pattern: "worker"},
		},
		Rules: DefaultRules(),
	})
	if w := Warnings(solo); len(w) != 0 {
		t.Fatalf("expected no warnings, got %v", w)
	}
}

func TestValidateRejectsExplicitTraitErrors(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "agent", Pattern: "node-agent", Scaling: "singleton"}},
		Rules:             DefaultRules(),
	})
	if err := Validate(spec); err == nil {
		t.Fatalf("singleton daemonset must fail validation")
	}
}
