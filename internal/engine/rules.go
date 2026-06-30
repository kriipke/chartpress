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
	idx := strings.Index(s, open)
	if idx < 0 {
		return data
	}
	insertAt := idx + len(open)
	merge := "\n{{- with .Values.global.commonAnnotations }}\n{{ toYaml . }}\n{{- end }}"
	return []byte(s[:insertAt] + merge + s[insertAt:])
}

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
//
// Phase-1 limitation: the env/envFrom wiring renders ONLY on the "deployment"
// workload, because only deployment.tpl consumes .Values.env/.Values.envFrom;
// statefulset.tpl and daemonset.tpl have no env/envFrom blocks and Phase 1 may
// not edit them, so StatefulSet/DaemonSet subcharts do NOT receive these env
// vars/mounts. The umbrella-level Secret still renders regardless of workload.
// This is a known Phase-1 limitation, deferred to Phase 2.
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

// applySharedNewrelicUmbrella emits a newrelic-config ConfigMap and newrelic-license
// Secret template into the umbrella chart, and seeds global.newrelic so the secret
// template renders without erroring.
func applySharedNewrelicUmbrella(ch *chart.Chart, spec Spec) {
	if !spec.Rules.SharedNewrelicConfig {
		return
	}
	cfg := spec.UmbrellaChartName + "-newrelic-config"
	lic := spec.UmbrellaChartName + "-newrelic-license"
	cm := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: " + cfg +
		"\ndata:\n  NEW_RELIC_ENABLED: \"true\"\n  NEW_RELIC_DISTRIBUTED_TRACING_ENABLED: \"true\"\n  NEW_RELIC_LABELS: \"app:" + spec.UmbrellaChartName + "\"\n"
	secret := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: " + lic +
		"\ntype: Opaque\nstringData:\n  NEW_RELIC_LICENSE_KEY: {{ .Values.global.newrelic.licenseKey | default \"\" | quote }}\n"
	ch.Templates = append(ch.Templates,
		&chart.File{Name: "templates/newrelic-config.yaml", Data: []byte(cm)},
		&chart.File{Name: "templates/newrelic-license.yaml", Data: []byte(secret)},
	)
	global, _ := ch.Values["global"].(map[string]interface{})
	if global == nil {
		global = map[string]interface{}{}
		ch.Values["global"] = global
	}
	if _, ok := global["newrelic"]; !ok {
		global["newrelic"] = map[string]interface{}{"licenseKey": ""}
	}
}

// applySharedNewrelicSubchart wires the shared newrelic ConfigMap via envFrom and
// adds the per-subchart NEW_RELIC_LICENSE_KEY (from the shared secret) and
// NEW_RELIC_APP_NAME (= subchart name) env entries.
//
// Phase-1 limitation: the env/envFrom wiring renders ONLY on the "deployment"
// workload, because only deployment.tpl consumes .Values.env/.Values.envFrom;
// statefulset.tpl and daemonset.tpl have no env/envFrom blocks and Phase 1 may
// not edit them, so StatefulSet/DaemonSet subcharts do NOT receive these env
// vars/mounts. The umbrella-level ConfigMap and Secret still render regardless
// of workload. This is a known Phase-1 limitation, deferred to Phase 2.
func applySharedNewrelicSubchart(sub *chart.Chart, spec Spec) {
	if !spec.Rules.SharedNewrelicConfig {
		return
	}
	appendEnvFrom(sub, map[string]interface{}{
		"configMapRef": map[string]interface{}{"name": spec.UmbrellaChartName + "-newrelic-config"},
	})
	env, _ := sub.Values["env"].([]interface{})
	env = append(env,
		map[string]interface{}{
			"name": "NEW_RELIC_LICENSE_KEY",
			"valueFrom": map[string]interface{}{
				"secretKeyRef": map[string]interface{}{
					"name": spec.UmbrellaChartName + "-newrelic-license",
					"key":  "NEW_RELIC_LICENSE_KEY",
				},
			},
		},
		map[string]interface{}{"name": "NEW_RELIC_APP_NAME", "value": sub.Metadata.Name},
	)
	sub.Values["env"] = env
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
