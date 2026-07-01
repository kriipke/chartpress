import React from "react";

/**
 * chartpress Dialog — a modal centered over a dim, lightly blurred backdrop.
 * Escape and backdrop-click close it; body scroll is locked while open. Title,
 * description, body (children), and footer slots. Render it unconditionally and
 * drive it with the open prop.
 */
export function Dialog({
  open,
  onClose,
  title,
  description,
  footer,
  size = "2",
  closeOnBackdrop = true,
  children,
  ...props
}) {
  React.useEffect(() => {
    if (!open) return;
    const onKey = (e) => { if (e.key === "Escape" && onClose) onClose(); };
    document.addEventListener("keydown", onKey);
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = prev;
    };
  }, [open, onClose]);

  if (!open) return null;

  const maxWidth = { "1": 400, "2": 520, "3": 680 }[size] || 520;

  return (
    <div
      onMouseDown={(e) => { if (closeOnBackdrop && e.target === e.currentTarget && onClose) onClose(); }}
      style={{
        position: "fixed",
        inset: 0,
        zIndex: 100,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: 24,
        background: "rgba(28, 32, 36, 0.45)",
        backdropFilter: "blur(2px)",
        WebkitBackdropFilter: "blur(2px)",
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        style={{
          width: "100%",
          maxWidth,
          maxHeight: "calc(100vh - 48px)",
          overflow: "auto",
          background: "var(--surface-card)",
          border: "1px solid var(--border-subtle)",
          borderRadius: "var(--radius-card)",
          boxShadow: "var(--shadow-5)",
          fontFamily: "var(--font-sans)",
          color: "var(--text-1)",
          boxSizing: "border-box",
        }}
        {...props}
      >
        {(title || description || onClose) && (
          <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 16, padding: "20px 22px 0" }}>
            <div>
              {title && <div style={{ fontSize: 18, fontWeight: 700, letterSpacing: "-0.01em" }}>{title}</div>}
              {description && <div style={{ fontSize: 13.5, color: "var(--text-2)", marginTop: 4, lineHeight: 1.5 }}>{description}</div>}
            </div>
            {onClose && (
              <button
                type="button"
                onClick={onClose}
                aria-label="Close"
                style={{
                  border: "none",
                  background: "transparent",
                  cursor: "pointer",
                  fontSize: 22,
                  lineHeight: 1,
                  color: "var(--text-muted)",
                  width: 28,
                  height: 28,
                  borderRadius: "var(--radius-3)",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  flexShrink: 0,
                }}
              >
                ×
              </button>
            )}
          </div>
        )}
        <div style={{ padding: "16px 22px 20px", fontSize: 14, lineHeight: 1.55, color: "var(--text-2)" }}>
          {children}
        </div>
        {footer && (
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 10, padding: "0 22px 20px" }}>
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}
