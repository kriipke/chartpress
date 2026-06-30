import React from "react";

/**
 * chartpress IconButton — a square, icon-only control. Used for the subchart
 * remove "×" and other compact toolbar actions.
 */
export function IconButton({
  children,
  variant = "outline",
  size = "2",
  color = "neutral",
  disabled = false,
  label,
  style = {},
  ...props
}) {
  const dims = { "1": 28, "2": 36, "3": 44 }[size] || 36;
  const [hover, setHover] = React.useState(false);
  const [focus, setFocus] = React.useState(false);

  const isDanger = color === "danger";
  const base = {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: dims,
    height: dims,
    padding: 0,
    fontFamily: "var(--font-sans)",
    fontSize: dims < 32 ? 16 : 18,
    lineHeight: 1,
    borderRadius: "var(--radius-input)",
    border: "1px solid transparent",
    background: "transparent",
    color: "var(--text-muted)",
    cursor: disabled ? "not-allowed" : "pointer",
    transition: "background-color .15s ease, border-color .15s ease, color .15s ease, box-shadow .15s ease",
    opacity: disabled ? 0.4 : 1,
    boxSizing: "border-box",
  };

  const variants = {
    outline: { borderColor: "var(--border-input)", background: "var(--surface-card)" },
    ghost: { borderColor: "transparent", background: "transparent" },
    soft: { borderColor: "transparent", background: "var(--surface-sunken)" },
  };

  const hoverStyle = !disabled && hover ? (
    isDanger
      ? { borderColor: "var(--red-7)", color: "var(--red-11)" }
      : { borderColor: "var(--accent-border)", color: "var(--text-accent)" }
  ) : {};

  return (
    <button
      type="button"
      aria-label={label}
      disabled={disabled}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      onFocus={() => setFocus(true)}
      onBlur={() => setFocus(false)}
      style={{
        ...base,
        ...variants[variant],
        ...hoverStyle,
        ...(focus ? { boxShadow: "var(--ring-focus)" } : {}),
        ...style,
      }}
      {...props}
    >
      {children}
    </button>
  );
}
