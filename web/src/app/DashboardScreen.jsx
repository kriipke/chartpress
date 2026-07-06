// Dashboard — the home tab (alongside Generate and Charts). It doesn't collect
// input; it orients you: what chartpress does (scaffold a chart + emit a per-
// subchart HANDOFF prompt you pair with your real code), a peek at that prompt,
// the catalog of subchart patterns it knows, and your recent charts.
import React from "react";
import { Card, Button, StatusBadge, Badge } from "../design/components";
import { Plus, ArrowRight, Package, Sparkle, Layers } from "./Icons.jsx";
import { PATTERNS } from "./spec.js";

// A faithful excerpt of the HANDOFF.md chartpress writes per subchart. The
// checklist mirrors the api-microservice pattern's finishing steps (patterns.json).
const HANDOFF = [
  { t: "c", s: "# HANDOFF — ledger-api  (pattern: api-microservice)" },
  { t: "b" },
  { t: "p", s: "Finish this Helm chart scaffold against the service's real code." },
  { t: "p", s: "Edit values.yaml and templates/ to match what the app actually does." },
  { t: "b" },
  { t: "todo", s: "Find the port the app listens on (code / Dockerfile EXPOSE) → service.port" },
  { t: "todo", s: "Wire real liveness & readiness probes; readiness may check deps" },
  { t: "todo", s: "Map env vars: shared config vs. this chart's own Secret" },
  { t: "todo", s: "Measure shutdown drain time → terminationGracePeriodSeconds" },
  { t: "todo", s: "Size resources.requests and the memory limit from expected load" },
];
const HANDOFF_TEXT = HANDOFF.map((l) => (l.t === "b" ? "" : l.t === "todo" ? `- [ ] ${l.s}` : l.s)).join("\n");

export function DashboardScreen({ user, charts = [], onGenerate, onOpenChart, onBrowse }) {
  const recents = charts.slice(0, 4);

  return (
    <div style={{ maxWidth: 760, margin: "0 auto" }}>
      <p style={{ margin: "0 0 6px", fontSize: 13, color: "var(--text-muted)" }}>
        {user ? `Welcome back, ${user.name}` : "chartpress"}
      </p>
      <h1 style={{ margin: "0 0 22px", fontSize: 27, fontWeight: 700, letterSpacing: "-0.02em", color: "var(--text-1)" }}>
        What are you <span style={{ color: "var(--accent-9)" }}>deploying</span>?
      </h1>

      <Card padding={0} elevation={2} style={{ overflow: "hidden" }}>
        <div style={{ padding: 24 }}>
          <p style={{ margin: 0, fontSize: 15, lineHeight: 1.6, color: "var(--text-1)" }}>
            <b style={{ fontWeight: 600 }}>chartpress</b> is a Helm chart scaffolding tool. Describe the
            services your app is made of and it generates a complete umbrella chart and one subchart per
            service — plus a matching <b style={{ fontWeight: 600 }}>LLM prompt</b> for each. Pair that prompt
            with the code you actually want to ship, and an agent (or you) finishes the chart against what
            the app really does.
          </p>

          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12, marginTop: 20 }}>
            <Step
              n="1" icon={<Package size={16} />} title="Scaffold"
              body="You name the services; chartpress writes the umbrella chart, a subchart for each, wires dependencies, and applies your rules."
            />
            <Step
              n="2" icon={<Sparkle size={16} />} title="Hand off to an LLM"
              body="Every subchart ships a HANDOFF checklist. Give it to an agent with your app code — it resolves ports, probes, env, and resources."
            />
          </div>
        </div>

        {/* A peek at the generated prompt */}
        <div style={{ padding: "0 24px 20px" }}>
          <div style={snippetWrap}>
            <div style={snippetBar}>
              <span style={{ display: "inline-flex", alignItems: "center", gap: 7, color: "var(--text-on-code)", opacity: 0.85, fontSize: 12, fontFamily: "var(--font-mono)" }}>
                <FileGlyph /> HANDOFF.md
              </span>
              <CopyButton text={HANDOFF_TEXT} />
            </div>
            <pre style={snippetPre}>
              {HANDOFF.map((l, i) => <Line key={i} line={l} />)}
            </pre>
          </div>
        </div>

        {/* Subchart pattern catalog */}
        <div style={{ padding: "0 24px 22px" }}>
          <div style={sectionLabel}>
            <Layers size={13} /> Subchart patterns it knows
            <Badge color="neutral" variant="soft" size="1" style={{ marginLeft: 2 }}>{PATTERNS.length}</Badge>
          </div>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
            {PATTERNS.map((p) => <Pill key={p.id} label={p.label} />)}
          </div>
        </div>

        {/* Actions */}
        <div style={ctaRow}>
          <Button size="3" leadingIcon={<Plus size={16} />} onClick={onGenerate}>Generate a chart</Button>
          <Button size="3" variant="soft" trailingIcon={<ArrowRight size={16} />} onClick={onBrowse}>Browse your charts</Button>
        </div>
      </Card>

      {recents.length > 0 && (
        <div style={{ marginTop: 34 }}>
          <div style={{ display: "flex", alignItems: "baseline", justifyContent: "space-between", marginBottom: 10 }}>
            <h2 style={{ margin: 0, fontSize: 13, fontWeight: 600, color: "var(--text-1)" }}>Pick up where you left off</h2>
            <button onClick={onBrowse} style={linkBtn}>All charts →</button>
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {recents.map((c, i) => <RecentRow key={c.name + i} chart={c} onOpen={onOpenChart} onBrowse={onBrowse} />)}
          </div>
        </div>
      )}
    </div>
  );
}

