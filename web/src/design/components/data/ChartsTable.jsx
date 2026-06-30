import React from "react";
import { StatusBadge } from "../feedback/StatusBadge.jsx";
import { Button } from "../forms/Button.jsx";

/**
 * chartpress ChartsTable — the Charts browser list. One row per generated chart:
 * name · phase badge · subchart count · last generated · action (Download when
 * Ready, error message when Failed). Header + hairline row separators.
 */
export function ChartsTable({ charts = [], onDownload, style = {} }) {
  const cell = { padding: "14px 16px", fontFamily: "var(--font-sans)", fontSize: "var(--font-size-2)", color: "var(--text-2)", verticalAlign: "middle" };
  const head = { ...cell, fontSize: 12, fontWeight: "var(--font-weight-semibold)", color: "var(--text-muted)", textTransform: "uppercase", letterSpacing: "0.04em", padding: "10px 16px" };

  return (
    <table style={{ width: "100%", borderCollapse: "collapse", background: "var(--surface-card)", ...style }}>
      <thead>
        <tr style={{ borderBottom: "1px solid var(--border-default)" }}>
          <th style={{ ...head, textAlign: "left" }}>Name</th>
          <th style={{ ...head, textAlign: "left" }}>Status</th>
          <th style={{ ...head, textAlign: "right", whiteSpace: "nowrap" }}>Subcharts</th>
          <th style={{ ...head, textAlign: "left" }}>Last generated</th>
          <th style={{ ...head, textAlign: "right" }}>Action</th>
        </tr>
      </thead>
      <tbody>
        {charts.map((c, i) => {
          const phase = String(c.phase || "Pending").toLowerCase();
          return (
            <tr key={c.name + i} style={{ borderBottom: "1px solid var(--slate-4)", animation: c.isNew ? "cp-row-in .35s var(--ease-standard)" : "none" }}>
              <td style={{ ...cell }}>
                <span style={{ fontFamily: "var(--font-mono)", fontSize: 13, fontWeight: 500, color: "var(--text-1)" }}>{c.name}</span>
              </td>
              <td style={cell}><StatusBadge phase={c.phase} /></td>
              <td style={{ ...cell, textAlign: "right", fontVariantNumeric: "tabular-nums" }}>{c.subchartCount ?? "—"}</td>
              <td style={{ ...cell, color: "var(--text-muted)", whiteSpace: "nowrap" }}>{formatTime(c.lastGenerated)}</td>
              <td style={{ ...cell, textAlign: "right" }}>
                {phase === "ready" ? (
                  <Button size="1" variant="soft" onClick={() => onDownload?.(c)} leadingIcon={<DownloadGlyph />}>Download</Button>
                ) : phase === "failed" ? (
                  <span style={{ fontSize: "var(--font-size-1)", color: "var(--red-11)", whiteSpace: "normal" }}>{c.message || "Generation failed"}</span>
                ) : (
                  <span style={{ fontSize: "var(--font-size-1)", color: "var(--text-muted)" }}>—</span>
                )}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

function DownloadGlyph() {
  return <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 3v12m0 0 4-4m-4 4-4-4M5 21h14" /></svg>;
}

function formatTime(ts) {
  if (!ts) return "—";
  const d = new Date(ts);
  if (isNaN(d)) return String(ts);
  const diff = (Date.now() - d.getTime()) / 1000;
  if (diff < 60) return "just now";
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}
