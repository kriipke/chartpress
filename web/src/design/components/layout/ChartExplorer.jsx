import React from "react";
import { FileTree } from "../data/FileTree.jsx";
import { CodeEditor } from "../data/CodeEditor.jsx";
import { StatusBadge } from "../feedback/StatusBadge.jsx";
import { Spinner } from "../feedback/Spinner.jsx";
import { Button } from "../forms/Button.jsx";

/**
 * chartpress ChartExplorer — the individual chart view. A two-pane shell: a
 * FileTree on the left and a CodeEditor on the right, under a header with the
 * chart name + StatusBadge and (when Ready) a Download action. When the chart's
 * phase is Failed it swaps the panes for an error panel with a Regenerate
 * action.
 *
 * `readOnly` (default) hides Save/Revert and disables editing: the backend
 * regenerates each chart from its spec, so the explorer browses rather than
 * persists. `loading` / `loadError` drive the right pane while a chart's files
 * are fetched from the server.
 */
export function ChartExplorer({
  name,
  phase = "Ready",
  nodes = [],
  files = {},
  message,
  onBack,
  onSave,
  onRegenerate,
  onDownload,
  readOnly = true,
  loading = false,
  loadError = "",
  style = {},
}) {
  const isFailed = String(phase).toLowerCase() === "failed";
  const firstFile = React.useMemo(() => firstFilePath(nodes), [nodes]);
  const [selected, setSelected] = React.useState(firstFile);
  const [work, setWork] = React.useState(files);
  const [saved, setSaved] = React.useState(files);

  React.useEffect(() => { setWork(files); setSaved(files); }, [files]);
  React.useEffect(() => { setSelected(firstFile); }, [firstFile]);

  const curContent = selected != null ? (work[selected] != null ? work[selected] : stubFor(selected)) : "";
  const dirty = !readOnly && selected != null && work[selected] !== saved[selected];

  const handleSave = () => {
    if (readOnly || selected == null) return;
    setSaved((s) => ({ ...s, [selected]: work[selected] }));
    if (onSave) onSave(selected, work[selected]);
  };
  const handleRevert = () => {
    if (readOnly || selected == null) return;
    setWork((w) => ({ ...w, [selected]: saved[selected] }));
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%", minHeight: 0, ...style }}>
      {/* header */}
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12, marginBottom: 16, flexShrink: 0 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 12, minWidth: 0 }}>
          {onBack && (
            <button type="button" onClick={onBack} aria-label="Back to charts" style={backBtn}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M19 12H5m0 0 6 6m-6-6 6-6" /></svg>
            </button>
          )}
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700, letterSpacing: "-0.01em", color: "var(--text-1)", fontFamily: "var(--font-mono)", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{name}</h1>
          <StatusBadge phase={phase} />
        </div>
        <div style={{ display: "flex", gap: 8, flexShrink: 0 }}>
          {isFailed
            ? <Button onClick={onRegenerate}>Regenerate</Button>
            : <Button variant="outline" onClick={onDownload}>Download .tgz</Button>}
        </div>
      </div>

      {isFailed ? (
        <FailedPanel name={name} message={message} onRegenerate={onRegenerate} />
      ) : loading ? (
        <CenteredPane>
          <Spinner size={20} color="var(--accent-9)" />
          <span style={{ color: "var(--text-2)", fontSize: 14 }}>Loading chart files…</span>
        </CenteredPane>
      ) : loadError ? (
        <CenteredPane>
          <div style={{ fontSize: 15, fontWeight: 600, color: "var(--text-1)" }}>Couldn&apos;t load files</div>
          <div style={{ fontFamily: "var(--font-mono)", fontSize: 12.5, color: "var(--red-11)", background: "var(--red-3)", border: "1px solid var(--red-6)", borderRadius: "var(--radius-3)", padding: "8px 12px", maxWidth: 480, whiteSpace: "pre-wrap" }}>{loadError}</div>
        </CenteredPane>
      ) : (
        <div style={{ display: "grid", gridTemplateColumns: "252px 1fr", gap: 14, flex: 1, minHeight: 0 }}>
          <div style={{ background: "var(--surface-card)", border: "1px solid var(--border-default)", borderRadius: "var(--radius-card)", padding: 10, overflow: "auto", minHeight: 0 }}>
            <div style={{ fontFamily: "var(--font-sans)", fontSize: 11, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.05em", color: "var(--text-muted)", padding: "4px 8px 8px" }}>Files</div>
            <FileTree nodes={nodes} selectedPath={selected} onSelect={setSelected} />
          </div>
          <div style={{ minHeight: 0, minWidth: 0 }}>
            <CodeEditor
              path={selected || "select-a-file"}
              value={curContent}
              dirty={dirty}
              readOnly={readOnly}
              onChange={(v) => setWork((w) => ({ ...w, [selected]: v }))}
              onSave={handleSave}
              onRevert={handleRevert}
              style={{ height: "100%" }}
            />
          </div>
        </div>
      )}
    </div>
  );
}

