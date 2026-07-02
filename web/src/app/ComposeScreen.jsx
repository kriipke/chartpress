// Compose screen — the third Generate entry: import a docker-compose file used
// in development. The server maps it deterministically (no LLM) to a Spec via
// /compose-to-config, returning {spec, notes}. On success we show a short
// interstitial summary + the notes ("we weren't sure / renamed / dropped"), then
// the user proceeds into the same Rich form the other two options land in. The
// typed App name overrides the compose top-level name (same contract as Prompt).
// States: idle · parsing (spinner) · error (server message verbatim, input kept)
// · result (summary + notes before Review & edit).
import React from "react";
import { Card, Field, Input, Textarea, Button, InlineError, Spinner } from "../design/components";
import { ArrowLeft, ArrowRight, Package, Check } from "./Icons.jsx";
import { isKebab, normalizeSpec } from "./spec.js";
import { draftFromCompose } from "./api.js";

export function ComposeScreen({ onBack, onDrafted }) {
  const [name, setName] = React.useState("");
  const [text, setText] = React.useState("");
  const [fileName, setFileName] = React.useState("");
  const [status, setStatus] = React.useState("idle"); // idle | parsing | error | result
  const [error, setError] = React.useState("");
  const [result, setResult] = React.useState(null); // { spec, notes, counts }
  const [dragging, setDragging] = React.useState(false);
  const fileInput = React.useRef(null);

  const nameError = name && !isKebab(name) ? "Use kebab-case: lowercase letters, digits, hyphens." : "";
  const parsing = status === "parsing";

  const readFile = (file) => {
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => { setText(String(reader.result || "")); setFileName(file.name); };
    reader.readAsText(file);
  };

  const onDrop = (e) => {
    e.preventDefault();
    setDragging(false);
    if (parsing) return;
    readFile(e.dataTransfer.files && e.dataTransfer.files[0]);
  };

  const draft = async () => {
    if (nameError || !text.trim()) return;
    setStatus("parsing");
    setError("");
    try {
      const res = await draftFromCompose(text.trim());
      // Envelope { spec, notes }; tolerate a bare spec just in case.
      const spec = res && res.spec ? res.spec : res;
      const notes = res && Array.isArray(res.notes) ? res.notes : [];
      setResult({
        spec,
        notes,
        counts: {
          subcharts: (spec.subcharts || []).length,
          dependencies: (spec.dependencies || []).length,
        },
      });
      setStatus("result");
    } catch (err) {
      setStatus("error");
      setError((err && err.message) || "Import failed. Your input was kept — try again.");
    }
  };

  const proceed = () => {
    const spec = normalizeSpec(result.spec);
    // The typed name overrides the compose top-level name (design contract).
    if (name.trim()) spec.umbrellaChartName = name.trim();
    onDrafted(spec);
  };

  // Interstitial: what we imported + anything worth reviewing, before the form.
  if (status === "result" && result) {
    const { counts, notes } = result;
    return (
      <div style={{ maxWidth: 620, margin: "0 auto" }}>
        <button onClick={() => setStatus("idle")} style={backBtn}>
          <ArrowLeft size={15} /> Back
        </button>
        <Card title="Imported your compose file" subtitle="Review what we mapped, then edit everything in the form before generating." padding={28}>
          <div style={{ display: "flex", flexDirection: "column", gap: 18, marginTop: 4 }}>
            <div style={summaryBox}>
              <Check size={16} color="var(--accent-9)" />
              <span>
                <b>{counts.subcharts}</b> {counts.subcharts === 1 ? "subchart" : "subcharts"} · <b>{counts.dependencies}</b> {counts.dependencies === 1 ? "dependency" : "dependencies"}
              </span>
            </div>

            {notes.length > 0 && (
              <div>
                <div style={{ fontSize: 12.5, fontWeight: 600, color: "var(--text-muted)", marginBottom: 8 }}>
                  {notes.length} thing{notes.length === 1 ? "" : "s"} to review
                </div>
                <ul style={notesList}>
                  {notes.map((n, i) => (
                    <li key={i} style={noteItem}>{n}</li>
                  ))}
                </ul>
              </div>
            )}

            <Button fullWidth size="3" onClick={proceed} leadingIcon={<ArrowRight size={16} />}>
              Review &amp; edit
            </Button>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 620, margin: "0 auto" }}>
      <button onClick={onBack} disabled={parsing} style={backBtn}>
        <ArrowLeft size={15} /> Back
      </button>
      <Card title="Import a docker-compose file" subtitle="Paste or drop the docker-compose.yaml you use for local dev. We'll map each service to a subchart or a dependency for you to review and edit." padding={28}>
        <div style={{ display: "flex", flexDirection: "column", gap: 20, marginTop: 4 }}>
          <Field label="App name" hint="Becomes the chart name and overrides the compose file's top-level name. kebab-case." error={nameError}>
            <Input mono placeholder="my-platform" value={name} disabled={parsing}
                   invalid={!!nameError} onChange={(e) => setName(e.target.value)} />
          </Field>

          <Field label="docker-compose.yaml">
            <div
              onDragOver={(e) => { e.preventDefault(); if (!parsing) setDragging(true); }}
              onDragLeave={() => setDragging(false)}
              onDrop={onDrop}
              style={{
                border: "1px dashed " + (dragging ? "var(--accent-7)" : "var(--border-default)"),
                background: dragging ? "var(--accent-3)" : "transparent",
                borderRadius: "var(--radius-input)", padding: 8, transition: "background-color .13s, border-color .13s",
              }}
            >
              <Textarea rows={12} disabled={parsing}
                placeholder={"Drop a file here, or paste your compose YAML…\n\nservices:\n  api:\n    build: .\n    ports:\n      - \"8080:8080\"\n  db:\n    image: postgres:16"}
                value={text} onChange={(e) => { setText(e.target.value); setFileName(""); }}
                style={{ fontFamily: "var(--font-mono)", border: "none", boxShadow: "none", background: "transparent", minHeight: 200 }} />
              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", padding: "2px 4px 0" }}>
                <button type="button" onClick={() => fileInput.current && fileInput.current.click()} disabled={parsing} style={chooseBtn}>
                  Choose file…
                </button>
                {fileName && <span style={{ fontSize: 12, color: "var(--text-muted)" }}>{fileName}</span>}
              </div>
              <input ref={fileInput} type="file" accept=".yml,.yaml,text/yaml,application/x-yaml" style={{ display: "none" }}
                     onChange={(e) => { readFile(e.target.files && e.target.files[0]); e.target.value = ""; }} />
            </div>
          </Field>

          {parsing && (
            <div style={parsingBox}>
              <Spinner size={16} color="var(--accent-9)" />
              <span>Mapping your services…</span>
            </div>
          )}
          {status === "error" && <InlineError>{error}</InlineError>}

          <Button fullWidth size="3" onClick={draft} loading={parsing}
                  leadingIcon={!parsing ? <Package size={16} /> : null} disabled={!text.trim() && !parsing}>
            {parsing ? "Mapping…" : "Import compose file"}
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
const parsingBox = {
  display: "flex", alignItems: "center", gap: 10, padding: "12px 14px",
  background: "var(--accent-3)", border: "1px solid var(--accent-6)", borderRadius: "var(--radius-input)",
  color: "var(--accent-11)", fontSize: 13, fontWeight: 500,
};
const summaryBox = {
  display: "flex", alignItems: "center", gap: 10, padding: "12px 14px",
  background: "var(--accent-3)", border: "1px solid var(--accent-6)", borderRadius: "var(--radius-input)",
  color: "var(--accent-11)", fontSize: 13.5,
};
const notesList = { listStyle: "none", margin: 0, padding: 0, display: "flex", flexDirection: "column", gap: 8 };
const noteItem = {
  fontSize: 12.5, lineHeight: 1.5, color: "var(--text-2)", padding: "9px 12px",
  background: "var(--surface-sunken)", border: "1px solid var(--border-default)", borderRadius: "var(--radius-input)",
};
const chooseBtn = {
  background: "transparent", border: "none", color: "var(--accent-11)", fontFamily: "var(--font-sans)",
  fontSize: 12.5, fontWeight: 600, cursor: "pointer", padding: "4px 2px",
};
