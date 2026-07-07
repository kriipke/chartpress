import React from "react";

/**
 * chartpress StatusBadge — the signature 4-phase chart-generation indicator.
 *   Pending     → neutral slate, static dot (queued)
 *   Generating  → brand azure, pulsing dot (alive / in-progress)
 *   Ready       → green, check (success)
 *   Failed      → red, cross (error)
 * Phase strings are case-insensitive.
 */
export function StatusBadge({ phase = "Pending", size = "2", style = {}, ...props }) {
  const key = String(phase).toLowerCase();
  const config = {
    pending: { label: "Pending", fg: "var(--status-pending-fg)", bg: "var(--status-pending-bg)", border: "var(--status-pending-border)", dot: "var(--slate-9)", pulse: false, icon: null },
    generating: { label: "Generating", fg: "var(--status-generating-fg)", bg: "var(--status-generating-bg)", border: "var(--status-generating-border)", dot: "var(--accent-9)", pulse: true, icon: null },
    ready: { label: "Ready", fg: "var(--status-ready-fg)", bg: "var(--status-ready-bg)", border: "var(--status-ready-border)", dot: "var(--green-9)", pulse: false, icon: "check" },
    failed: { label: "Failed", fg: "var(--status-failed-fg)", bg: "var(--status-failed-bg)", border: "var(--status-failed-border)", dot: "var(--red-9)", pulse: false, icon: "cross" },
    // Expired: an anonymous chart whose server-side artifact was reaped past its
    // TTL. The spec is still remembered locally, so it can be regenerated.
    expired: { label: "Expired", fg: "var(--status-pending-fg)", bg: "var(--status-pending-bg)", border: "var(--status-pending-border)", dot: "var(--slate-8)", pulse: false, icon: null },
  };
  const c = config[key] || config.pending;
  const sizes = { "1": { fontSize: 11, padding: "2px 8px 2px 7px", gap: 5, dot: 6 }, "2": { fontSize: 12, padding: "4px 10px 4px 8px", gap: 6, dot: 7 } };
  const s = sizes[size] || sizes["2"];

  const Marker = () => {
    if (c.icon === "check") {
      return <svg width={s.dot + 4} height={s.dot + 4} viewBox="0 0 24 24" fill="none" stroke="var(--green-11)" strokeWidth="3.5" strokeLinecap="round" strokeLinejoin="round"><path d="M20 6 9 17l-5-5" /></svg>;
    }
    if (c.icon === "cross") {
      return <svg width={s.dot + 3} height={s.dot + 3} viewBox="0 0 24 24" fill="none" stroke="var(--red-11)" strokeWidth="3.5" strokeLinecap="round" strokeLinejoin="round"><path d="M18 6 6 18M6 6l12 12" /></svg>;
    }
    return (
      <span
        style={{
          width: s.dot, height: s.dot, borderRadius: "50%", background: c.dot, flexShrink: 0,
          animation: c.pulse ? "cp-pulse 1.2s var(--ease-standard) infinite" : "none",
          boxShadow: c.pulse ? "0 0 0 3px var(--accent-a4)" : "none",
        }}
      />
    );
  };

  return (
    <span
      role="status"
      style={{
        display: "inline-flex", alignItems: "center", gap: s.gap, padding: s.padding,
        fontFamily: "var(--font-sans)", fontSize: s.fontSize, fontWeight: "var(--font-weight-semibold)",
        lineHeight: 1, color: c.fg, background: c.bg, border: `1px solid ${c.border}`,
        borderRadius: "var(--radius-badge)", whiteSpace: "nowrap", ...style,
      }}
      {...props}
    >
      <Marker />
      {c.label}
    </span>
  );
}
