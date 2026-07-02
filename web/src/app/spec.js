// chartpress spec helpers: kebab-case validation, the workload/ingress/trait
// enums, the locked rule defaults, the grouped rules reference, and the
// client-side mirror of engine.ResolveTraits (dependent defaulting) so the
// preview, override chips, and warnings match the server exactly.
//
// patterns.json is a verbatim copy of internal/engine/patterns.json — the
// engine's patterns_contract_test.go fails CI if the two drift.
import patternsData from "./patterns.json";

const KEBAB = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;
export const isKebab = (s) => KEBAB.test(s);

export const WORKLOADS = ["deployment", "statefulset", "daemonset"];
export const DEFAULT_PATTERN = "api-microservice";

export const PATTERNS = patternsData.patterns;
export const PATTERN_BY_ID = Object.fromEntries(PATTERNS.map((p) => [p.id, p]));

export const EXPOSURE_OPTIONS = [
  { value: "http", label: "http — serves HTTP" },
  { value: "grpc", label: "grpc — serves gRPC" },
  { value: "tcp", label: "tcp — other TCP server" },
  { value: "none", label: "none — pulls work, serves nothing" },
];

export const SCALING_OPTIONS = [
  { value: "auto", label: "auto — HPA available" },
  { value: "fixed", label: "fixed — pinned count, no HPA" },
  { value: "singleton", label: "singleton — exactly one" },
];

export const INGRESS_OPTIONS = [
  { value: "alb", label: "alb — AWS ALB" },
  { value: "nginx", label: "nginx" },
  { value: "traefik", label: "traefik" },
  { value: "istio", label: "istio — Gateway/VirtualService" },
  { value: "gce", label: "gce — GCE" },
  { value: "none", label: "none — disable ingress" },
];

// engine.DefaultRules(): ingress=alb plus the three "generate" toggles and
// linked_templates on; everything else off. generate_handoff is nil=true on
// the server; the form models it as an explicit true.
export const DEFAULT_RULES = {
  ingress: "alb",
  linked_templates: true,
  generate_umbrella_readme: true,
  generate_subchart_readme: true,
  include_docs: true,
  generate_handoff: true,
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
      { key: "include_docs", label: "Docs directory", desc: "Include the topic docs (probes, HPA, ingress, …). The best-practices doc and agent files are always kept." },
      { key: "generate_handoff", label: "Handoff file", desc: "Generate HANDOFF.md — the per-subchart finishing checklist an LLM (or you) resolves against the app code." },
    ],
  },
  {
    title: "Chart behavior",
    rules: [
      { key: "linked_templates", label: "Linked templates", desc: "Subcharts share common template logic (DRY). Turn off to make each fully self-contained." },
      { key: "common_annotations", label: "Common annotations", desc: "Add shared annotations (part-of, managed-by) to every resource." },
      { key: "resource_names_match_chart_name", label: "Names match chart name", desc: "Name resources exactly after the chart, without the release-name prefix." },
      { key: "shared_secrets_config", label: "Shared secret", desc: "Create one shared Secret and inject it into subcharts that opt in (shared env)." },
      { key: "shared_newrelic_config", label: "Shared New Relic config", desc: "Create shared New Relic config + license and wire it into subcharts that opt in." },
    ],
  },
];

// The trait keys a subchart may explicitly carry (everything else on a row is
// name/description/UI state).
export const TRAIT_KEYS = ["workload", "exposure", "port", "ingress", "scaling", "shared_env"];

// resolveTraits mirrors engine.ResolveTraits: pattern defaults + explicit
// overrides, applied DEPENDENTLY — a default can never produce an invalid
// combination; the form prevents invalid explicit keys, so unlike the server
// this mirror clamps instead of erroring.
export function resolveTraits(sc, rules) {
  const p = PATTERN_BY_ID[sc.pattern || DEFAULT_PATTERN] || PATTERN_BY_ID[DEFAULT_PATTERN];
  const d = p.defaults;
  const workload = sc.workload || d.workload;
  const exposure = sc.exposure || d.exposure;

  let port = 0;
  if (exposure !== "none") {
    port = sc.port || (exposure === "grpc" ? 50051 : 8080);
  }

  const ingressable = (exposure === "http" || exposure === "grpc") && rules.ingress !== "none";
  const ingress = sc.ingress != null ? (sc.ingress && ingressable) : (d.ingress && ingressable);

  let scaling = sc.scaling || d.scaling;
  if (workload === "daemonset") scaling = "fixed";

  const shared_env = sc.shared_env != null ? sc.shared_env : d.shared_env;

  return { pattern: p.id, workload, exposure, port, ingress, scaling, shared_env };
}

