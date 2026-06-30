// internal/engine/linked_test.go
package engine

import (
	"regexp"
	"strings"
	"testing"
)

func TestLinkedFalseInlinesUmbrellaDefines(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             func() Rules { r := DefaultRules(); r.LinkedTemplates = false; return r }(),
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	deps := ch.Dependencies()
	if len(deps) != 1 {
		t.Fatalf("want 1 subchart, got %d", len(deps))
	}
	sub := deps[0]
	var helpers string
	for _, tf := range sub.Templates {
		if tf.Name == "templates/_helpers.tpl" {
			helpers = string(tf.Data)
		}
	}
	// The inlined defines carry the umbrella-NAME prefix (demo.*), matching the
	// subchart's renamed include stubs (`include "demo.deployment"`), so the
	// subchart resolves them standalone. (A pristine umbrella-chart.* define would
	// be dead — no include references it after rename.)
	if !strings.Contains(helpers, `define "demo.deployment"`) {
		t.Fatalf("expected renamed umbrella deployment define inlined into subchart helpers:\n%s", helpers)
	}
	if strings.Contains(helpers, `define "umbrella-chart.deployment"`) {
		t.Fatalf("inlined defines must be renamed (demo.*), not the pristine umbrella-chart.* names:\n%s", helpers)
	}

	// Rendered as part of the umbrella, it still produces a Deployment.
	man := allManifests(renderChart(t, ch))
	if !strings.Contains(man, "kind: Deployment") {
		t.Fatalf("inlined subchart should still render a Deployment in the umbrella:\n%s", man)
	}

	// The real point of linked_templates=false (design §4.2): the subchart has ZERO
	// dependency on the umbrella's partials — every umbrella-named template the
	// subchart references (include/template "demo.*") must be defined within the
	// subchart's own templates. With the pre-fix pristine inlining the subchart
	// referenced "demo.*" (renamed includes) but only defined "umbrella-chart.*",
	// leaving every umbrella-partial reference dangling.
	var subText strings.Builder
	for _, tf := range sub.Templates {
		subText.Write(tf.Data)
		subText.WriteString("\n")
	}
	text := subText.String()
	defined := map[string]bool{}
	for _, m := range regexp.MustCompile(`define\s+"(demo\.[A-Za-z0-9]+)"`).FindAllStringSubmatch(text, -1) {
		defined[m[1]] = true
	}
	refs := regexp.MustCompile(`(?:include|template)\s+"(demo\.[A-Za-z0-9]+)"`).FindAllStringSubmatch(text, -1)
	if len(refs) == 0 {
		t.Fatalf("expected the subchart to reference umbrella-named templates (demo.*):\n%s", text)
	}
	for _, m := range refs {
		if !defined[m[1]] {
			t.Fatalf("subchart references %q but does not define it locally — not self-contained:\n%s", m[1], text)
		}
	}
}

func TestLinkedTrueDoesNotInline(t *testing.T) {
	ch, err := BuildChart(basicSpec(), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	// defaults linked=true → subchart carries only its own defines, no inlined
	// umbrella workload define.
	for _, d := range ch.Dependencies() {
		for _, tf := range d.Templates {
			if tf.Name == "templates/_helpers.tpl" && strings.Contains(string(tf.Data), `define "demo.deployment"`) {
				t.Fatalf("linked=true should not inline umbrella defines")
			}
		}
	}
}
