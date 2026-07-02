package crd

import (
	"os"
	"reflect"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
	sigsyaml "sigs.k8s.io/yaml"
)

const crdPath = "../../crds/crd-helmchart.yaml"

// chartCRDPath is the CRD Helm actually installs (chart/crds/). It must not
// drift from the canonical CRD above, or the operator/backend path silently
// prunes spec fields (pattern, dependencies, exposure, …) that the engine reads.
const chartCRDPath = "../../chart/crds/chartpressconfigs.yaml"

func readYAML(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]interface{}
	if err := sigsyaml.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return m
}

func nested(t *testing.T, m map[string]interface{}, keys ...string) interface{} {
	t.Helper()
	var cur interface{} = m
	for _, k := range keys {
		mp, ok := cur.(map[string]interface{})
		if !ok {
			t.Fatalf("path %v: %q is not a map", keys, k)
		}
		cur = mp[k]
	}
	return cur
}

func TestCRDGroupAndNameMigrated(t *testing.T) {
	m := readYAML(t, crdPath)
	if g := nested(t, m, "spec", "group"); g != "chartpress.dev" {
		t.Fatalf("spec.group = %v, want chartpress.dev", g)
	}
	if n := nested(t, m, "metadata", "name"); n != "chartpressconfigs.chartpress.dev" {
		t.Fatalf("metadata.name = %v, want chartpressconfigs.chartpress.dev", n)
	}
}

func TestCRDSchemaShape(t *testing.T) {
	m := readYAML(t, crdPath)
	versions := nested(t, m, "spec", "versions").([]interface{})
	var schema map[string]interface{}
	for _, v := range versions {
		vm := v.(map[string]interface{})
		if vm["name"] == "v1alpha1" {
			schema = vm["schema"].(map[string]interface{})
		}
	}
	if schema == nil {
		t.Fatal("CRD has no v1alpha1 version")
	}
	specProps := nested(t, schema, "openAPIV3Schema", "properties", "spec", "properties").(map[string]interface{})

	// spec.description present
	if _, ok := specProps["description"]; !ok {
		t.Fatal("spec.description property missing")
	}
	// subcharts[].description present, workload enum is exactly the 3 allowed
	subItems := nested(t, specProps, "subcharts", "items", "properties").(map[string]interface{})
	if _, ok := subItems["description"]; !ok {
		t.Fatal("subcharts[].description property missing")
	}
	wEnum := toStrings(t, nested(t, subItems, "workload", "enum"))
	assertSetEqual(t, wEnum, engine.AllowedWorkloads, "workload enum")

	// rules.ingress is a single string enum; possible_ingresses is gone
	rulesProps := nested(t, specProps, "rules", "properties").(map[string]interface{})
	if _, present := rulesProps["possible_ingresses"]; present {
		t.Fatal("rules.possible_ingresses must be removed")
	}
	if typ := nested(t, rulesProps, "ingress", "type"); typ != "string" {
		t.Fatalf("rules.ingress type = %v, want string", typ)
	}
	iEnum := toStrings(t, nested(t, rulesProps, "ingress", "enum"))
	assertSetEqual(t, iEnum, engine.AllowedIngress, "ingress enum")
}

func TestExampleCRsAreValidSpecs(t *testing.T) {
	for _, path := range []string{"../../crds/helmchart-iot.yaml", "../../crds/helmchart-ml.yaml"} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var cr struct {
			APIVersion string      `json:"apiVersion"`
			Spec       engine.Spec `json:"spec"`
		}
		if err := sigsyaml.Unmarshal(b, &cr); err != nil {
			t.Fatalf("unmarshal %s: %v", path, err)
		}
		if cr.APIVersion != "chartpress.dev/v1alpha1" {
			t.Fatalf("%s apiVersion = %q, want chartpress.dev/v1alpha1", path, cr.APIVersion)
		}
		if err := engine.Validate(engine.Normalize(cr.Spec)); err != nil {
			t.Fatalf("%s is not a valid spec: %v", path, err)
		}
	}
}

// TestChartCRDMatchesCanonical guards against the two CRD copies drifting. The
// canonical schema lives in crds/crd-helmchart.yaml; the chart ships its own
// copy under chart/crds/ because Helm installs CRDs only from that directory.
// Kubernetes prunes unknown fields on structural schemas, so a stale chart CRD
// would drop the rich spec fields the engine depends on without any error.
func TestChartCRDMatchesCanonical(t *testing.T) {
	canonical := readYAML(t, crdPath)
	shipped := readYAML(t, chartCRDPath)
	if !reflect.DeepEqual(canonical["spec"], shipped["spec"]) {
		t.Fatalf("chart/crds CRD schema has drifted from crds/crd-helmchart.yaml; "+
			"regenerate %s from %s so the installed CRD carries every spec field", chartCRDPath, crdPath)
	}
}

func toStrings(t *testing.T, v interface{}) []string {
	t.Helper()
	raw, ok := v.([]interface{})
	if !ok {
		t.Fatalf("enum is not a list: %T", v)
	}
	out := make([]string, len(raw))
	for i, e := range raw {
		out[i] = e.(string)
	}
	return out
}

func assertSetEqual(t *testing.T, got, want []string, label string) {
	t.Helper()
	gm := map[string]bool{}
	for _, g := range got {
		gm[g] = true
	}
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
	for _, w := range want {
		if !gm[w] {
			t.Fatalf("%s missing %q (got %v)", label, w, got)
		}
	}
}
