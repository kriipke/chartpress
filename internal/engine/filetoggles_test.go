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
	for _, f := range ch.Files {
		if f.Name == "README.adoc" {
			t.Fatalf("umbrella README.adoc should be stripped")
		}
		if strings.HasPrefix(f.Name, "docs/") {
			t.Fatalf("docs/ should be stripped, found %s", f.Name)
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
