import React from "react";

/**
 * chartpress FileTree — an interactive, collapsible file/folder tree for
 * exploring a rendered chart bundle. Folders expand/collapse; files are
 * selectable and fire onSelect(path). File-type glyphs come from the extension.
 *
 * `nodes`: array of { name, children? } — a node with a `children` array is a
 * folder, otherwise a file. Paths are derived by joining names with "/".
 */
export function FileTree({ nodes = [], selectedPath, onSelect, defaultCollapsed = false, style = {} }) {
  return (
    <div style={{ fontFamily: "var(--font-mono)", fontSize: 13, userSelect: "none", ...style }}>
      {nodes.map((n) => (
        <TreeNode key={n.name} node={n} depth={0} prefix="" selectedPath={selectedPath} onSelect={onSelect} defaultCollapsed={defaultCollapsed} />
      ))}
    </div>
  );
}

function TreeNode({ node, depth, prefix, selectedPath, onSelect, defaultCollapsed }) {
  const isFolder = Array.isArray(node.children);
  const path = prefix ? prefix + "/" + node.name : node.name;
  const [open, setOpen] = React.useState(!defaultCollapsed);
  const [hover, setHover] = React.useState(false);
  const selected = selectedPath === path;

  const rowPad = 8 + depth * 15;

  return (
    <div>
      <div
        onClick={() => (isFolder ? setOpen((o) => !o) : onSelect && onSelect(path))}
        onMouseEnter={() => setHover(true)}
        onMouseLeave={() => setHover(false)}
        style={{
          display: "flex", alignItems: "center", gap: 6,
          padding: "4px 8px 4px " + rowPad + "px",
          cursor: "pointer", borderRadius: "var(--radius-3)",
          background: selected ? "var(--accent-3)" : hover ? "var(--slate-3)" : "transparent",
          color: selected ? "var(--accent-11)" : "var(--text-1)",
          fontWeight: selected ? 600 : 400,
          whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis",
        }}
      >
        <span style={{ width: 12, flexShrink: 0, display: "inline-flex", justifyContent: "center", color: "var(--text-muted)" }}>
          {isFolder ? <Chevron open={open} /> : null}
        </span>
        <span style={{ flexShrink: 0, display: "inline-flex", color: isFolder ? "var(--accent-9)" : fileColor(node.name) }}>
          {isFolder ? <FolderGlyph open={open} /> : <FileGlyph name={node.name} />}
        </span>
        <span style={{ overflow: "hidden", textOverflow: "ellipsis" }}>{node.name}</span>
      </div>
      {isFolder && open && node.children.map((c) => (
        <TreeNode key={c.name} node={c} depth={depth + 1} prefix={path} selectedPath={selectedPath} onSelect={onSelect} defaultCollapsed={defaultCollapsed} />
      ))}
    </div>
  );
}

function Chevron({ open }) {
  return (
    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round"
      style={{ transform: open ? "rotate(90deg)" : "none", transition: "transform .12s" }}>
      <path d="m9 6 6 6-6 6" />
    </svg>
  );
}

function FolderGlyph({ open }) {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
      {open
        ? <path d="M3 8h18l-2 10a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h5l2 3" />
        : <path d="M3 6a1 1 0 0 1 1-1h5l2 3h8a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1Z" />}
    </svg>
  );
}

function FileGlyph({ name }) {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
      <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8Z" /><path d="M14 3v5h5" />
    </svg>
  );
}

/** ext → color, so YAML / tpl / md read apart at a glance. */
function fileColor(name) {
  const ext = String(name).split(".").pop().toLowerCase();
  if (ext === "yaml" || ext === "yml") return "var(--accent-9)";
  if (ext === "tpl") return "#7b2ff7";
  if (ext === "md") return "var(--green-11)";
  if (ext === "json") return "#b8860b";
  return "var(--text-muted)";
}
