// internal/engine/patterns.go
//
// The pattern registry: the named workload shapes a subchart can be declared
// as (`pattern: worker`). The registry ships as embedded JSON because it is
// frozen data, not code — a shipped pattern's defaults never change (specs
// store intent and are re-resolved on every generate), so changing defaults
// requires a new pattern id or a major version. The web client carries a
// verbatim copy (web/src/app/patterns.json); patterns_contract_test.go keeps
// the two identical.
package engine

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed patterns.json
var patternsJSON []byte

// PatternDefaults are the trait defaults a pattern expands to. They are
// applied dependently (see ResolveTraits), so a default can never produce an
// invalid resolved spec.
type PatternDefaults struct {
	Workload  string `json:"workload"`
	Exposure  string `json:"exposure"`
	Ingress   bool   `json:"ingress"`
	Scaling   string `json:"scaling"`
	SharedEnv bool   `json:"shared_env"`
}

// Pattern is one registry entry: identity for the picker (label, example,
// badges), the trait defaults, and the handoff checklist the generated
// HANDOFF.md section is built from.
type Pattern struct {
	ID        string          `json:"id"`
	Label     string          `json:"label"`
	Example   string          `json:"example"`
	Badges    []string        `json:"badges"`
	Defaults  PatternDefaults `json:"defaults"`
	Checklist []string        `json:"checklist"`
}

// DefaultPattern is what an omitted pattern resolves to; its defaults
// reproduce the pre-pattern output byte-for-byte (back-compat).
const DefaultPattern = "api-microservice"

var (
	patterns  []Pattern
	patternBy map[string]Pattern
)

func init() {
	var reg struct {
		Patterns []Pattern `json:"patterns"`
	}
	if err := json.Unmarshal(patternsJSON, &reg); err != nil {
		panic(fmt.Sprintf("engine: embedded patterns.json is invalid: %v", err))
	}
	patterns = reg.Patterns
	patternBy = make(map[string]Pattern, len(patterns))
	for _, p := range patterns {
		patternBy[p.ID] = p
	}
	if _, ok := patternBy[DefaultPattern]; !ok {
		panic("engine: patterns.json is missing the default pattern " + DefaultPattern)
	}
}

// Patterns returns the registry in declaration order.
func Patterns() []Pattern {
	return patterns
}

// PatternByID looks up a pattern; ok is false for unknown ids.
func PatternByID(id string) (Pattern, bool) {
	p, ok := patternBy[id]
	return p, ok
}

// PatternIDs returns pattern ids in declaration order. When excludeCustom is
// true the human-only "custom" escape hatch is omitted — used for the
// text-to-config classification enum (the model must pick a real pattern).
func PatternIDs(excludeCustom bool) []string {
	ids := make([]string, 0, len(patterns))
	for _, p := range patterns {
		if excludeCustom && p.ID == "custom" {
			continue
		}
		ids = append(ids, p.ID)
	}
	return ids
}
