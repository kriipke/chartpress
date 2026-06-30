import React from "react";

/**
 * chartpress Card — the white panel surface. 1px hairline border, 12px radius,
 * 28px padding by default. Optional title/subtitle header. variant="sunken"
 * uses the faint gray well; elevation adds a resting shadow.
 */
export function Card({
  children,
  title,
  subtitle,
  header,
  footer,
  variant = "surface",
  elevation = 0,
  padding = 28,
  interactive = false,
  style = {},
  ...props
}) {
  const [hover, setHover] = React.useState(false);
  const shadows = [null, "var(--shadow-1)", "var(--shadow-2)", "var(--shadow-3)", "var(--shadow-4)"];

  const bg = variant === "sunken" ? "var(--surface-sunken)" : "var(--surface-card)";

  return (
    <div
      onMouseEnter={() => interactive && setHover(true)}
      onMouseLeave={() => interactive && setHover(false)}
      style={{
        background: bg,
        border: "1px solid var(--border-default)",
        borderRadius: "var(--radius-card)",
        boxShadow: interactive && hover ? "var(--shadow-accent)" : shadows[elevation] || "none",
        transition: "box-shadow .2s var(--ease-standard)",
        boxSizing: "border-box",
        ...style,
      }}
      {...props}
    >
      {(title || subtitle || header) && (
        <div style={{ padding: `${padding}px ${padding}px 0` }}>
          {header || (
            <>
              {title && <div style={{ fontFamily: "var(--font-sans)", fontSize: "var(--font-size-6)", fontWeight: "var(--font-weight-bold)", color: "var(--text-1)", letterSpacing: "var(--letter-spacing-tight)", margin: 0 }}>{title}</div>}
              {subtitle && <div style={{ fontFamily: "var(--font-sans)", fontSize: "var(--font-size-2)", color: "var(--text-2)", marginTop: 4 }}>{subtitle}</div>}
            </>
          )}
        </div>
      )}
      <div style={{ padding }}>{children}</div>
      {footer && <div style={{ padding: `0 ${padding}px ${padding}px` }}>{footer}</div>}
    </div>
  );
}
