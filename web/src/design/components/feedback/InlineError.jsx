import React from "react";

/**
 * chartpress InlineError — the form/server error block. Mirrors the app's error
 * panel: red-tinted background, red border, red text, preserves whitespace so
 * raw server messages render verbatim.
 */
export function InlineError({ children, style = {}, ...props }) {
  if (!children) return null;
  return (
    <div
      role="alert"
      style={{
        padding: "10px 12px",
        background: "var(--red-surface)",
        border: "1px solid var(--red-6)",
        borderRadius: "var(--radius-input)",
        color: "var(--red-11)",
        fontFamily: "var(--font-sans)",
        fontSize: "var(--font-size-1)",
        lineHeight: 1.5,
        whiteSpace: "pre-wrap",
        ...style,
      }}
      {...props}
    >
      {children}
    </div>
  );
}
