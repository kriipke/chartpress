// internal/engine/rules.go
package engine

import (
	"fmt"
	"regexp"
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

// applyCommonAnnotationsUmbrella seeds global.commonAnnotations in the umbrella
// chart's values so that Helm propagates them to every subchart at render time.
func applyCommonAnnotationsUmbrella(ch *chart.Chart, spec Spec) {
	if !spec.Rules.CommonAnnotations {
		return
	}
	global, _ := ch.Values["global"].(map[string]interface{})
	if global == nil {
		global = map[string]interface{}{}
		ch.Values["global"] = global
	}
	global["commonAnnotations"] = map[string]interface{}{
		"app.kubernetes.io/part-of": spec.UmbrellaChartName,
		"chartpress.dev/managed":    "true",
	}
}

// applyCommonAnnotationsSubchart appends the global.commonAnnotations merge block
// into the subchart's <name>.annotations named template. Must be called AFTER
// replacePlaceholders so the define is already named "<sub.Metadata.Name>.annotations".
func applyCommonAnnotationsSubchart(sub *chart.Chart, spec Spec) {
	if !spec.Rules.CommonAnnotations {
		return
	}
	for _, t := range sub.Templates {
		if t.Name == "templates/_helpers.tpl" {
			t.Data = appendToAnnotationsDefine(t.Data, sub.Metadata.Name)
		}
	}
}

func appendToAnnotationsDefine(data []byte, chartName string) []byte {
	open := "{{- define \"" + chartName + ".annotations\" -}}"
	s := string(data)
	idx := indexOf(s, open)
	if idx < 0 {
		return data
	}
	insertAt := idx + len(open)
	merge := "\n{{- with .Values.global.commonAnnotations }}\n{{ toYaml . }}\n{{- end }}"
	return []byte(s[:insertAt] + merge + s[insertAt:])
}

func indexOf(s, sub string) int { return strings.Index(s, sub) }

// applySharedSecretsUmbrella emits a shared-secrets Secret template into the
// umbrella chart and seeds global.sharedSecrets.data so the template renders.
func applySharedSecretsUmbrella(ch *chart.Chart, spec Spec) {
	if !spec.Rules.SharedSecretsConfig {
		return
	}
	name := spec.UmbrellaChartName + "-shared-secrets"
	tmpl := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: " + name +
		"\ntype: Opaque\nstringData:\n{{- range $k, $v := .Values.global.sharedSecrets.data }}\n  {{ $k }}: {{ $v | quote }}\n{{- end }}\n"
	ch.Templates = append(ch.Templates, &chart.File{Name: "templates/shared-secrets.yaml", Data: []byte(tmpl)})

	global, _ := ch.Values["global"].(map[string]interface{})
	if global == nil {
		global = map[string]interface{}{}
		ch.Values["global"] = global
	}
	if _, exists := global["sharedSecrets"]; !exists {
		global["sharedSecrets"] = map[string]interface{}{"data": map[string]interface{}{}}
	}
}

// applySharedSecretsSubchart appends an envFrom secretRef entry to the subchart
// values so the deployment template mounts the shared-secrets Secret.
func applySharedSecretsSubchart(sub *chart.Chart, spec Spec) {
	if !spec.Rules.SharedSecretsConfig {
		return
	}
	appendEnvFrom(sub, map[string]interface{}{
		"secretRef": map[string]interface{}{"name": spec.UmbrellaChartName + "-shared-secrets"},
	})
}

// appendEnvFrom adds an entry to the subchart's .Values.envFrom (the deployment
// template already ranges over it).
func appendEnvFrom(sub *chart.Chart, entry map[string]interface{}) {
	cur, _ := sub.Values["envFrom"].([]interface{})
	sub.Values["envFrom"] = append(cur, entry)
}

// applyResourceNaming rewrites the subchart fullname helper to emit just the chart
// name. The helper define line looks like:
//
//	{{- define "api.fullname" -}}
//	{{- template "umbrella-chart.fullname" . }}-{{ .Chart.Name }}
//	{{- end }}
func applyResourceNaming(sub *chart.Chart, match bool) {
	if !match {
		return
	}
	def := regexp.MustCompile(`(?s)(\{\{-?\s*define\s+"` + regexp.QuoteMeta(sub.Metadata.Name) + `\.fullname"\s*-?\}\}).*?(\{\{-?\s*end\s*-?\}\})`)
	for _, t := range sub.Templates {
		if t.Name == "templates/_helpers.tpl" {
			t.Data = def.ReplaceAll(t.Data, []byte("${1}\n{{ .Chart.Name }}\n${2}"))
		}
	}
}
