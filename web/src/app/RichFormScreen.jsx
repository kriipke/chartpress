// Rich form — the core spec editor, shared by the manual (empty) and prompt
// (pre-filled) paths. Umbrella · subcharts · rules · a live structure preview.
// Submitting calls the real /generate; a server error surfaces inline (verbatim)
// and keeps the user on the form.
import React from "react";
import { Card, Field, Input, Select, Button, IconButton, Checkbox, StructurePreview, InlineError } from "../design/components";
import { ArrowLeft, Plus, Layers } from "./Icons.jsx";
import { isKebab, WORKLOADS, INGRESS_OPTIONS, DEFAULT_RULES } from "./spec.js";
import { RULE_GROUPS } from "./spec.js";

export function RichFormScreen({ initialSpec, onBack, onSubmit }) {
  const seed = initialSpec || { umbrellaChartName: "", description: "", subcharts: [{ name: "", workload: "deployment", description: "" }], rules: DEFAULT_RULES };
  const [name, setName] = React.useState(seed.umbrellaChartName || "");
  const [desc, setDesc] = React.useState(seed.description || "");
  const [subcharts, setSubcharts] = React.useState(seed.subcharts && seed.subcharts.length ? seed.subcharts : [{ name: "", workload: "deployment", description: "" }]);
  const [rules, setRules] = React.useState({ ...DEFAULT_RULES, ...(seed.rules || {}) });
  const [submitting, setSubmitting] = React.useState(false);
  const [submitError, setSubmitError] = React.useState("");
  const [touched, setTouched] = React.useState(false);

  const nameError = touched && !name.trim() ? "Required." : touched && !isKebab(name) ? "kebab-case only." : "";
  const subError = (s) => (touched && s.name && !isKebab(s.name) ? "kebab-case" : "");
  const canSubmit = name.trim() && isKebab(name) && subcharts.some((s) => s.name.trim() && isKebab(s.name));

  const setRule = (k, v) => setRules((r) => ({ ...r, [k]: v }));
  const addRow = () => setSubcharts((p) => [...p, { name: "", workload: "deployment", description: "" }]);
  const removeRow = (i) => setSubcharts((p) => (p.length === 1 ? p : p.filter((_, x) => x !== i)));
  const updRow = (i, f, v) => setSubcharts((p) => p.map((s, x) => (x === i ? { ...s, [f]: v } : s)));

  const submit = async () => {
    setTouched(true);
    setSubmitError("");
    if (!canSubmit) return;
    setSubmitting(true);
    const spec = {
      umbrellaChartName: name.trim(),
      description: desc.trim(),
      subcharts: subcharts.filter((s) => s.name.trim()).map((s) => ({ name: s.name.trim(), workload: s.workload, description: s.description || "" })),
      rules,
    };
    try {
      await onSubmit(spec); // parent applies /generate and navigates to Charts on success
    } catch (err) {
      setSubmitError((err && err.message) || String(err));
      setSubmitting(false);
    }
  };

  return (
    <div style={{ maxWidth: 1040, margin: "0 auto" }}>
      <button onClick={onBack} disabled={submitting} style={formBackBtn}><ArrowLeft size={15} /> Back</button>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 340px", gap: 24, alignItems: "start" }}>
        {/* left: the editor */}
        <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          <Card padding={28}>
            <h1 style={{ margin: "0 0 4px", fontSize: 24, fontWeight: 700, color: "var(--text-1)", letterSpacing: "-0.01em" }}>Chart specification</h1>
            <p style={{ margin: "0 0 22px", fontSize: 14, color: "var(--text-2)" }}>
              {initialSpec ? "Drafted from your prompt — review and edit before generating." : "Define your umbrella chart and its subcharts."}
            </p>
            <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
              <Field label="Umbrella chart name" required error={nameError} hint="Lowercase, kebab-case.">
                <Input mono placeholder="my-umbrella" value={name} invalid={!!nameError}
                       onBlur={() => setTouched(true)} onChange={(e) => setName(e.target.value)} />
              </Field>
              <Field label="Description">
                <Input placeholder="Example platform chart" value={desc} onChange={(e) => setDesc(e.target.value)} />
              </Field>
            </div>
          </Card>

          {/* subcharts */}
          <Card padding={28}>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 14 }}>
              <span style={sectionLabel}>Subcharts</span>
              <Button variant="outline" size="1" leadingIcon={<Plus size={14} />} onClick={addRow}>Add subchart</Button>
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
              <div style={{ display: "grid", gridTemplateColumns: "1fr 150px 1fr 36px", gap: 8, fontSize: 11, fontWeight: 600, color: "var(--text-muted)", textTransform: "uppercase", letterSpacing: "0.04em" }}>
                <span>Name</span><span>Workload</span><span>Description</span><span></span>
              </div>
              {subcharts.map((s, i) => (
                <div key={i} style={{ display: "grid", gridTemplateColumns: "1fr 150px 1fr 36px", gap: 8, alignItems: "start" }}>
                  <Input mono placeholder="api" value={s.name} invalid={!!subError(s)} onChange={(e) => updRow(i, "name", e.target.value)} />
                  <Select options={WORKLOADS} value={s.workload} onChange={(e) => updRow(i, "workload", e.target.value)} />
                  <Input placeholder="optional" value={s.description} onChange={(e) => updRow(i, "description", e.target.value)} />
                  <IconButton label="Remove subchart" color="danger" disabled={subcharts.length === 1} onClick={() => removeRow(i)}>×</IconButton>
                </div>
              ))}
            </div>
          </Card>

          {/* rules */}
          <Card padding={28}>
            <span style={sectionLabel}>Rules</span>
            <div style={{ marginTop: 14, marginBottom: 22 }}>
              <Field label="Ingress" hint="Which ingress controller the whole platform uses. istio produces a Gateway/VirtualService; none disables ingress.">
                <div style={{ maxWidth: 280 }}>
                  <Select options={INGRESS_OPTIONS} value={rules.ingress} onChange={(e) => setRule("ingress", e.target.value)} />
                </div>
              </Field>
            </div>
            {RULE_GROUPS.map((g) => (
              <div key={g.title} style={{ marginBottom: 18 }}>
                <div style={{ fontSize: 12, fontWeight: 600, color: "var(--text-muted)", textTransform: "uppercase", letterSpacing: "0.04em", marginBottom: 12 }}>{g.title}</div>
                <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
                  {g.rules.map((r) => (
                    <Checkbox key={r.key} checked={!!rules[r.key]} onChange={(v) => setRule(r.key, v)} label={r.label} description={r.desc} />
                  ))}
                </div>
              </div>
            ))}
          </Card>

          <Button size="3" fullWidth onClick={submit} loading={submitting} leadingIcon={!submitting ? <Layers size={16} /> : null}>
            {submitting ? "Submitting…" : "Submit & generate"}
          </Button>
          {touched && !canSubmit && (
            <p style={{ margin: 0, fontSize: 12, color: "var(--red-11)", textAlign: "center" }}>
              Add a kebab-case umbrella name and at least one valid subchart.
            </p>
          )}
          {submitError && <InlineError>{submitError}</InlineError>}
        </div>

        {/* right: sticky structure preview */}
        <div style={{ position: "sticky", top: 24 }}>
          <StructurePreview umbrellaName={name || "umbrella-chart"} subcharts={subcharts} rules={rules} />
          <p style={{ margin: "10px 2px 0", fontSize: 11, color: "var(--text-muted)", lineHeight: 1.5 }}>
            Live preview of the chart structure that would be generated. Updates as you edit.
          </p>
        </div>
      </div>
    </div>
  );
}

const formBackBtn = {
  display: "inline-flex", alignItems: "center", gap: 6, marginBottom: 14, padding: "6px 10px 6px 6px",
  background: "transparent", border: "none", color: "var(--text-2)", fontFamily: "var(--font-sans)",
  fontSize: 13, fontWeight: 500, cursor: "pointer", borderRadius: "var(--radius-3)",
};
const sectionLabel = { fontSize: 13, fontWeight: 600, color: "var(--text-1)" };
