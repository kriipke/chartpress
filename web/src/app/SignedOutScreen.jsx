// SignedOutScreen — the branded landing shown to signed-out users when GitHub
// sign-in is configured. Centered sign-in prompt on the signature dotted
// pattern. `onSignIn` kicks off the OAuth redirect. `onBrowse` (optional) lets
// the user continue into the app without signing in, preserving the open,
// non-gating behaviour when a deployer wants it.
import React from "react";
import { SignInButton } from "../design/components";
import logoUrl from "../assets/logo-blue.png";

export function SignedOutScreen({ onSignIn, onBrowse }) {
  return (
    <div style={{
      minHeight: "calc(100vh - 65px)", display: "flex", flexDirection: "column",
      alignItems: "center", justifyContent: "center", textAlign: "center", padding: 40,
      backgroundImage: "var(--pattern-dots)", backgroundSize: "var(--pattern-dot-size) var(--pattern-dot-size)",
    }}>
      <img src={logoUrl} alt="chartpress" width="72" height="72" style={{ objectFit: "contain", marginBottom: 20 }} />
      <h1 style={{ margin: "0 0 8px", fontSize: 28, fontWeight: 700, letterSpacing: "-0.02em", color: "var(--text-1)" }}>
        chart<span style={{ color: "var(--accent-9)" }}>press</span>
      </h1>
      <p style={{ margin: "0 0 26px", fontSize: 15, color: "var(--text-2)", maxWidth: 380, lineHeight: 1.55 }}>
        Turn a structured spec into a downloadable Helm chart bundle. Sign in to generate and manage your charts.
      </p>
      <SignInButton onClick={onSignIn} />
      {onBrowse && (
        <button type="button" onClick={onBrowse} style={browseLink}>
          Continue without signing in
        </button>
      )}
    </div>
  );
}

const browseLink = {
  marginTop: 16, background: "transparent", border: "none", color: "var(--text-2)",
  cursor: "pointer", fontSize: 13.5, fontWeight: 500, fontFamily: "var(--font-sans)",
  textDecoration: "underline", textUnderlineOffset: 2, padding: 4,
};
