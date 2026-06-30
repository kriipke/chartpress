import React from "react";

/**
 * chartpress Button — the primary action control.
 * Variants: solid (brand violet fill), outline (accent border), ghost (text),
 * soft (tinted fill). Sizes 1–3. Supports loading and full-width.
 */
export function Button({
  children,
  variant = "solid",
  size = "2",
  color = "accent",
  loading = false,
  disabled = false,
  fullWidth = false,
  leadingIcon = null,
  trailingIcon = null,
  type = "button",
  style = {},
  ...props
}) {
  const sizes = {
    "1": { height: 28, padding: "0 10px", fontSize: "var(--font-size-1)", radius: "var(--radius-3)", gap: 6 },
    "2": { height: 36, padding: "0 14px", fontSize: "var(--font-size-2)", radius: "var(--radius-button)", gap: 8 },
    "3": { height: 44, padding: "0 18px", fontSize: "var(--font-size-3)", radius: "var(--radius-button)", gap: 8 },
  };
  const s = sizes[size] || sizes["2"];

  const isDanger = color === "danger";
  const solidBg = isDanger ? "var(--red-9)" : "var(--accent-solid)";
  const solidBgHover = isDanger ? "var(--red-10)" : "var(--accent-solid-hover)";
  const lineColor = isDanger ? "var(--red-11)" : "var(--text-accent)";
  const tint = isDanger ? "var(--red-3)" : "var(--accent-tint)";
  const tintHover = isDanger ? "var(--red-4)" : "var(--accent-tint-hover)";
  const border = isDanger ? "var(--red-6)" : "var(--accent-border)";

  const base = {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    gap: s.gap,
    height: s.height,
    padding: s.padding,
    width: fullWidth ? "100%" : "auto",
    fontFamily: "var(--font-sans)",
    fontSize: s.fontSize,
    fontWeight: "var(--font-weight-semibold)",
    lineHeight: 1,
    borderRadius: s.radius,
    border: "1px solid transparent",
    cursor: disabled || loading ? "not-allowed" : "pointer",
    transition: "background-color .15s ease, border-color .15s ease, color .15s ease, box-shadow .15s ease",
    whiteSpace: "nowrap",
    userSelect: "none",
    opacity: disabled ? 0.55 : 1,
    boxSizing: "border-box",
  };

  const variants = {
    solid: { background: solidBg, color: "var(--text-on-accent)", borderColor: solidBg },
    outline: { background: "transparent", color: lineColor, borderColor: border },
    soft: { background: tint, color: lineColor, borderColor: "transparent" },
    ghost: { background: "transparent", color: lineColor, borderColor: "transparent" },
  };

  const [hover, setHover] = React.useState(false);
  const [focus, setFocus] = React.useState(false);

  const hoverStyle = !disabled && !loading && hover ? {
    solid: { background: solidBgHover, borderColor: solidBgHover },
    outline: { background: "var(--accent-tint)" },
    soft: { background: tintHover },
    ghost: { background: "var(--accent-tint)" },
  }[variant] : {};

  const focusStyle = focus ? { boxShadow: "var(--ring-focus)" } : {};

  return (
    <button
      type={type}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      onFocus={() => setFocus(true)}
      onBlur={() => setFocus(false)}
      style={{ ...base, ...variants[variant], ...hoverStyle, ...focusStyle, ...style }}
      {...props}
    >
      {loading ? <Spinner /> : leadingIcon}
      {children}
      {!loading && trailingIcon}
    </button>
  );
}

function Spinner() {
  return (
    <span
      aria-hidden="true"
      style={{
        width: "1em",
        height: "1em",
        borderRadius: "50%",
        border: "2px solid currentColor",
        borderTopColor: "transparent",
        display: "inline-block",
        animation: "cp-spin .7s linear infinite",
      }}
    />
  );
}