function FailedPanel({ name, message, onRegenerate }) {
  return (
    <div style={{
      flex: 1, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center",
      textAlign: "center", padding: 40, borderRadius: "var(--radius-card)",
      border: "1px solid var(--border-default)", background: "var(--surface-card)",
      backgroundImage: "var(--pattern-dots)", backgroundSize: "var(--pattern-dot-size) var(--pattern-dot-size)",
    }}>
      <div style={{ width: 46, height: 46, borderRadius: "50%", background: "var(--red-3)", color: "var(--red-11)", display: "flex", alignItems: "center", justifyContent: "center", marginBottom: 16 }}>
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M12 9v4M12 17h.01M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" /></svg>
      </div>
      <div style={{ fontSize: 18, fontWeight: 700, color: "var(--text-1)", marginBottom: 6 }}>Generation failed</div>
      <div style={{ fontSize: 14, color: "var(--text-2)", maxWidth: 460, lineHeight: 1.55, marginBottom: 4 }}>
        <span style={{ fontFamily: "var(--font-mono)", color: "var(--text-1)" }}>{name}</span> couldn't be rendered.
      </div>
      {message && (
        <div style={{ fontFamily: "var(--font-mono)", fontSize: 12.5, color: "var(--red-11)", background: "var(--red-3)", border: "1px solid var(--red-6)", borderRadius: "var(--radius-3)", padding: "8px 12px", margin: "10px 0 20px", maxWidth: 480 }}>
          {message}
        </div>
      )}
      <Button onClick={onRegenerate}>Regenerate chart</Button>
    </div>
  );
}

// Centered message pane, sized to fill the explorer body (loading / load error).
function CenteredPane({ children }) {
  return (
    <div style={{
      flex: 1, minHeight: 0, display: "flex", flexDirection: "column", alignItems: "center",
      justifyContent: "center", gap: 12, textAlign: "center", padding: 40,
      border: "1px solid var(--border-default)", borderRadius: "var(--radius-card)",
      background: "var(--surface-card)",
    }}>
      {children}
    </div>
  );
}

function firstFilePath(nodes) {
  const walk = (list, prefix) => {
    for (const n of list || []) {
      const p = prefix ? prefix + "/" + n.name : n.name;
      if (Array.isArray(n.children)) {
        const r = walk(n.children, p);
        if (r) return r;
      } else {
        return p;
      }
    }
    return null;
  };
  return walk(nodes, "");
}

function stubFor(path) {
  return "# " + path + "\n# (no preview available for this file)\n";
}

const backBtn = {
  display: "inline-flex", alignItems: "center", justifyContent: "center", width: 32, height: 32,
  borderRadius: "var(--radius-3)", border: "1px solid var(--border-default)", background: "var(--surface-card)",
  color: "var(--text-2)", cursor: "pointer", flexShrink: 0,
};
