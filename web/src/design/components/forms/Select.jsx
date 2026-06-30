import React from "react";

/**
 * chartpress Select — styled native <select> with a chevron. Used for the
 * subchart workload (3 options) and ingress (6 options) dropdowns. Native for
 * accessibility + correct mobile behavior.
 */
export function Select({
  options = [],
  size = "2",
  invalid = false,
  disabled = false,
  style = {},
  ...props
}) {
  const sizes = {
    "1": { height: 30, padding: "0 30px 0 9px", fontSize: "var(--font-size-1)" },
    "2": { height: 38, padding: "0 32px 0 11px", fontSize: "var(--font-size-2)" },
    "3": { height: 44, padding: "0 36px 0 13px", fontSize: "var(--font-size-3)" },
  };
  const s = sizes[size] || sizes["2"];
  const [focus, setFocus] = React.useState(false);

  const borderColor = invalid
    ? "var(--red-7)"
    : focus
    ? "var(--accent-solid)"
    : "var(--border-input)";

  // chevron drawn as an inline SVG data-uri so the component is self-contained
  const chevron =
    "url(\"data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='%2360646c' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><path d='m6 9 6 6 6-6'/></svg>\")";

  return (
    <select
      disabled={disabled}
      aria-invalid={invalid || undefined}
      onFocus={() => setFocus(true)}
      onBlur={() => setFocus(false)}
      style={{
        height: s.height,
        padding: s.padding,
        width: "100%",
        appearance: "none",
        WebkitAppearance: "none",
        MozAppearance: "none",
        fontFamily: "var(--font-sans)",
        fontSize: s.fontSize,
        color: "var(--text-1)",
        background: `${disabled ? "var(--surface-sunken)" : "var(--surface-card)"} ${chevron} no-repeat right 10px center`,
        border: `1px solid ${borderColor}`,
        borderRadius: "var(--radius-input)",
        boxShadow: focus && !invalid ? "var(--ring-focus)" : invalid && focus ? "0 0 0 3px rgba(229,72,77,.15)" : "none",
        outline: "none",
        cursor: disabled ? "not-allowed" : "pointer",
        transition: "border-color .15s ease, box-shadow .15s ease",
        boxSizing: "border-box",
        opacity: disabled ? 0.6 : 1,
        ...style,
      }}
      {...props}
    >
      {options.map((o) => {
        const value = typeof o === "string" ? o : o.value;
        const label = typeof o === "string" ? o : o.label;
        return <option key={value} value={value}>{label}</option>;
      })}
    </select>
  );
}