function Step({ n, icon, title, body }) {
  return (
    <div style={{ background: "var(--surface-sunken)", border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-4)", padding: "13px 14px" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 6 }}>
        <span style={{ width: 24, height: 24, borderRadius: "50%", background: "var(--accent-3)", color: "var(--accent-11)", display: "grid", placeItems: "center", flexShrink: 0 }}>{icon}</span>
        <span style={{ fontSize: 13.5, fontWeight: 600, color: "var(--text-1)" }}>{title}</span>
        <span style={{ marginLeft: "auto", fontSize: 11, fontWeight: 600, color: "var(--text-muted)", fontVariantNumeric: "tabular-nums" }}>{n}</span>
      </div>
      <p style={{ margin: 0, fontSize: 12.5, lineHeight: 1.5, color: "var(--text-2)" }}>{body}</p>
    </div>
  );
}

function Pill({ label }) {
  return (
    <span style={{
      fontSize: 12.5, fontFamily: "var(--font-mono)", color: "var(--text-2)",
      background: "var(--surface-card)", border: "1px solid var(--border-default)",
      borderRadius: "var(--radius-full)", padding: "4px 11px", whiteSpace: "nowrap",
    }}>{label}</span>
  );
}

function Line({ line }) {
  if (line.t === "b") return <div style={{ height: "1.55em" }} />;
  if (line.t === "c") return <div style={{ color: "#7bd88f" }}>{line.s}</div>;
  if (line.t === "p") return <div style={{ color: "#9a9ab5" }}>{line.s}</div>;
  return (
    <div style={{ color: "var(--text-on-code)" }}>
      <span style={{ color: "#8fb3ff" }}>- [ ] </span>{line.s}
    </div>
  );
}

function CopyButton({ text }) {
  const [copied, setCopied] = React.useState(false);
  const copy = () => {
    try {
      navigator.clipboard?.writeText(text).then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 1400);
      });
    } catch { /* clipboard unavailable — ignore */ }
  };
  return (
    <button onClick={copy} style={copyBtn}>
      {copied ? "Copied" : "Copy"}
    </button>
  );
}

function RecentRow({ chart, onOpen, onBrowse }) {
  const [hover, setHover] = React.useState(false);
  // The explorer's file fetch 409s until a chart is Ready, and this row is a stale
  // snapshot that won't re-poll — so only open Ready/Failed charts directly. Send a
  // still-building chart to the Charts list, where polling shows it advance.
  const phase = String(chart.phase || "").toLowerCase();
  const openable = phase === "ready" || phase === "failed";
  const handleClick = () => (openable ? onOpen?.(chart) : onBrowse?.());
  return (
    <button
      onClick={handleClick}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: "flex", alignItems: "center", gap: 12, textAlign: "left", width: "100%",
        padding: "11px 14px", background: "var(--surface-card)", cursor: "pointer",
        border: "1px solid " + (hover ? "var(--slate-7)" : "var(--border-default)"),
        borderRadius: "var(--radius-4)", boxShadow: hover ? "var(--shadow-2)" : "none",
        transition: "box-shadow .12s, border-color .12s",
      }}
    >
      <span style={{ fontFamily: "var(--font-mono)", fontWeight: 500, fontSize: 13.5, color: "var(--text-1)", flex: 1, minWidth: 0, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
        {chart.name}
      </span>
      <span style={{ fontSize: 12, color: "var(--text-muted)", whiteSpace: "nowrap" }}>
        {chart.subchartCount != null ? `${chart.subchartCount} subchart${chart.subchartCount === 1 ? "" : "s"}` : ""}
        {chart.lastGenerated ? ` · ${timeAgo(chart.lastGenerated)}` : ""}
      </span>
      <StatusBadge phase={chart.phase} size="1" />
    </button>
  );
}

function timeAgo(ts) {
  if (!ts) return "";
  const d = new Date(ts);
  if (isNaN(d)) return String(ts);
  const diff = (Date.now() - d.getTime()) / 1000;
  if (diff < 60) return "just now";
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function FileGlyph() {
  return <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8l-5-5Z" /><path d="M14 3v5h5" /></svg>;
}

const snippetWrap = {
  background: "var(--surface-code)", borderRadius: "var(--radius-code)", overflow: "hidden",
  border: "1px solid rgba(255,255,255,0.06)",
};
const snippetBar = {
  display: "flex", alignItems: "center", justifyContent: "space-between",
  padding: "8px 12px", borderBottom: "1px solid rgba(255,255,255,0.07)",
};
const snippetPre = {
  margin: 0, padding: "12px 14px", fontFamily: "var(--font-mono)", fontSize: 12.5,
  lineHeight: "var(--line-height-code)", color: "var(--text-on-code)",
  whiteSpace: "pre-wrap", wordBreak: "break-word", overflowX: "auto",
};
const copyBtn = {
  background: "rgba(255,255,255,0.08)", color: "var(--text-on-code)", border: "none",
  borderRadius: "var(--radius-2)", padding: "3px 9px", fontSize: 11.5, fontWeight: 600,
  fontFamily: "var(--font-sans)", cursor: "pointer",
};
const sectionLabel = {
  display: "flex", alignItems: "center", gap: 7, marginBottom: 10,
  fontSize: 12, fontWeight: 600, color: "var(--text-muted)",
};
const ctaRow = {
  display: "flex", gap: 10, flexWrap: "wrap", padding: "18px 24px",
  borderTop: "1px solid var(--border-subtle)", background: "var(--surface-sunken)",
};
const linkBtn = {
  background: "transparent", border: "none", color: "var(--text-accent)",
  fontSize: 13, fontWeight: 500, cursor: "pointer", fontFamily: "var(--font-sans)", padding: 0,
};
