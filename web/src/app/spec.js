// chartpress spec helpers: kebab-case validation, the workload/ingress enums,
// the locked rule defaults, and the grouped rules reference (labels + tooltips).
// These mirror internal/engine (AllowedWorkloads, AllowedIngress, DefaultRules,
// and the name regex) so client-side validation matches the server.

const KEBAB = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;
export const isKebab = (s) => KEBAB.test(s);

export const WORKLOADS = ["deployment", "statefulset", "daemonset"];

export const INGRESS_OPTIONS = [
  { value: "alb", label: "alb — AWS ALB" },
  { value: "nginx", label: "nginx" },
  { value: "traefik", label: "traefik" },
  { value: "istio", label: "istio — Gateway/VirtualService" },
  { value: "gce", label: "gce — GCE" },
  { value: "none", label: "none — disable ingress" },
];

// engine.DefaultRules(): ingress=alb plus the three "generate" toggles and
// linked_templates on; everything else off.
export const DEFAULT_RULES = {
  ingress: "alb",
  linked_templates: true,
  generate_umbrella_readme: true,
  generate_subchart_readme: true,
  include_docs: true,
  common_annotations: false,
  resource_names_match_chart_name: false,
  shared_secrets_config: false,
  shared_newrelic_config: false,
};

// Grouped checkbox rules (ingress is a dropdown, handled separately).
export const RULE_GROUPS = [
  {
    title: "Output files",
    rules: [
      { key: "generate_umbrella_readme", label: "Umbrella README", desc: "Include a README for the umbrella chart." },
      { key: "generate_subchart_readme", label: "Subchart READMEs", desc: "Include a README inside each subchart." },
      { key: "include_docs", label: "Docs directory", desc: "Include the docs/ directory." },
    ],
  },
  {
    title: "Chart behavior",
    rules: [
      { key: "linked_templates", label: "Linked templates", desc: "Subcharts share common template logic (DRY). Turn off to make each fully self-contained." },
      { key: "common_annotations", label: "Common annotations", desc: "Add shared annotations (part-of, managed-by) to every resource." },
      { key: "resource_names_match_chart_name", label: "Names match chart name", desc: "Name resources exactly after the chart, without the release-name prefix." },
      { key: "shared_secrets_config", label: "Shared secret", desc: "Create one shared Secret and inject it into every subchart." },
      { key: "shared_newrelic_config", label: "Shared New Relic config", desc: "Create shared New Relic config + license and wire it into every subchart." },
    ],
  },
];

// Normalize a spec returned by the server (or built locally) into the shape the
// RichForm seeds from: a non-empty subchart list and a full rules object.
export function normalizeSpec(spec) {
  const s = spec || {};
  const subs = Array.isArray(s.subcharts) && s.subcharts.length
    ? s.subcharts.map((x) => ({ name: x.name || "", workload: x.workload || "deployment", description: x.description || "" }))
    : [{ name: "", workload: "deployment", description: "" }];
  return {
    umbrellaChartName: s.umbrellaChartName || "",
    description: s.description || "",
    subcharts: subs,
    rules: { ...DEFAULT_RULES, ...(s.rules || {}) },
  };
}
