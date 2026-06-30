import React from "react";

/**
 * chartpress Badge — small rounded label. Neutral by default; accent / green /
 * red / amber for semantic use. Soft (tinted) or solid surface.
 */
export function Badge({
  children,
  color = "neutral",
  variant = "soft",
  size = "2",
  style = {},
  leadingDot = false,
  ...props
}) {
  const palettes = {
    neutral: { fg: "var(--slate-11)", bg: "var(--slate-3)", border: "var(--slate-6)", solid: "var(--slate-9)" },
    accent: { fg: "var(--violet-11)", bg: "var(--violet-3)", border: "var(--violet-6)", solid: "var(--violet-9)" },
    green: { fg: "var(--green-11)", bg: "var(--green-3)", border: "var(--green-6)", solid: "var(--green-9)" },
    red: { fg: "var(--red-11)", bg: "var(--red-3)", border: "var(--red-6)", solid: "var(--red-9)" },
    amber: { fg: "var(--amber-11)", bg: "var(--amber-3)", border: "var(--amber-6)", solid: "var(--amber-9)" },
  };
  const p = palettes[color] || palettes.neutral;
  const sizes = {
    "1": { fontSize: 11, padding: "2px 7px", gap: 4, dot: 5 },
    "2": { fontSize: 12, padding: "3px 9px", gap: 5, dot: 6 },
  };
  const s = sizes[size] || sizes["2"];

  const variants = {
    soft: { background: p.bg, color: p.fg, border: "1px solid transparent" },
    outline: { background: "transparent", color: p.fg, border: `1px solid ${p.border}` },
    solid: { background: p.solid, color: "#fff", border: "1px solid transparent" },
  };

  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: s.gap,
        padding: s.padding,
        fontFamily: "var(--font-sans)",
        fontSize: s.fontSize,
        fontWeight: "var(--font-weight-semibold)",
        lineHeight: 1,
        borderRadius: "var(--radius-badge)",
        whiteSpace: "nowrap",
        ...variants[variant],
        ...style,
      }}
      {...props}
    >
      {leadingDot && (
        <span style={{ width: s.dot, height: s.dot, borderRadius: "50%", background: variant === "solid" ? "#fff" : p.solid, flexShrink: 0 }} />
      )}
      {children}
    </span>
  );
}
