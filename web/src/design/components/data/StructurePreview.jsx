import React from "react";

/**
 * chartpress StructurePreview — the dark terminal-style read-only preview of the
 * chart tree that would be generated. Renders box-drawing characters in mono.
 * Pass a precomputed `tree` string, or `umbrellaName` + `subcharts` + `rules`
 * (and optionally `traits`, an array of resolved traits parallel to subcharts:
 * {pattern, workload, exposure, ingress, scaling}) to have it build the tree
 * itself — with traits, each subchart's manifest list reflects the tailoring
 * (a worker shows no service.yaml/ingress.yaml, a singleton no hpa.yaml).
 */
export function StructurePreview({
  tree,
  umbrellaName = "umbrella-chart",
  subcharts = [],
  rules = {},
  traits = null,
  dependencies = [],
  label = "Structure preview",
  style = {},
}) {
  const text = tree != null ? tree : buildTree(umbrellaName, subcharts, rules, traits, dependencies);
  return (
    <div style={style}>
      {label && (
        <span style={{ display: "block", fontFamily: "var(--font-sans)", fontSize: "var(--text-label-size)", fontWeight: "var(--font-weight-semibold)", color: "var(--text-2)", marginBottom: 8 }}>
          {label}
        </span>
      )}
      <pre
        style={{
          margin: 0,
          background: "var(--surface-code)",
          color: "var(--text-on-code)",
          borderRadius: "var(--radius-code)",
          padding: 18,
          fontFamily: "var(--font-mono)",
          fontSize: 12.5,
          lineHeight: "var(--line-height-code)",
          overflowX: "auto",
          whiteSpace: "pre",
        }}
      >
        {text}
      </pre>
    </div>
  );
}

// The manifest list a subchart's templates/ carries after trait tailoring —
// mirrors internal/engine/traits.go (template drops) + applyWorkload.
function subchartManifests(t) {
  if (!t) return null;
  const files = [`${t.workload || "deployment"}.yaml`];
  if (t.exposure !== "none") files.push("service.yaml");
  if (t.ingress) files.push("ingress.yaml");
  if (t.scaling === "auto" && t.workload !== "daemonset") files.push("hpa.yaml");
  if (t.exposure !== "none") files.push("networkPolicy.yaml");
  return files;
}

function buildTree(name, subcharts, rules, traits, dependencies) {
  const entries = (subcharts || [])
    .map((s, i) => ({ s, t: traits ? traits[i] : null }))
    .filter(({ s }) => s && s.name && s.name.trim());
  const deps = (dependencies || []).filter(Boolean);
  const r = rules || {};
  let t = `${(name || "umbrella-chart").trim() || "umbrella-chart"}/\n`;
  t += deps.length ? `├── Chart.yaml   (+${deps.length} ${deps.length === 1 ? "dependency" : "dependencies"}: ${deps.join(", ")})\n` : "├── Chart.yaml\n";
  t += "├── values.yaml\n";
  if (r.generate_handoff !== false) t += "├── HANDOFF.md\n";
  if (r.generate_umbrella_readme) t += "├── README.md\n";
  if (r.include_docs) t += "├── docs/\n";
  if (r.linked_templates) t += "├── templates/   (shared logic)\n";
  t += "└── charts/\n";
  if (entries.length === 0) {
    t += "        (add a subchart to populate)\n";
    return t;
  }
  entries.forEach(({ s, t: rt }, i) => {
    const last = i === entries.length - 1;
    const branch = last ? "    └──" : "    ├──";
    const pipe = last ? "        " : "    │   ";
    const tag = rt ? rt.pattern : s.workload || "deployment";
    t += `${branch} ${s.name.trim()}/   (${tag})\n`;
    t += `${pipe}├── Chart.yaml\n`;
    t += `${pipe}├── values.yaml\n`;
    if (r.generate_subchart_readme) t += `${pipe}├── README.md\n`;
    t += `${pipe}└── templates/\n`;
    const files = subchartManifests(rt);
    if (!files) return;
    files.forEach((f, j) => {
      const fLast = j === files.length - 1;
      t += `${pipe}    ${fLast ? "└──" : "├──"} ${f}\n`;
    });
  });
  return t;
}
