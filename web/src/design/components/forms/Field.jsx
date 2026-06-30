import React from "react";

/**
 * chartpress Field — label + control + optional hint/error wrapper. Standardizes
 * the 13px semibold label and inline error treatment used across the spec form.
 */
export function Field({
  label,
  htmlFor,
  required = false,
  hint,
  error,
  children,
  style = {},
}) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, ...style }}>
      {label && (
        <label
          htmlFor={htmlFor}
          style={{
            fontFamily: "var(--font-sans)",
            fontSize: "var(--text-label-size)",
            fontWeight: "var(--text-label-weight)",
            color: "var(--text-1)",
          }}
        >
          {label}
          {required && <span style={{ color: "var(--red-11)", marginLeft: 3 }}>*</span>}
        </label>
      )}
      {children}
      {error ? (
        <span style={{ fontFamily: "var(--font-sans)", fontSize: "var(--font-size-1)", color: "var(--red-11)", lineHeight: 1.4 }}>
          {error}
        </span>
      ) : hint ? (
        <span style={{ fontFamily: "var(--font-sans)", fontSize: "var(--font-size-1)", color: "var(--text-muted)", lineHeight: 1.4 }}>
          {hint}
        </span>
      ) : null}
    </div>
  );
}
