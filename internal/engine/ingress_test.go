// internal/engine/ingress_test.go
package engine

import (
	"strings"
	"testing"
)

func ingressSpec(controller string) Spec {
	s := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             func() Rules { r := DefaultRules(); r.Ingress = controller; return r }(),
	})
	return s
}

func renderWithHost(t *testing.T, controller string) string {
	t.Helper()
	ch, err := BuildChart(ingressSpec(controller), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart(%s): %v", controller, err)
	}
	// set a host on the subchart so the ingress guard fires
	for _, d := range ch.Dependencies() {
		d.Values["ingress"] = map[string]interface{}{"host": "api.example.com", "path": "/"}
	}
	return allManifests(renderChart(t, ch))
}

func TestIngressClasses(t *testing.T) {
	cases := map[string]string{
		"alb":     "ingressClassName: alb",
		"nginx":   "ingressClassName: nginx",
		"traefik": "ingressClassName: traefik",
		"gce":     "ingressClassName: gce",
	}
	for ctrl, want := range cases {
		man := renderWithHost(t, ctrl)
		if !strings.Contains(man, "kind: Ingress") || !strings.Contains(man, want) {
			t.Fatalf("%s: expected Ingress with %q, got:\n%s", ctrl, want, man)
		}
	}
}

func TestIngressNoneOmitsIngress(t *testing.T) {
	man := renderWithHost(t, "none")
	if strings.Contains(man, "kind: Ingress") {
		t.Fatalf("ingress=none should emit no Ingress, got:\n%s", man)
	}
}
