// ProfileScreen — the signed-in user's GitHub profile: identity header on the
// dotted pattern, chart stats, and account details. `user` carries the fields
// from /auth/me (name, handle, email, registry, avatarUrl); `charts` is the live
// list so the stats reflect the real backend.
import React from "react";
import { Card, Button, GithubMark } from "../design/components";

export function ProfileScreen({ user, charts = [], onBack }) {
  const ready = charts.filter((c) => String(c.phase).toLowerCase() === "ready").length;
  const failed = charts.filter((c) => String(c.phase).toLowerCase() === "failed").length;
  const handle = user.handle || "";

  return (
    <div style={{ maxWidth: 900, margin: "0 auto" }}>
      <button type="button" onClick={onBack} style={backLink}>
        <ArrowLeftGlyph /> Back
      </button>

      {/* identity header on the dotted pattern */}
      <div style={{
        display: "flex", alignItems: "center", gap: 20, padding: "26px 28px", marginTop: 12,
        borderRadius: "var(--radius-card)", border: "1px solid var(--border-default)",
        background: "var(--surface-card)", backgroundImage: "var(--pattern-dots)",
        backgroundSize: "var(--pattern-dot-size) var(--pattern-dot-size)",
      }}>
        {user.avatarUrl
          ? <img src={user.avatarUrl} alt="" width="84" height="84" style={{ borderRadius: "50%", boxShadow: "0 0 0 1px var(--border-default), var(--shadow-3)", display: "block" }} />
          : <span style={avatarFallback}>{initials(user.name || handle)}</span>}
        <div style={{ minWidth: 0 }}>
          <div style={{ fontSize: 24, fontWeight: 700, letterSpacing: "-0.01em", color: "var(--text-1)" }}>{user.name}</div>
          {handle && (
            <div style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 14, color: "var(--text-2)", marginTop: 3 }}>
              <GithubMark size={15} /> {handle}
            </div>
          )}
        </div>
        {handle && (
          <a href={"https://github.com/" + handle.replace(/^@/, "")} target="_blank" rel="noopener noreferrer" style={{ marginLeft: "auto" }}>
            <Button variant="outline">View on GitHub</Button>
          </a>
        )}
      </div>

      {/* stats */}
      <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 14, marginTop: 16 }}>
        <Stat label="Charts" value={charts.length} />
        <Stat label="Ready" value={ready} tone="ready" />
        <Stat label="Failed" value={failed} tone="failed" />
      </div>

      {/* account */}
      <div style={{ marginTop: 16 }}>
        <Card title="Account" subtitle="Connected via GitHub OAuth" padding={0}>
          <Row label="Name" value={user.name} />
          {handle && <Row label="GitHub" value={handle} />}
          {user.email && <Row label="Email" value={user.email} />}
          {user.registry && <Row label="Default registry" value={user.registry} mono last />}
        </Card>
      </div>
    </div>
  );
}

function Stat({ label, value, tone }) {
  const color = tone === "ready" ? "var(--green-11)" : tone === "failed" ? "var(--red-11)" : "var(--accent-11)";
  return (
    <Card padding={18}>
      <div style={{ fontSize: 30, fontWeight: 700, color, fontVariantNumeric: "tabular-nums", letterSpacing: "-0.02em" }}>{value}</div>
      <div style={{ fontSize: 13, color: "var(--text-2)", marginTop: 2 }}>{label}</div>
    </Card>
  );
}

function Row({ label, value, mono, last }) {
  return (
    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", padding: "13px 18px", borderBottom: last ? "none" : "1px solid var(--slate-4)" }}>
      <span style={{ fontSize: 13.5, color: "var(--text-2)" }}>{label}</span>
      <span style={{ fontSize: 13.5, color: "var(--text-1)", fontWeight: 500, fontFamily: mono ? "var(--font-mono)" : "var(--font-sans)" }}>{value}</span>
    </div>
  );
}

function initials(s) {
  const parts = String(s || "?").replace(/^@/, "").split(/[\s._-]+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

function ArrowLeftGlyph() {
  return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M19 12H5m0 0 6 6m-6-6 6-6" /></svg>;
}

const backLink = {
  display: "inline-flex", alignItems: "center", gap: 6, background: "transparent", border: "none",
  color: "var(--text-2)", cursor: "pointer", fontSize: 14, fontWeight: 500, fontFamily: "var(--font-sans)", padding: 0,
};
const avatarFallback = {
  width: 84, height: 84, borderRadius: "50%", display: "inline-flex", alignItems: "center", justifyContent: "center",
  background: "var(--accent-9)", color: "#fff", fontSize: 30, fontWeight: 600, flexShrink: 0,
};
