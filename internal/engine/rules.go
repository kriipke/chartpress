// internal/engine/rules.go
package engine

import (
	"regexp"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
)

// applyWorkload swaps the subchart's workload manifest. The subchart template ships
// templates/deployment.yaml = {{ include "umbrella-chart.deployment" . }}; for other
// workloads we drop it and emit templates/<workload>.yaml including the matching
// umbrella named template. Statefulsets additionally get the headless Service
// their serviceName references; daemonsets lose the HPA (HPAs cannot target them).
func applyWorkload(sub *chart.Chart, workload string) {
	switch workload {
	case "statefulset":
		sub.Templates = dropTemplate(sub.Templates, "templates/deployment.yaml")
		sub.Templates = append(sub.Templates,
			&chart.File{
				Name: "templates/statefulset.yaml",
				Data: []byte("{{ include \"umbrella-chart.statefulset\" . }}\n"),
			},
			&chart.File{
				Name: "templates/headless-service.yaml",
				Data: []byte("{{ include \"umbrella-chart.headlessService\" . }}\n"),
			})
	case "daemonset":
		sub.Templates = dropTemplate(sub.Templates, "templates/deployment.yaml")
		sub.Templates = dropTemplate(sub.Templates, "templates/hpa.yaml")
		sub.Templates = append(sub.Templates, &chart.File{
			Name: "templates/daemonset.yaml",
			Data: []byte("{{ include \"umbrella-chart.daemonset\" . }}\n"),
		})
	}
}

func dropTemplate(files []*chart.File, name string) []*chart.File {
	return dropFile(files, func(n string) bool { return n == name })
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
	idx := strings.Index(s, open)
	if idx < 0 {
		return data
	}
	insertAt := idx + len(open)
	merge := "\n{{- with .Values.global.commonAnnotations }}{{ toYaml . | trim }}\n{{- end }}"
	return []byte(s[:insertAt] + merge + s[insertAt:])
}

// applySharedSecretsUmbrella emits a shared-secrets Secret template into the
// umbrella chart; umbrellaValuesText seeds global.sharedSecrets.data in the
// generated values.yaml. The template tolerates a user deleting that block.
func applySharedSecretsUmbrella(ch *chart.Chart, spec Spec) {
	if !spec.Rules.SharedSecretsConfig {
		return
	}
	name := spec.UmbrellaChartName + "-shared-secrets"
	tmpl := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: " + name +
		"\ntype: Opaque\nstringData:\n{{- range $k, $v := dig \"sharedSecrets\" \"data\" (dict) (.Values.global | default dict) }}\n  {{ $k }}: {{ $v | quote }}\n{{- end }}\n"
	ch.Templates = append(ch.Templates, &chart.File{Name: "templates/shared-secrets.yaml", Data: []byte(tmpl)})
}

// applySharedNewrelicUmbrella emits a newrelic-config ConfigMap and newrelic-license
// Secret template into the umbrella chart; umbrellaValuesText seeds
// global.newrelic.licenseKey in the generated values.yaml.
func applySharedNewrelicUmbrella(ch *chart.Chart, spec Spec) {
	if !spec.Rules.SharedNewrelicConfig {
		return
	}
	cfg := spec.UmbrellaChartName + "-newrelic-config"
	lic := spec.UmbrellaChartName + "-newrelic-license"
	cm := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: " + cfg +
		"\ndata:\n  NEW_RELIC_ENABLED: \"true\"\n  NEW_RELIC_DISTRIBUTED_TRACING_ENABLED: \"true\"\n  NEW_RELIC_LABELS: \"app:" + spec.UmbrellaChartName + "\"\n"
	secret := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: " + lic +
		"\ntype: Opaque\nstringData:\n  NEW_RELIC_LICENSE_KEY: {{ dig \"newrelic\" \"licenseKey\" \"\" (.Values.global | default dict) | quote }}\n"
	ch.Templates = append(ch.Templates,
		&chart.File{Name: "templates/newrelic-config.yaml", Data: []byte(cm)},
		&chart.File{Name: "templates/newrelic-license.yaml", Data: []byte(secret)},
	)
}

// collectUmbrellaDefines returns the concatenated bodies of every umbrella .tpl
// named-template file (the files whose name ends in .tpl under templates/).
// Must be called on a PRISTINE (un-renamed) umbrella so the define names remain
// "umbrella-chart.*" rather than the renamed umbrella name.
func collectUmbrellaDefines(umbrella *chart.Chart) string {
	var b strings.Builder
	for _, t := range umbrella.Templates {
		if strings.HasSuffix(t.Name, ".tpl") {
			b.Write(t.Data)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// applyInlining appends the umbrella named-template defines into the subchart's
// _helpers.tpl so the subchart resolves its includes standalone.
func applyInlining(sub *chart.Chart, defines string) {
	for _, t := range sub.Templates {
		if t.Name == "templates/_helpers.tpl" {
			t.Data = append(append([]byte{}, t.Data...), append([]byte("\n"), []byte(defines)...)...)
			return
		}
	}
	sub.Templates = append(sub.Templates, &chart.File{
		Name: "templates/_helpers.tpl",
		Data: []byte(defines),
	})
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
