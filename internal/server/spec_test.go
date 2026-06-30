// internal/server/spec_test.go
package server

import (
	"strings"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
)

func TestDecodeSpecMinimalBodyGetsDefaultRules(t *testing.T) {
	body := `{"umbrellaChartName":"demo","subcharts":[{"name":"api","workload":"deployment"}]}`
	spec, err := decodeSpec(strings.NewReader(body))
	if err != nil {
		t.Fatalf("decodeSpec: %v", err)
	}
	if spec.UmbrellaChartName != "demo" {
		t.Fatalf("name = %q, want demo", spec.UmbrellaChartName)
	}
	if len(spec.Subcharts) != 1 || spec.Subcharts[0].Workload != "deployment" {
		t.Fatalf("subcharts = %+v", spec.Subcharts)
	}
	want := engine.DefaultRules()
	if spec.Rules != want {
		t.Fatalf("rules = %+v, want defaults %+v", spec.Rules, want)
	}
}

func TestDecodeSpecHonorsExplicitFalseAndTrue(t *testing.T) {
	// linked_templates defaults true; explicit false must survive (not be re-defaulted).
	// common_annotations defaults false; explicit true must survive.
	body := `{"umbrellaChartName":"demo","subcharts":[{"name":"api","workload":"statefulset"}],
		"rules":{"linked_templates":false,"common_annotations":true,"ingress":"nginx"}}`
	spec, err := decodeSpec(strings.NewReader(body))
	if err != nil {
		t.Fatalf("decodeSpec: %v", err)
	}
	if spec.Rules.LinkedTemplates {
		t.Fatal("linked_templates: explicit false was lost")
	}
	if !spec.Rules.CommonAnnotations {
		t.Fatal("common_annotations: explicit true was lost")
	}
	if spec.Rules.Ingress != "nginx" {
		t.Fatalf("ingress = %q, want nginx", spec.Rules.Ingress)
	}
	// untouched booleans still take their defaults
	if !spec.Rules.IncludeDocs || !spec.Rules.GenerateUmbrellaReadme {
		t.Fatalf("omitted true-defaults were dropped: %+v", spec.Rules)
	}
}

func TestDecodeSpecNormalizesNames(t *testing.T) {
	body := `{"umbrellaChartName":"  Demo  ","subcharts":[{"name":"API","workload":"Deployment"}]}`
	spec, err := decodeSpec(strings.NewReader(body))
	if err != nil {
		t.Fatalf("decodeSpec: %v", err)
	}
	if spec.UmbrellaChartName != "demo" || spec.Subcharts[0].Name != "api" || spec.Subcharts[0].Workload != "deployment" {
		t.Fatalf("not normalized: %+v", spec)
	}
}
