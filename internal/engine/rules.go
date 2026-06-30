// internal/engine/rules.go
package engine

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
)

// applyWorkload swaps the subchart's workload manifest. The subchart template ships
// templates/deployment.yaml = {{ include "umbrella-chart.deployment" . }}; for other
// workloads we drop it and emit templates/<workload>.yaml including the matching
// umbrella named template.
func applyWorkload(sub *chart.Chart, workload string) {
	// Only statefulset/daemonset swap the workload template; deployment (and any
	// other value, which Validate rejects upstream) is a no-op.
	if workload != "statefulset" && workload != "daemonset" {
		return
	}
	sub.Templates = dropTemplate(sub.Templates, "templates/deployment.yaml")
	sub.Templates = append(sub.Templates, &chart.File{
		Name: fmt.Sprintf("templates/%s.yaml", workload),
		Data: []byte(fmt.Sprintf("{{ include \"umbrella-chart.%s\" . }}\n", workload)),
	})
}

func dropTemplate(files []*chart.File, name string) []*chart.File {
	out := files[:0]
	for _, f := range files {
		if f.Name != name {
			out = append(out, f)
		}
	}
	return out
}

func dropFile(files []*chart.File, pred func(name string) bool) []*chart.File {
	out := files[:0]
	for _, f := range files {
		if !pred(f.Name) {
			out = append(out, f)
		}
	}
	return out
}

func applyUmbrellaFileToggles(ch *chart.Chart, r Rules) {
	if !r.GenerateUmbrellaReadme {
		ch.Files = dropFile(ch.Files, func(n string) bool { return n == "README.adoc" })
	}
	if !r.IncludeDocs {
		ch.Files = dropFile(ch.Files, func(n string) bool { return strings.HasPrefix(n, "docs/") })
	}
}

func applySubchartFileToggles(sub *chart.Chart, r Rules) {
	if !r.GenerateSubchartReadme {
		sub.Files = dropFile(sub.Files, func(n string) bool { return n == "README.adoc" })
	}
}
