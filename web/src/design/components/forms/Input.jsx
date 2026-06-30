import React from "react";

/**
 * chartpress Input — single-line text field. Brand focus ring (violet halo +
 * accent border). Set invalid for kebab-case / validation errors. Optional
 * leading/trailing adornments and monospace mode for chart names.
 */
export function Input({
  size = "2",
  invalid = false,
  mono = false,
  disabled = false,
  leading = null,
  trailing = null,
  style = {},
  wrapperStyle = {},
  ...props
}) {
  const sizes = {
    "1": { height: 30, padding: "0 9px", fontSize: "var(--font-size-1)" },
    "2": { height: 38, padding: "0 11px", fontSize: "var(--font-size-2)" },
    "3": { height: 44, padding: "0 13px", fontSize: "var(--font-size-3)" },
  };
  const s = sizes[size] || sizes["2"];
  const [focus, setFocus] = React.useState(false);

  const borderColor = invalid
    ? "var(--red-7)"
    : focus
    ? "var(--accent-solid)"
    : "var(--border-input)";
  const ring = invalid
    ? "0 0 0 3px rgba(229,72,77,.15)"
    : focus
    ? "var(--ring-focus)"
    : "none";

  const wrap = {
    display: "flex",
    alignItems: "center",
    gap: 8,
    height: s.height,
    padding: s.padding,
    background: disabled ? "var(--surface-sunken)" : "var(--surface-card)",
    border: `1px solid ${borderColor}`,
    borderRadius: "var(--radius-input)",
    boxShadow: ring,
    transition: "border-color .15s ease, box-shadow .15s ease",
    boxSizing: "border-box",
    cursor: disabled ? "not-allowed" : "text",
    opacity: disabled ? 0.6 : 1,
    ...wrapperStyle,
  };

  const input = {
    flex: 1,
    minWidth: 0,
    height: "100%",
    border: "none",
    outline: "none",
    background: "transparent",
    fontFamily: mono ? "var(--font-mono)" : "var(--font-sans)",
    fontSize: s.fontSize,
    color: "var(--text-1)",
    padding: 0,
    ...style,
  };

  const adorn = { display: "inline-flex", alignItems: "center", color: "var(--text-muted)", flexShrink: 0 };

  return (
    <div style={wrap}>
      {leading ? <span style={adorn}>{leading}</span> : null}
      <input
        disabled={disabled}
        aria-invalid={invalid || undefined}
        onFocus={(e) => { setFocus(true); props.onFocus?.(e); }}
        onBlur={(e) => { setFocus(false); props.onBlur?.(e); }}
        style={input}
        {...props}
      />
      {trailing ? <span style={adorn}>{trailing}</span> : null}
    </div>
  );
}
