// Choose screen — the Generate wizard entry fork: manual vs from a prompt.
import React from "react";
import { ChoiceCard } from "../design/components";
import { Form, Sparkle } from "./Icons.jsx";

export function ChooseScreen({ onManual, onPrompt }) {
  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <div style={{ textAlign: "center", marginBottom: 32 }}>
        <h1 style={{ margin: "0 0 6px", fontSize: 28, fontWeight: 700, color: "var(--text-1)", letterSpacing: "-0.01em" }}>
          Generate a Helm chart
        </h1>
        <p style={{ margin: 0, fontSize: 15, color: "var(--text-2)" }}>
          Choose how you'd like to define your umbrella chart.
        </p>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
        <ChoiceCard
          icon={<Form size={22} />}
          title="Generate manually"
          description="Fill in the spec yourself — umbrella, subcharts, and rules."
          onClick={onManual}
        />
        <ChoiceCard
          icon={<Sparkle size={22} />}
          title="Generate from a prompt"
          description="Describe your app in plain language; we draft the spec for you to review."
          onClick={onPrompt}
        />
      </div>
    </div>
  );
}
