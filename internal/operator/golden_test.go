// SPDX-License-Identifier: Apache-2.0
package operator

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kriipke/chartpress/internal/engine"
)

// TestGoldenChartFixture renders tests/chartpress.json through the same
// chartRenderer the operator uses and compares the result byte-for-byte
// against the checked-in tests/chart.zip. zipDir never writes wall-clock
// timestamps (see render.go), so the archive is reproducible across runs and
// machines; a mismatch means generation actually changed.
//
// Run with CHARTPRESS_UPDATE_GOLDEN=1 to regenerate tests/chart.zip after an
// intentional change to the templates or engine output.
func TestGoldenChartFixture(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(repoRoot, "tests", "chartpress.json")
	fixturePath := filepath.Join(repoRoot, "tests", "chart.zip")

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	var spec engine.Spec
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("decode %s: %v", configPath, err)
	}
	spec = engine.Normalize(spec)

	r := &chartRenderer{templatesDir: filepath.Join(repoRoot, "templates")}
	got, err := r.RenderZip(spec)
	if err != nil {
		t.Fatalf("RenderZip: %v", err)
	}

	if os.Getenv("CHARTPRESS_UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(fixturePath, got, 0o644); err != nil {
			t.Fatalf("write golden fixture: %v", err)
		}
		t.Logf("regenerated %s (%d bytes)", fixturePath, len(got))
		return
	}

	want, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read golden fixture %s: %v", fixturePath, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("tests/chart.zip is stale (generation changed): got %d bytes, want %d bytes; "+
			"run `make update-chart-fixture` and review the diff before committing", len(got), len(want))
	}
}
