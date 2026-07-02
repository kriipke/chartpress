// internal/engine/patterns_contract_test.go
//
// The web client renders the pattern picker from its own copy of the registry
// (web/src/app/patterns.json) because the web Docker build context is ./web/
// and go:embed cannot cross packages — two files, one contract: this test
// fails CI on any drift.
package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestPatternsJSONMatchesWebCopy(t *testing.T) {
	webCopy, err := os.ReadFile("../../web/src/app/patterns.json")
	if err != nil {
		t.Fatalf("read web copy (did you add web/src/app/patterns.json?): %v", err)
	}
	// Compare canonicalized JSON so formatting-only differences don't fail;
	// any semantic difference (defaults, checklist wording, ids) does.
	var a, b interface{}
	if err := json.Unmarshal(patternsJSON, &a); err != nil {
		t.Fatalf("engine patterns.json: %v", err)
	}
	if err := json.Unmarshal(webCopy, &b); err != nil {
		t.Fatalf("web patterns.json: %v", err)
	}
	ca, _ := json.Marshal(a)
	cb, _ := json.Marshal(b)
	if !bytes.Equal(ca, cb) {
		t.Fatalf("web/src/app/patterns.json has drifted from internal/engine/patterns.json — copy the engine file verbatim")
	}
}

func TestPatternRegistryShape(t *testing.T) {
	if len(Patterns()) != 13 { // twelve patterns + custom
		t.Fatalf("registry has %d entries, want 13", len(Patterns()))
	}
	for _, p := range Patterns() {
		if p.ID == "" || p.Label == "" || len(p.Checklist) == 0 {
			t.Fatalf("pattern %+v is missing id/label/checklist", p)
		}
		if !contains(AllowedWorkloads, p.Defaults.Workload) {
			t.Fatalf("pattern %s has invalid default workload %q", p.ID, p.Defaults.Workload)
		}
		if !contains(AllowedExposures, p.Defaults.Exposure) {
			t.Fatalf("pattern %s has invalid default exposure %q", p.ID, p.Defaults.Exposure)
		}
		if !contains(AllowedScalings, p.Defaults.Scaling) {
			t.Fatalf("pattern %s has invalid default scaling %q", p.ID, p.Defaults.Scaling)
		}
	}
}
