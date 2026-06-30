package operator

import (
	"archive/zip"
	"bytes"
	"path/filepath"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
)

func zipEntries(t *testing.T, b []byte) map[string]bool {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	return names
}

func TestChartRendererProducesValidChartZip(t *testing.T) {
	spec := engine.Normalize(engine.Spec{
		UmbrellaChartName: "demo-platform",
		Subcharts:         []engine.Subchart{{Name: "api", Workload: "deployment"}},
		Rules:             engine.DefaultRules(),
	})
	r := &chartRenderer{templatesDir: filepath.Join("..", "..", "templates")}

	b, err := r.RenderZip(spec)
	if err != nil {
		t.Fatalf("RenderZip: %v", err)
	}
	names := zipEntries(t, b)
	for _, want := range []string{
		"demo-platform/Chart.yaml",
		"demo-platform/charts/api/Chart.yaml",
	} {
		if !names[want] {
			t.Fatalf("zip missing %q; entries = %v", want, names)
		}
	}
}

func TestDecodeSpecFromUnstructured(t *testing.T) {
	obj := crObj("shop", 1) // helper defined in reconcile_test.go (Task 5)
	spec, err := decodeSpec(obj)
	if err != nil {
		t.Fatalf("decodeSpec: %v", err)
	}
	if spec.UmbrellaChartName != "shop" || len(spec.Subcharts) != 1 ||
		spec.Subcharts[0].Workload != "deployment" || spec.Rules.Ingress != "alb" {
		t.Fatalf("decoded spec = %+v", spec)
	}
}