// traitOverrides lists the explicit keys whose value differs from what the
// pattern would have resolved without them — the chips on a collapsed row.
export function traitOverrides(sc, rules) {
  const out = [];
  for (const key of TRAIT_KEYS) {
    if (sc[key] == null || sc[key] === "" || sc[key] === 0) continue;
    const without = { ...sc };
    delete without[key];
    if (resolveTraits(without, rules)[key] !== resolveTraits(sc, rules)[key]) out.push(key);
  }
  return out;
}

// specWarnings mirrors engine.Warnings: the non-blocking spec-level lint.
export function specWarnings(spec) {
  const rules = spec.rules || DEFAULT_RULES;
  const named = (spec.subcharts || []).filter((s) => s.name && s.name.trim());
  const resolved = named.map((s) => resolveTraits(s, rules));
  const warns = [];
  const gateways = named.filter((_, i) => resolved[i].pattern === "edge-gateway").map((s) => s.name);
  const pub = named.filter((_, i) => resolved[i].ingress).map((s) => s.name);
  named.forEach((s, i) => {
    if (resolved[i].pattern === "admin-dashboard" && resolved[i].ingress) {
      warns.push(`"${s.name}" (admin-dashboard) is reachable via ingress — protect this route (auth annotations / private ingress class) before going live.`);
    }
  });
  if (gateways.length > 0 && pub.length > 1) {
    warns.push(`Edge gateway ${gateways.join(", ")} is meant to be the platform's only public entry, but ${pub.length} subcharts resolve ingress: true (${pub.join(", ")}) — intended?`);
  }
  return warns;
}

// cleanSubchart strips UI state and unset trait keys, returning the INTENT
// the spec stores: pattern + only the explicitly chosen trait keys.
export function cleanSubchart(s) {
  const out = { name: (s.name || "").trim(), description: s.description || "" };
  if (s.pattern) out.pattern = s.pattern;
  for (const key of TRAIT_KEYS) {
    if (s[key] != null && s[key] !== "" && s[key] !== 0) out[key] = s[key];
  }
  return out;
}

// Normalize a spec returned by the server (or built locally) into the shape
// the RichForm seeds from: a non-empty subchart list carrying pattern + any
// explicit trait keys, and a full rules object.
export function normalizeSpec(spec) {
  const s = spec || {};
  const subs = Array.isArray(s.subcharts) && s.subcharts.length
    ? s.subcharts.map((x) => {
        const row = { name: x.name || "", description: x.description || "", pattern: x.pattern || DEFAULT_PATTERN };
        for (const key of TRAIT_KEYS) {
          if (x[key] != null && x[key] !== "" && x[key] !== 0) row[key] = x[key];
        }
        return row;
      })
    : [{ name: "", description: "", pattern: DEFAULT_PATTERN }];
  return {
    umbrellaChartName: s.umbrellaChartName || "",
    description: s.description || "",
    subcharts: subs,
    dependencies: Array.isArray(s.dependencies) ? s.dependencies.filter(Boolean) : [],
    rules: { ...DEFAULT_RULES, ...(s.rules || {}) },
  };
}

// The infrastructure-dependency keys the registry knows (for the form's
// suggestions and to label known vs. TODO-stub dependencies). Kept in sync
// with internal/engine/dependencies.json by convention — small enough that a
// contract test isn't warranted, but update both when the registry grows.
export const KNOWN_DEPENDENCIES = [
  "postgresql", "mysql", "redis", "valkey", "kafka", "rabbitmq", "mongodb", "elasticsearch",
];
