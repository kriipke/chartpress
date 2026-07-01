import React from "react";

/**
 * chartpress Checkbox — used for the 8 rule toggles. A controlled box with a
 * label and optional helper/tooltip description. Brand azure when checked.
 */
export function Checkbox({
  checked = false,
  onChange,
  label,
  description,
  disabled = false,
  id,
  style = {},
  ...props
}) {
  const [hover, setHover] = React.useState(false);
  const [focus, setFocus] = React.useState(false);
  const autoId = React.useId ? React.useId() : "cp-cb";
  const cbId = id || autoId;

  const box = {
    width: 18,
    height: 18,
    flexShrink: 0,
    marginTop: 1,
    borderRadius: "var(--radius-2)",
    border: `1px solid ${checked ? "var(--accent-solid)" : hover ? "var(--accent-border)" : "var(--border-strong)"}`,
    background: checked ? "var(--accent-solid)" : "var(--surface-card)",
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    transition: "background-color .15s ease, border-color .15s ease, box-shadow .15s ease",
    boxShadow: focus ? "var(--ring-focus)" : "none",
    cursor: disabled ? "not-allowed" : "pointer",
  };

  return (
    <label
      htmlFor={cbId}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: "flex",
        gap: 10,
        alignItems: "flex-start",
        cursor: disabled ? "not-allowed" : "pointer",
        opacity: disabled ? 0.55 : 1,
        ...style,
      }}
    >
      <input
        id={cbId}
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange?.(e.target.checked, e)}
        onFocus={() => setFocus(true)}
        onBlur={() => setFocus(false)}
        style={{ position: "absolute", opacity: 0, width: 1, height: 1, margin: 0 }}
        {...props}
      />
      <span aria-hidden="true" style={box}>
        {checked && (
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
            <path d="M20 6 9 17l-5-5" />
          </svg>
        )}
      </span>
      {(label || description) && (
        <span style={{ display: "flex", flexDirection: "column", gap: 2, minWidth: 0 }}>
          {label && (
            <span style={{ fontFamily: "var(--font-sans)", fontSize: "var(--font-size-2)", fontWeight: "var(--font-weight-medium)", color: "var(--text-1)", lineHeight: 1.35 }}>
              {label}
            </span>
          )}
          {description && (
            <span style={{ fontFamily: "var(--font-sans)", fontSize: "var(--font-size-1)", color: "var(--text-muted)", lineHeight: 1.45 }}>
              {description}
            </span>
          )}
        </span>
      )}
    </label>
  );
}
