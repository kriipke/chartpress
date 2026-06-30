import React from "react";

/**
 * chartpress Textarea — multi-line input for the prompt "Describe your app"
 * field. Same focus treatment as Input; resizable vertically.
 */
export function Textarea({
  invalid = false,
  disabled = false,
  rows = 5,
  style = {},
  ...props
}) {
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

  return (
    <textarea
      rows={rows}
      disabled={disabled}
      aria-invalid={invalid || undefined}
      onFocus={(e) => { setFocus(true); props.onFocus?.(e); }}
      onBlur={(e) => { setFocus(false); props.onBlur?.(e); }}
      style={{
        width: "100%",
        padding: "10px 12px",
        fontFamily: "var(--font-sans)",
        fontSize: "var(--font-size-2)",
        lineHeight: "var(--line-height-3)",
        color: "var(--text-1)",
        background: disabled ? "var(--surface-sunken)" : "var(--surface-card)",
        border: `1px solid ${borderColor}`,
        borderRadius: "var(--radius-input)",
        boxShadow: ring,
        outline: "none",
        resize: "vertical",
        minHeight: 88,
        transition: "border-color .15s ease, box-shadow .15s ease",
        boxSizing: "border-box",
        opacity: disabled ? 0.6 : 1,
        ...style,
      }}
      {...props}
    />
  );
}
