import React from "react";

/**
 * chartpress CodeEditor — a deliberately lightweight "mini VS Code" file pane:
 * a filename header (+ language label, dirty dot, Save / Revert), a line-number
 * gutter, and an editable area with real YAML / Helm-template syntax
 * highlighting (a transparent textarea layered over a highlighted <pre>).
 *
 * Controlled: pass `value` + `onChange`. `dirty` drives the Save button and the
 * header dot; `onSave` / `onRevert` fire the respective actions.
 */
export function CodeEditor({
  path = "untitled",
  value = "",
  onChange,
  onSave,
  onRevert,
  dirty = false,
  readOnly = false,
  language,
  style = {},
}) {
  const taRef = React.useRef(null);
  const preRef = React.useRef(null);
  const gutterRef = React.useRef(null);

  const lang = language || langFromExt(path);
  const fileName = String(path).split("/").pop();
  const lineCount = value.split("\n").length;

  const syncScroll = () => {
    const ta = taRef.current;
    if (!ta) return;
    if (preRef.current) { preRef.current.scrollTop = ta.scrollTop; preRef.current.scrollLeft = ta.scrollLeft; }
    if (gutterRef.current) gutterRef.current.scrollTop = ta.scrollTop;
  };

  const onKeyDown = (e) => {
    // Tab inserts two spaces instead of moving focus
    if (e.key === "Tab" && !readOnly) {
      e.preventDefault();
      const ta = e.target;
      const s = ta.selectionStart, end = ta.selectionEnd;
      const next = value.slice(0, s) + "  " + value.slice(end);
      onChange && onChange(next);
      requestAnimationFrame(() => { ta.selectionStart = ta.selectionEnd = s + 2; });
    }
    // Cmd/Ctrl+S saves
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "s") {
      e.preventDefault();
      if (dirty && onSave) onSave();
    }
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", minHeight: 0, height: "100%", background: "var(--surface-card)", border: "1px solid var(--border-default)", borderRadius: "var(--radius-code)", overflow: "hidden", ...style }}>
      {/* header */}
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12, padding: "9px 12px", borderBottom: "1px solid var(--border-subtle)", background: "var(--slate-2)", flexShrink: 0 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 0 }}>
          <span style={{ display: "inline-flex", color: "var(--text-muted)" }}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round"><path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8Z" /><path d="M14 3v5h5" /></svg>
          </span>
          <span style={{ fontFamily: "var(--font-mono)", fontSize: 12.5, color: "var(--text-1)", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{fileName}</span>
          {dirty && <span title="Unsaved changes" style={{ width: 7, height: 7, borderRadius: "50%", background: "var(--accent-9)", flexShrink: 0 }} />}
          <span style={{ fontFamily: "var(--font-mono)", fontSize: 10.5, textTransform: "uppercase", letterSpacing: "0.05em", color: "var(--text-muted)", border: "1px solid var(--border-default)", borderRadius: 5, padding: "1px 6px", marginLeft: 2, flexShrink: 0 }}>{lang}</span>
        </div>
        {!readOnly && (
          <div style={{ display: "flex", alignItems: "center", gap: 6, flexShrink: 0 }}>
            <MiniBtn onClick={onRevert} disabled={!dirty}>Revert</MiniBtn>
            <MiniBtn onClick={onSave} disabled={!dirty} primary>Save</MiniBtn>
          </div>
        )}
      </div>

      {/* body: gutter + code */}
      <div style={{ display: "flex", flex: 1, minHeight: 0, position: "relative" }}>
        <div ref={gutterRef} style={{ ...codeFont, overflow: "hidden", padding: "14px 10px 14px 14px", textAlign: "right", color: "var(--slate-9)", userSelect: "none", background: "var(--slate-2)", borderRight: "1px solid var(--border-subtle)", flexShrink: 0 }}>
          {Array.from({ length: lineCount }, (_, i) => <div key={i}>{i + 1}</div>)}
        </div>
        <div style={{ position: "relative", flex: 1, minWidth: 0 }}>
          <pre ref={preRef} aria-hidden="true" style={{ ...codeFont, ...codeBox, position: "absolute", inset: 0, width: "100%", height: "100%", margin: 0, color: "var(--text-1)", pointerEvents: "none", overflow: "hidden" }}
            dangerouslySetInnerHTML={{ __html: highlight(value) + "\n" }} />
          <textarea
            ref={taRef}
            value={value}
            readOnly={readOnly}
            spellCheck={false}
            onChange={(e) => onChange && onChange(e.target.value)}
            onScroll={syncScroll}
            onKeyDown={onKeyDown}
            style={{
              ...codeFont, ...codeBox, position: "absolute", inset: 0, width: "100%", height: "100%",
              border: "none", outline: "none", resize: "none", background: "transparent",
              color: "transparent", caretColor: "var(--text-1)", overflow: "auto",
            }}
          />
        </div>
      </div>
    </div>
  );
}

