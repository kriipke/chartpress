// internal/engine/handoff.go
//
// The generated HANDOFF.md: one sectioned file at the umbrella root — a
// protocol header, platform-level items (values layering, warnings), then one
// section per subchart keyed by its pattern's checklist. It is addressed to
// the coding agent (or developer) finishing the chart against the application
// code. Completion is incremental: delete a subchart's section when that
// component is done, delete the file when the chart is done. Gated by
// rules.generate_handoff (default true).
package engine

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
)

// applyHandoff generates HANDOFF.md into the umbrella chart's files. Must run
// AFTER renameChart (it writes real names, not placeholders) and after the
// file toggles (it is gated by its own rule, not by include_docs).
func applyHandoff(ch *chart.Chart, spec Spec, resolved []ResolvedTraits) {
	if !spec.Rules.HandoffEnabled() {
		return
	}
	ch.Files = append(ch.Files, &chart.File{
		Name: "HANDOFF.md",
		Data: []byte(handoffText(spec, resolved)),
	})
}

func handoffText(spec Spec, resolved []ResolvedTraits) string {
	var b strings.Builder

	// --- protocol header ---
	b.WriteString("# HANDOFF — finish this chart against the application code\n\n")
	b.WriteString("This file is addressed to the coding agent (or developer) who has this chart\n")
	b.WriteString("**and** the application source code side by side. chartpress generated the\n")
	b.WriteString("first half of this chart; your job is the second half: resolve every\n")
	b.WriteString("app-specific placeholder against the real code.\n\n")
	b.WriteString("Completion is incremental: **delete a subchart's section below when that\n")
	b.WriteString("component is finished; delete this file when the chart is done** — its\n")
	b.WriteString("absence is the \"chart is finished\" signal.\n\n")
	b.WriteString("## Ground rules\n\n")
	b.WriteString("1. **The best practices are load-bearing.** They are documented in\n")
	b.WriteString("   [docs/best-practices.adoc](docs/best-practices.adoc) — read it before\n")
	b.WriteString("   editing. Placeholders are **replaced, never deleted**; disabled practices\n")
	b.WriteString("   (PDB, NetworkPolicy, securityContext) are **enabled, never removed**. If a\n")
	b.WriteString("   practice truly cannot apply, record why in that subchart's README.\n")
	b.WriteString("2. **Work from evidence, not convention.** Every answer should come from the\n")
	b.WriteString("   application code, Dockerfile, or CI config. If the code doesn't answer a\n")
	b.WriteString("   question (e.g. no health endpoint exists), the fix may belong in the app.\n")
	b.WriteString("3. **Don't restructure.** Keep selector labels, helper names, the values\n")
	b.WriteString("   schema, and the file layout as generated. Never change `selectorLabels`;\n")
	b.WriteString("   never set `replicas` alongside an HPA.\n")
	b.WriteString("4. **Validate every change**: `make template`, `make lint`, `make test`\n")
	b.WriteString("   (render + kubeconform) from the chart root.\n")
	b.WriteString("5. Don't add subcharts for databases, caches, or brokers — those belong in\n")
	b.WriteString("   `Chart.yaml` `dependencies:` as upstream charts. Sidecars are containers\n")
	b.WriteString("   in the owning pod, never new subcharts.\n\n")

	// --- platform section ---
	b.WriteString("## Platform\n\n")
	b.WriteString("- Values layering: a subchart's own `values.yaml` holds its defaults; the\n")
	b.WriteString("  umbrella `values.yaml` overrides them under the subchart's name; shared\n")
	b.WriteString("  settings live under `global:`.\n")
	b.WriteString("- [ ] Set `global.repository` and pin every image tag\n")
	b.WriteString("  (`grep -rn \"TODO(chartpress)\" .` is the authoritative list).\n")
	if spec.Rules.SharedSecretsConfig {
		b.WriteString("- [ ] Fill `global.sharedSecrets.data` (or wire your secrets operator into\n")
		b.WriteString("  the envFrom seam) — never commit real secrets to values.\n")
	}
	if spec.Rules.SharedNewrelicConfig {
		b.WriteString("- [ ] Set `global.newrelic.licenseKey` from a secure source.\n")
	}
	if items := dependencyHandoffItems(spec); len(items) > 0 {
		for _, item := range items {
			b.WriteString("- [ ] " + item + "\n")
		}
		b.WriteString("- [ ] Run `make deps` (helm dependency build) once the dependency\n")
		b.WriteString("  repositories/versions above are resolved — Helm won't render the chart\n")
		b.WriteString("  until its dependencies are fetched.\n")
	}
	for _, w := range Warnings(spec) {
		b.WriteString("- [ ] ⚠ " + w + "\n")
	}
	b.WriteString("\n")

	// --- per-subchart sections ---
	for i, sc := range spec.Subcharts {
		rt := resolved[i]
		p, _ := PatternByID(rt.Pattern)
		fmt.Fprintf(&b, "## charts/%s — %s\n\n", sc.Name, p.Label)
		fmt.Fprintf(&b, "Generated as `%s` (%s).%s\n\n", rt.Pattern, traitSummary(rt), overrideNote(sc, p))
		for _, item := range p.Checklist {
			b.WriteString("- [ ] " + item + "\n")
		}
		for _, item := range overrideChecklist(sc, rt) {
			b.WriteString("- [ ] " + item + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Done means\n\n")
	b.WriteString("- `grep -rn \"TODO(chartpress)\" .` returns nothing.\n")
	b.WriteString("- `make test` passes.\n")
	b.WriteString("- Every section above is resolved and deleted, and this file with them.\n")
	return b.String()
}

func traitSummary(rt ResolvedTraits) string {
	parts := []string{rt.Workload}
	if rt.Exposure == "none" {
		parts = append(parts, "no service")
	} else {
		parts = append(parts, fmt.Sprintf("%s :%d", rt.Exposure, rt.Port))
	}
	if rt.Ingress {
		parts = append(parts, "ingress")
	}
	parts = append(parts, rt.Scaling)
	if !rt.SharedEnv {
		parts = append(parts, "no shared env")
	}
	return strings.Join(parts, " · ")
}

// overrideNote flags sections whose subchart deviates from the pattern's
// defaults, so the reader knows the deltas were chosen, not generated.
func overrideNote(sc Subchart, p Pattern) string {
	var over []string
	if sc.Workload != "" && sc.Workload != p.Defaults.Workload {
		over = append(over, "workload")
	}
	if sc.Exposure != "" && sc.Exposure != p.Defaults.Exposure {
		over = append(over, "exposure")
	}
	if sc.Ingress != nil && *sc.Ingress != p.Defaults.Ingress {
		over = append(over, "ingress")
	}
	if sc.Scaling != "" && sc.Scaling != p.Defaults.Scaling {
		over = append(over, "scaling")
	}
	if sc.SharedEnv != nil && *sc.SharedEnv != p.Defaults.SharedEnv {
		over = append(over, "shared_env")
	}
	if len(over) == 0 {
		return ""
	}
	return " Explicit overrides: " + strings.Join(over, ", ") + "."
}

// overrideChecklist adds items that only apply because of a trait override.
func overrideChecklist(sc Subchart, rt ResolvedTraits) []string {
	var items []string
	if rt.Pattern == "grpc-service" && rt.Ingress {
		items = append(items,
			"External gRPC: confirm the ingress controller speaks gRPC end-to-end (TLS/h2) and the client supports the controller's gRPC framing.")
	}
	if rt.Scaling == "singleton" && rt.Pattern != "scheduler" {
		items = append(items,
			"Singleton override: determine what happens if two instances ever overlap; Recreate covers rollouts, not node partitions — add leader election if overlap is unsafe.")
	}
	return items
}
