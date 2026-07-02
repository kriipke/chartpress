// Choose screen — the Generate wizard entry fork: manual, from a prompt, or
// from a docker-compose file.
import React from "react";
import { ChoiceCard } from "../design/components";
import { Form, Sparkle, Package } from "./Icons.jsx";

export function ChooseScreen({ onManual, onPrompt, onCompose }) {
  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <div style={{ textAlign: "center", marginBottom: 32 }}>
        <h1 style={{ margin: "0 0 6px", fontSize: 28, fontWeight: 700, color: "var(--text-1)", letterSpacing: "-0.01em" }}>
          Generate a Helm chart
        </h1>
        <p style={{ margin: 0, fontSize: 15, color: "var(--text-2)" }}>
          Choose how you'd like to define your umbrella chart.
        </p>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(3, minmax(0, 1fr))", gap: 16 }}>
        <ChoiceCard
          icon={<Form size={22} />}
          title="Generate manually"
          description="You specify the structure yourself — name each subchart (the microservices that make up your app), pick a workload type for each, and set the generation rules."
          onClick={onManual}
        />
        <ChoiceCard
          icon={<Sparkle size={22} />}
          title="Generate from a prompt"
          description="Describe your app in plain English. We'll populate our best guess at the subcharts / microservices you'll need to build it — then you review and edit before generating."
          onClick={onPrompt}
        />
        <ChoiceCard
          icon={<Package size={22} />}
          title="Import a docker-compose file"
          description="Already have a docker-compose.yaml for local dev? We'll turn each service into a subchart or a dependency — then you review and edit before generating."
          onClick={onCompose}
        />
      </div>
    </div>
  );
}
