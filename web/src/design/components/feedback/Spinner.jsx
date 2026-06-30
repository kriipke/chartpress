import React from "react";

/** chartpress Spinner — circular loading indicator. Inherits color via currentColor. */
export function Spinner({ size = 16, thickness = 2, color = "currentColor", style = {}, ...props }) {
  return (
    <span
      role="status"
      aria-label="Loading"
      style={{
        display: "inline-block",
        width: size,
        height: size,
        borderRadius: "50%",
        border: `${thickness}px solid ${color}`,
        borderTopColor: "transparent",
        opacity: 0.9,
        animation: "cp-spin .7s linear infinite",
        verticalAlign: "middle",
        ...style,
      }}
      {...props}
    />
  );
}
