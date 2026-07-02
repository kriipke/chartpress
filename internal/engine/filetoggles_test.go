// internal/engine/filetoggles_test.go
package engine

import (
	"strings"
	"testing"
)

func TestFileTogglesStripReadmesAndDocs(t *testing.T) {
	spec := Normalize(Spec{
		UmbrellaChartName: "demo",
		Subcharts:         []Subchart{{Name: "api", Workload: "deployment"}},
		Rules: Rules{
			Ingress:                "alb",
			LinkedTemplates:        true,
			GenerateUmbrellaReadme: false,
			GenerateSubchartReadme: false,
			IncludeDocs:            false,
		},
	})
	ch, err := BuildChart(spec, testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	found := map[string]bool{}
	for _, f := range ch.Files {
		found[f.Name] = true
		if f.Name == "README.adoc" {
			t.Fatalf("umbrella README.adoc should be stripped")
		}
		if strings.HasPrefix(f.Name, "docs/") && f.Name != "docs/best-practices.adoc" {
			t.Fatalf("docs/ should be stripped, found %s", f.Name)
		}
	}
	// The finishing contract survives every toggle combination: the best-practices
	// doc plus the handoff and agent-instruction files that reference it.
	for _, keep := range []string{"docs/best-practices.adoc", "HANDOFF.md", "AGENTS.md", "CLAUDE.md"} {
		if !found[keep] {
			t.Fatalf("%s must be kept even with all file toggles off", keep)
		}
	}
	for _, d := range ch.Dependencies() {
		for _, f := range d.Files {
			if f.Name == "README.adoc" {
				t.Fatalf("subchart README.adoc should be stripped")
			}
		}
	}
}

func TestFileTogglesKeepByDefault(t *testing.T) {
	ch, err := BuildChart(basicSpec(), testdataTemplates)
	if err != nil {
		t.Fatalf("BuildChart: %v", err)
	}
	var hasReadme bool
	for _, f := range ch.Files {
		if f.Name == "README.adoc" {
			hasReadme = true
		}
	}
	if !hasReadme {
		t.Fatalf("umbrella README.adoc should be present by default")
	}
}
