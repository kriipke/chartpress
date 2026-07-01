import React from "react";

/**
 * chartpress EmptyState — centered icon + title + description + action. Used for
 * the empty Charts list ("No charts yet → Generate one").
 */
export function EmptyState({ icon, title, description, action, style = {} }) {
  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        textAlign: "center",
        gap: 6,
        padding: "56px 28px",
        ...style,
      }}
    >
      {icon && (
        <div
          style={{
            display: "inline-flex",
            alignItems: "center",
            justifyContent: "center",
            width: 48,
            height: 48,
            marginBottom: 8,
            borderRadius: "var(--radius-5)",
            background: "var(--accent-3)",
            color: "var(--accent-11)",
          }}
        >
          {icon}
        </div>
      )}
      <div style={{ fontFamily: "var(--font-sans)", fontSize: "var(--font-size-4)", fontWeight: "var(--font-weight-bold)", color: "var(--text-1)" }}>
        {title}
      </div>
      {description && (
        <div style={{ fontFamily: "var(--font-sans)", fontSize: "var(--font-size-2)", color: "var(--text-2)", maxWidth: 380, lineHeight: 1.5 }}>
          {description}
        </div>
      )}
      {action && <div style={{ marginTop: 12 }}>{action}</div>}
    </div>
  );
}
