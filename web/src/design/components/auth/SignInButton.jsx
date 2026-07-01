import React from "react";

/**
 * chartpress SignInButton — the GitHub OAuth entry point. A solid dark button
 * carrying the GitHub mark; this is the signed-out state's primary action.
 * Provide onClick to kick off the OAuth redirect.
 */
export function SignInButton({ label = "Sign in with GitHub", onClick, disabled = false, style = {}, ...props }) {
  const [hover, setHover] = React.useState(false);
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 9,
        padding: "9px 16px",
        border: "none",
        borderRadius: "var(--radius-button)",
        background: disabled ? "var(--slate-8)" : hover ? "#000" : "var(--slate-12)",
        color: "#fff",
        fontFamily: "var(--font-sans)",
        fontSize: 14,
        fontWeight: 600,
        cursor: disabled ? "not-allowed" : "pointer",
        opacity: disabled ? 0.6 : 1,
        transition: "background-color .15s",
        ...style,
      }}
      {...props}
    >
      <GithubMark size={17} />
      {label}
    </button>
  );
}

/** GitHub's mark, filled — kept local so the component is self-contained. */
export function GithubMark({ size = 18, style = {} }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" style={{ display: "block", ...style }} aria-hidden="true">
      <path d="M12 .5C5.73.5.5 5.74.5 12.02c0 5.1 3.29 9.42 7.86 10.95.58.11.79-.25.79-.56 0-.28-.01-1.02-.02-2-3.2.7-3.88-1.54-3.88-1.54-.53-1.34-1.28-1.7-1.28-1.7-1.05-.72.08-.7.08-.7 1.16.08 1.77 1.2 1.77 1.2 1.03 1.77 2.7 1.26 3.36.96.1-.75.4-1.26.73-1.55-2.56-.29-5.25-1.28-5.25-5.7 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.46.11-3.05 0 0 .97-.31 3.18 1.18a11 11 0 0 1 5.8 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.59.24 2.76.12 3.05.74.81 1.18 1.84 1.18 3.1 0 4.43-2.69 5.41-5.26 5.69.41.36.78 1.06.78 2.14 0 1.55-.02 2.8-.02 3.18 0 .31.21.68.8.56A11.53 11.53 0 0 0 23.5 12.02C23.5 5.74 18.27.5 12 .5Z" />
    </svg>
  );
}