function MiniBtn({ children, onClick, disabled, primary }) {
  const [hover, setHover] = React.useState(false);
  return (
    <button type="button" onClick={onClick} disabled={disabled}
      onMouseEnter={() => setHover(true)} onMouseLeave={() => setHover(false)}
      style={{
        fontFamily: "var(--font-sans)", fontSize: 12, fontWeight: 600, padding: "4px 11px",
        borderRadius: "var(--radius-2)", cursor: disabled ? "not-allowed" : "pointer",
        border: primary ? "none" : "1px solid var(--border-default)",
        background: primary ? (disabled ? "var(--accent-7)" : hover ? "var(--accent-10)" : "var(--accent-9)") : hover && !disabled ? "var(--slate-3)" : "var(--surface-card)",
        color: primary ? "#fff" : disabled ? "var(--text-muted)" : "var(--text-1)",
        opacity: primary && disabled ? 0.7 : 1, transition: "background-color .13s",
      }}>
      {children}
    </button>
  );
}

const codeFont = { fontFamily: "var(--font-mono)", fontSize: 12.5, lineHeight: "20px", tabSize: 2, WebkitTabSize: 2 };
const codeBox = { padding: 14, whiteSpace: "pre", overflowWrap: "normal", wordBreak: "normal", boxSizing: "border-box" };

/* ---------- syntax highlighting (YAML + Helm templates) — light theme ---------- */
const C = {
  comment: "var(--slate-10)",
  key: "var(--accent-11)",
  string: "var(--green-11)",
  bool: "#9a6700",
  num: "#9a6700",
  tpl: "#7b2ff7",
  punct: "var(--text-muted)",
};

function esc(s) {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function span(color, text) {
  return '<span style="color:' + color + '">' + text + "</span>";
}

// Inline scan: comments, {{ templates }}, strings, booleans, numbers. Single
// left-to-right pass so we never re-match already-emitted markup.
function highlightInline(text) {
  const re = /(#.*)|(\{\{[^}]*\}\}|\{\{-?[^}]*-?\}\})|("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')|(\btrue\b|\bfalse\b|\bnull\b|\byes\b|\bno\b)|(-?\b\d+(?:\.\d+)?\b)/g;
  let out = "", last = 0, m;
  while ((m = re.exec(text))) {
    out += esc(text.slice(last, m.index));
    let color = C.num;
    if (m[1]) color = C.comment;
    else if (m[2]) color = C.tpl;
    else if (m[3]) color = C.string;
    else if (m[4]) color = C.bool;
    out += span(color, esc(m[0]));
    last = m.index + m[0].length;
  }
  out += esc(text.slice(last));
  return out;
}

function highlightLine(line) {
  // key: value  (optionally a "- " list-item prefix)
  const km = line.match(/^(\s*(?:-\s+)?)([A-Za-z0-9_.\-\/]+)(:)(\s*)([\s\S]*)$/);
  if (km) {
    return esc(km[1]) + span(C.key, esc(km[2])) + span(C.punct, ":") + esc(km[4]) + highlightInline(km[5]);
  }
  return highlightInline(line);
}

function highlight(code) {
  return String(code).split("\n").map(highlightLine).join("\n");
}

function langFromExt(path) {
  const ext = String(path).split(".").pop().toLowerCase();
  if (ext === "yaml" || ext === "yml") return "YAML";
  if (ext === "tpl") return "Helm";
  if (ext === "md") return "Markdown";
  if (ext === "json") return "JSON";
  return "text";
}
