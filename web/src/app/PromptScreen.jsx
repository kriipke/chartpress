// Prompt screen — collect app name + description, then draft a spec via the
// real /text-to-config endpoint (server-side LLM). The drafted spec pre-fills
// the Rich form for review. States: idle · drafting (spinner, disabled) · error
// (server message kept verbatim, input preserved). The typed name overrides the
// model's suggested umbrella name.
import React from "react";
import { Card, Field, Input, Textarea, Button, InlineError, Spinner } from "../design/components";
import { ArrowLeft, Sparkle } from "./Icons.jsx";
import { isKebab, normalizeSpec } from "./spec.js";
import { draftFromPrompt } from "./api.js";

export function PromptScreen({ onBack, onDrafted }) {
  const [name, setName] = React.useState("");
  const [desc, setDesc] = React.useState("");
  const [status, setStatus] = React.useState("idle"); // idle | drafting | error
  const [error, setError] = React.useState("");

  const nameError = name && !isKebab(name) ? "Use kebab-case: lowercase letters, digits, hyphens." : "";
  const drafting = status === "drafting";

  const draft = async () => {
    if (nameError || !desc.trim()) return;
    setStatus("drafting");
    setError("");
    try {
      const drafted = await draftFromPrompt(desc.trim());
      const spec = normalizeSpec(drafted);
      // The typed name overrides the model's suggestion (design contract).
      if (name.trim()) spec.umbrellaChartName = name.trim();
      setStatus("idle");
      onDrafted(spec);
    } catch (err) {
      setStatus("error");
      setError((err && err.message) || "Drafting failed. Your input was kept — try again.");
    }
  };

  return (
    <div style={{ maxWidth: 620, margin: "0 auto" }}>
      <button onClick={onBack} disabled={drafting} style={backBtn}>
        <ArrowLeft size={15} /> Back
      </button>
      <Card title="Describe your app" subtitle="We'll draft an umbrella-chart spec from your description for you to review and edit." padding={28}>
        <div style={{ display: "flex", flexDirection: "column", gap: 20, marginTop: 4 }}>
          <Field label="App name" hint="Becomes the chart name and overrides the model's suggestion. kebab-case." error={nameError}>
            <Input mono placeholder="my-platform" value={name} disabled={drafting}
                   invalid={!!nameError} onChange={(e) => setName(e.target.value)} />
          </Field>
          <Field label="Describe your app">
            <Textarea rows={6} disabled={drafting}
              placeholder="e.g. A SaaS platform with a REST API, a background worker, a Redis cache, and a Postgres database. Use nginx ingress."
              value={desc} onChange={(e) => setDesc(e.target.value)} />
          </Field>

          {drafting && (
            <div style={draftingBox}>
              <Spinner size={16} color="var(--accent-9)" />
              <span>Drafting your chart…</span>
            </div>
          )}
          {status === "error" && <InlineError>{error}</InlineError>}

          <Button fullWidth size="3" onClick={draft} loading={drafting}
                  leadingIcon={!drafting ? <Sparkle size={16} /> : null} disabled={!desc.trim() && !drafting}>
            {drafting ? "Drafting…" : "Draft my chart"}
          </Button>
        </div>
      </Card>
    </div>
  );
}

const backBtn = {
  display: "inline-flex", alignItems: "center", gap: 6, marginBottom: 14, padding: "6px 10px 6px 6px",
  background: "transparent", border: "none", color: "var(--text-2)", fontFamily: "var(--font-sans)",
  fontSize: 13, fontWeight: 500, cursor: "pointer", borderRadius: "var(--radius-3)",
};
const draftingBox = {
  display: "flex", alignItems: "center", gap: 10, padding: "12px 14px",
  background: "var(--accent-3)", border: "1px solid var(--accent-6)", borderRadius: "var(--radius-input)",
  color: "var(--accent-11)", fontSize: 13, fontWeight: 500,
};
