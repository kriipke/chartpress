import React from "react";

/**
 * chartpress StructurePreview — the dark terminal-style read-only preview of the
 * chart tree that would be generated. Renders box-drawing characters in mono.
 * Pass a precomputed `tree` string, or `umbrellaName` + `subcharts` + `rules`
 * to have it build the tree itself.
 */
export function StructurePreview({
  tree,
  umbrellaName = "umbrella-chart",
  subcharts = [],
  rules = {},
  label = "Structure preview",
  style = {},
}) {
  const text = tree != null ? tree : buildTree(umbrellaName, subcharts, rules);
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

function buildTree(name, subcharts, rules) {
  const named = (subcharts || []).filter((s) => s && s.name && s.name.trim());
  const r = rules || {};
  let t = `${(name || "umbrella-chart").trim() || "umbrella-chart"}/\n`;
  t += "├── Chart.yaml\n";
  t += "├── values.yaml\n";
  if (r.generate_umbrella_readme) t += "├── README.md\n";
  if (r.include_docs) t += "├── docs/\n";
  if (r.linked_templates) t += "├── templates/   (shared logic)\n";
  t += "└── charts/\n";
  if (named.length === 0) {
    t += "        (add a subchart to populate)\n";
    return t;
  }
  named.forEach((sc, i) => {
    const last = i === named.length - 1;
    const branch = last ? "    └──" : "    ├──";
    const pipe = last ? "        " : "    │   ";
    t += `${branch} ${sc.name.trim()}/   (${sc.workload || "deployment"})\n`;
    t += `${pipe}├── Chart.yaml\n`;
    t += `${pipe}├── values.yaml\n`;
    if (r.generate_subchart_readme) t += `${pipe}├── README.md\n`;
    t += `${pipe}└── templates/\n`;
  });
  return t;
}
