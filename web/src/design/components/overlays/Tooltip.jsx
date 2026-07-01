import React from "react";

/**
 * chartpress Tooltip — a terse hover/focus hint. Dark bubble, short open delay,
 * four sides. Wrap the trigger element as children; keep copy to a phrase.
 */
export function Tooltip({ content, side = "top", delay = 120, maxWidth = 240, children, ...props }) {
  const [open, setOpen] = React.useState(false);
  const timer = React.useRef();

  const show = () => { clearTimeout(timer.current); timer.current = setTimeout(() => setOpen(true), delay); };
  const hide = () => { clearTimeout(timer.current); setOpen(false); };
  React.useEffect(() => () => clearTimeout(timer.current), []);

  const pos = {
    top: { bottom: "100%", left: "50%", transform: "translateX(-50%)", marginBottom: 8 },
    bottom: { top: "100%", left: "50%", transform: "translateX(-50%)", marginTop: 8 },
    left: { right: "100%", top: "50%", transform: "translateY(-50%)", marginRight: 8 },
    right: { left: "100%", top: "50%", transform: "translateY(-50%)", marginLeft: 8 },
  }[side] || {};

  return (
    <span
      style={{ position: "relative", display: "inline-flex" }}
      onMouseEnter={show}
      onMouseLeave={hide}
      onFocus={show}
      onBlur={hide}
      {...props}
    >
      {children}
      {open && content != null && (
        <span
          role="tooltip"
          style={{
            position: "absolute",
            zIndex: 50,
            ...pos,
            maxWidth,
            width: "max-content",
            background: "var(--slate-12)",
            color: "#fff",
            fontFamily: "var(--font-sans)",
            fontSize: 12,
            fontWeight: 500,
            lineHeight: 1.4,
            textAlign: "left",
            padding: "6px 9px",
            borderRadius: "var(--radius-3)",
            boxShadow: "var(--shadow-3)",
            pointerEvents: "none",
            whiteSpace: "normal",
          }}
        >
          {content}
        </span>
      )}
    </span>
  );
}
