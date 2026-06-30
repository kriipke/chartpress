import React from "react";

/**
 * chartpress ChoiceCard — a large, equal-weight selectable card for the "Choose"
 * fork (Generate manually vs from a prompt). Icon, title, description, and a
 * selected/hover state with the brand accent.
 */
export function ChoiceCard({
  icon,
  title,
  description,
  selected = false,
  onClick,
  badge,
  style = {},
  ...props
}) {
  const [hover, setHover] = React.useState(false);
  const active = selected || hover;

  return (
    <button
      type="button"
      onClick={onClick}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: "flex",
        flexDirection: "column",
        alignItems: "flex-start",
        gap: 10,
        textAlign: "left",
        padding: 24,
        width: "100%",
        background: "var(--surface-card)",
        border: `1px solid ${active ? "var(--accent-solid)" : "var(--border-default)"}`,
        borderRadius: "var(--radius-card)",
        boxShadow: selected ? "var(--ring-focus)" : hover ? "var(--shadow-accent)" : "none",
        cursor: "pointer",
        transition: "border-color .18s var(--ease-standard), box-shadow .2s var(--ease-standard)",
        boxSizing: "border-box",
        ...style,
      }}
      {...props}
    >
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", width: "100%" }}>
        {icon && (
          <span
            style={{
              display: "inline-flex", alignItems: "center", justifyContent: "center",
              width: 44, height: 44, borderRadius: "var(--radius-5)",
              background: active ? "var(--violet-9)" : "var(--violet-3)",
              color: active ? "#fff" : "var(--violet-11)",
              transition: "background-color .18s var(--ease-standard), color .18s var(--ease-standard)",
            }}
          >
            {icon}
          </span>
        )}
        {badge}
      </div>
      <div style={{ fontFamily: "var(--font-sans)", fontSize: "var(--font-size-4)", fontWeight: "var(--font-weight-bold)", color: "var(--text-1)" }}>
        {title}
      </div>
      {description && (
        <div style={{ fontFamily: "var(--font-sans)", fontSize: "var(--font-size-2)", color: "var(--text-2)", lineHeight: 1.5 }}>
          {description}
        </div>
      )}
    </button>
  );
}
