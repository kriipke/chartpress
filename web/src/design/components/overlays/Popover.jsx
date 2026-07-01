import React from "react";

/**
 * chartpress Popover — a click-triggered floating panel anchored to its trigger.
 * White card, hairline border, floating shadow. Closes on outside click or
 * Escape. Controlled via open/onOpenChange, or uncontrolled via defaultOpen.
 */
export function Popover({
  trigger,
  children,
  side = "bottom",
  align = "start",
  open: openProp,
  defaultOpen = false,
  onOpenChange,
  width = 280,
  ...props
}) {
  const [openState, setOpenState] = React.useState(defaultOpen);
  const isControlled = openProp !== undefined;
  const open = isControlled ? openProp : openState;
  const ref = React.useRef();

  const setOpen = (v) => {
    if (!isControlled) setOpenState(v);
    if (onOpenChange) onOpenChange(v);
  };

  React.useEffect(() => {
    if (!open) return;
    const onDoc = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    const onKey = (e) => { if (e.key === "Escape") setOpen(false); };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const vertical = side === "top"
    ? { bottom: "100%", marginBottom: 8 }
    : { top: "100%", marginTop: 8 };
  const horizontal = align === "end" ? { right: 0 } : { left: 0 };

  return (
    <span ref={ref} style={{ position: "relative", display: "inline-flex" }} {...props}>
      <span onClick={() => setOpen(!open)} style={{ display: "inline-flex" }}>
        {trigger}
      </span>
      {open && (
        <div
          role="dialog"
          style={{
            position: "absolute",
            zIndex: 60,
            width,
            ...vertical,
            ...horizontal,
            background: "var(--surface-card)",
            border: "1px solid var(--border-default)",
            borderRadius: "var(--radius-card)",
            boxShadow: "var(--shadow-4)",
            padding: 14,
            fontFamily: "var(--font-sans)",
            fontSize: 14,
            lineHeight: 1.5,
            color: "var(--text-1)",
            boxSizing: "border-box",
          }}
        >
          {children}
        </div>
      )}
    </span>
  );
}
