// Rich form — the core spec editor, shared by the manual (empty) and prompt
// (pre-filled) paths. Umbrella · subcharts (pattern picker + trait overrides)
// · rules · a live structure preview. The pattern picker IS the configurator:
// adding a subchart presents the card grid; the trait questionnaire survives
// as each row's Advanced panel and the Custom card. Submitting calls the real
// /generate; a server error surfaces inline (verbatim) and keeps the user on
// the form.
import React from "react";
import { Card, Field, Input, Select, Button, IconButton, Checkbox, Badge, StructurePreview, InlineError, Dialog } from "../design/components";
import { ArrowLeft, Plus, Layers } from "./Icons.jsx";
import {
  isKebab, WORKLOADS, INGRESS_OPTIONS, EXPOSURE_OPTIONS, SCALING_OPTIONS,
  DEFAULT_RULES, RULE_GROUPS, PATTERNS, PATTERN_BY_ID, DEFAULT_PATTERN, TRAIT_KEYS,
  KNOWN_DEPENDENCIES, resolveTraits, traitOverrides, specWarnings, cleanSubchart, normalizeSpec,
} from "./spec.js";

export function RichFormScreen({ initialSpec, onBack, onSubmit }) {
  const seed = normalizeSpec(initialSpec);
  const [name, setName] = React.useState(seed.umbrellaChartName);
  const [desc, setDesc] = React.useState(seed.description);
  const [subcharts, setSubcharts] = React.useState(seed.subcharts);
  const [dependencies, setDependencies] = React.useState(seed.dependencies || []);
  const [depDraft, setDepDraft] = React.useState("");
  const [rules, setRules] = React.useState(seed.rules);
  const [submitting, setSubmitting] = React.useState(false);
  const [submitError, setSubmitError] = React.useState("");
  const [touched, setTouched] = React.useState(false);
  // Drafted/loaded rows arrive collapsed; rows added via the picker open
  // expanded (that's the configurator moment). null = picker closed;
  // "new" = picking for a new row; a number = re-picking for that row.
  const [expanded, setExpanded] = React.useState(() => new Set(initialSpec ? [] : [0]));
  const [pickerFor, setPickerFor] = React.useState(null);

  const nameError = touched && !name.trim() ? "Required." : touched && !isKebab(name) ? "kebab-case only." : "";
  const subError = (s) => (touched && s.name && !isKebab(s.name) ? "kebab-case" : "");
  const canSubmit = name.trim() && isKebab(name) && subcharts.some((s) => s.name.trim() && isKebab(s.name));

  const warnings = specWarnings({ umbrellaChartName: name, subcharts, rules });
  const resolvedAll = subcharts.map((s) => resolveTraits(s, rules));

  const setRule = (k, v) => setRules((r) => ({ ...r, [k]: v }));
  const toggleExpanded = (i) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      next.has(i) ? next.delete(i) : next.add(i);
      return next;
    });

  const pickPattern = (id) => {
    if (pickerFor === "new") {
      const idx = subcharts.length;
      setSubcharts((p) => [...p, { name: "", description: "", pattern: id }]);
      setExpanded((e) => new Set([...e, idx]));
    } else if (typeof pickerFor === "number") {
      // Re-picking resets the row's overrides: the new pattern's defaults win.
      setSubcharts((p) => p.map((s, x) => (x === pickerFor ? { name: s.name, description: s.description, pattern: id } : s)));
      setExpanded((e) => new Set([...e, pickerFor]));
    }
    setPickerFor(null);
  };

  const removeRow = (i) => {
    setSubcharts((p) => (p.length === 1 ? p : p.filter((_, x) => x !== i)));
    setExpanded((prev) => {
      const next = new Set();
      for (const x of prev) {
        if (x < i) next.add(x);
        else if (x > i) next.add(x - 1);
      }
      return next;
    });
  };

  const updRow = (i, f, v) => setSubcharts((p) => p.map((s, x) => (x === i ? { ...s, [f]: v } : s)));

  const addDependency = () => {
    const key = depDraft.trim().toLowerCase();
    setDepDraft("");
    if (key && !dependencies.includes(key)) setDependencies((d) => [...d, key]);
  };
  const removeDependency = (key) => setDependencies((d) => d.filter((k) => k !== key));

  // setTrait writes an explicit trait key — unless the chosen value is exactly
  // what the pattern would resolve anyway, in which case the key is removed
  // (the row stays clean intent, and the chip disappears).
  const setTrait = (i, key, value) =>
    setSubcharts((p) =>
      p.map((s, x) => {
        if (x !== i) return s;
        const next = { ...s };
        delete next[key];
        if (value != null && value !== "" && resolveTraits(next, rules)[key] !== value) next[key] = value;
        // Dependent cleanup: keys that no longer apply are dropped, never kept
        // hidden (an invisible override is how invalid specs happen).
        const rt = resolveTraits(next, rules);
        if (rt.exposure === "none" || rt.exposure === "tcp") delete next.ingress;
        if (rt.exposure === "none") delete next.port;
        if (rt.workload === "daemonset") delete next.scaling;
        return next;
      })
    );

  const resetTraits = (i) =>
    setSubcharts((p) =>
      p.map((s, x) => {
        if (x !== i) return s;
        const next = { ...s };
        for (const key of TRAIT_KEYS) delete next[key];
        return next;
      })
    );

  const submit = async () => {
    setTouched(true);
    setSubmitError("");
    if (!canSubmit) return;
    setSubmitting(true);
    const spec = {
      umbrellaChartName: name.trim(),
      description: desc.trim(),
      subcharts: subcharts.filter((s) => s.name.trim()).map(cleanSubchart),
      dependencies,
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
              {initialSpec ? "Drafted from your input — review and edit before generating." : "Define your umbrella chart and its subcharts."}
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
              <Button variant="outline" size="1" leadingIcon={<Plus size={14} />} onClick={() => setPickerFor("new")}>Add subchart</Button>
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
              {subcharts.map((s, i) => (
                <SubchartRow
                  key={i}
                  sub={s}
                  resolved={resolvedAll[i]}
                  overrides={traitOverrides(s, rules)}
                  rules={rules}
                  expanded={expanded.has(i)}
                  error={subError(s)}
                  removable={subcharts.length > 1}
                  onToggle={() => toggleExpanded(i)}
                  onName={(v) => updRow(i, "name", v)}
                  onDescription={(v) => updRow(i, "description", v)}
                  onTrait={(key, v) => setTrait(i, key, v)}
                  onReset={() => resetTraits(i)}
                  onRepick={() => setPickerFor(i)}
                  onRemove={() => removeRow(i)}
                />
              ))}
            </div>
          </Card>

          {/* infrastructure dependencies */}
          <Card padding={28}>
            <span style={sectionLabel}>Infrastructure dependencies</span>
            <p style={{ margin: "6px 0 14px", fontSize: 13, color: "var(--text-2)", lineHeight: 1.5 }}>
              Databases, caches, and brokers you don't write yourself. These become umbrella
              <code style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}> Chart.yaml</code> dependencies
              from mature upstream charts — never generated subcharts. Known keys are pinned; anything else lands as a TODO stub.
            </p>
            {dependencies.length > 0 && (
              <div style={{ display: "flex", gap: 6, flexWrap: "wrap", marginBottom: 12 }}>
                {dependencies.map((key) => (
                  <span key={key} style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                    <Badge color={KNOWN_DEPENDENCIES.includes(key) ? "accent" : "amber"} variant="soft">
                      {key}{KNOWN_DEPENDENCIES.includes(key) ? "" : " (TODO)"}
                    </Badge>
                    <button type="button" onClick={() => removeDependency(key)} aria-label={`Remove ${key}`} style={depRemoveBtn}>×</button>
                  </span>
                ))}
              </div>
            )}
            <div style={{ display: "flex", gap: 8, maxWidth: 360 }}>
              <Input mono placeholder="postgresql" value={depDraft}
                     list="chartpress-known-deps"
                     onChange={(e) => setDepDraft(e.target.value)}
                     onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); addDependency(); } }} />
              <datalist id="chartpress-known-deps">
                {KNOWN_DEPENDENCIES.map((k) => <option key={k} value={k} />)}
              </datalist>
              <Button variant="outline" size="1" leadingIcon={<Plus size={14} />} onClick={addDependency}>Add</Button>
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

          {warnings.length > 0 && (
            <div style={warnBox}>
              {warnings.map((w, i) => (
                <div key={i} style={{ display: "flex", gap: 8, alignItems: "flex-start" }}>
                  <span aria-hidden="true">⚠</span>
                  <span>{w}</span>
                </div>
              ))}
            </div>
          )}

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
          <StructurePreview umbrellaName={name || "umbrella-chart"} subcharts={subcharts} rules={rules} traits={resolvedAll} dependencies={dependencies} />
          <p style={{ margin: "10px 2px 0", fontSize: 11, color: "var(--text-muted)", lineHeight: 1.5 }}>
            Live preview of the chart structure that would be generated. Updates as you edit.
          </p>
        </div>
      </div>

      <PatternPicker open={pickerFor !== null} onClose={() => setPickerFor(null)} onPick={pickPattern} />
    </div>
  );
}

/* ---------- subchart row: chip strip collapsed, trait panel expanded ---------- */

function SubchartRow({ sub, resolved, overrides, rules, expanded, error, removable, onToggle, onName, onDescription, onTrait, onReset, onRepick, onRemove }) {
  const pattern = PATTERN_BY_ID[sub.pattern || DEFAULT_PATTERN];
  const sharedRulesOn = rules.shared_secrets_config || rules.shared_newrelic_config;
  const ingressable = (resolved.exposure === "http" || resolved.exposure === "grpc") && rules.ingress !== "none";

  return (
    <div style={{ border: "1px solid var(--border-default)", borderRadius: "var(--radius-4)", background: "var(--surface-card)" }}>
      {/* header */}
      <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "10px 12px" }}>
        <button type="button" onClick={onToggle} aria-label={expanded ? "Collapse" : "Configure"} style={chevronBtn}>
          {expanded ? "▾" : "▸"}
        </button>
        <div style={{ width: 180 }}>
          <Input mono placeholder="api" value={sub.name} invalid={!!error} onChange={(e) => onName(e.target.value)} />
        </div>
        <button type="button" onClick={onRepick} title="Change pattern" style={chipBtn}>
          <Badge color="accent">{pattern.label}</Badge>
        </button>
        <div style={{ display: "flex", gap: 6, flexWrap: "wrap", flex: 1, minWidth: 0 }}>
          {overrides.map((key) => (
            <Badge key={key} variant="outline" size="1">
              {key === "shared_env" ? (resolved.shared_env ? "+shared env" : "no shared env")
                : key === "ingress" ? (resolved.ingress ? "+ingress" : "no ingress")
                : key === "port" ? `:${resolved.port}`
                : resolved[key]}
            </Badge>
          ))}
          {!expanded && (
            <span style={{ fontSize: 12, color: "var(--text-muted)", alignSelf: "center", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
              {summaryLine(resolved)}
            </span>
          )}
        </div>
        <IconButton label="Remove subchart" color="danger" disabled={!removable} onClick={onRemove}>×</IconButton>
      </div>

      {/* expanded: the trait questionnaire, pre-answered by the pattern */}
      {expanded && (
        <div style={{ borderTop: "1px solid var(--border-subtle)", padding: "14px 12px 16px 40px", display: "flex", flexDirection: "column", gap: 14 }}>
          <Field label="Description">
            <Input placeholder="optional" value={sub.description || ""} onChange={(e) => onDescription(e.target.value)} />
          </Field>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
            <Field label="Workload">
              <Select options={WORKLOADS} value={resolved.workload} onChange={(e) => onTrait("workload", e.target.value)} />
            </Field>
            <Field label="How is it contacted?">
              <Select options={EXPOSURE_OPTIONS} value={resolved.exposure} onChange={(e) => onTrait("exposure", e.target.value)} />
            </Field>
            {resolved.exposure !== "none" && (
              <Field label="Port">
                <Input mono type="number" min={1} max={65535} value={resolved.port}
                       onChange={(e) => onTrait("port", Number(e.target.value) || 0)} />
              </Field>
            )}
            {resolved.workload !== "daemonset" && (
              <Field label="Can more than one run?">
                <Select options={SCALING_OPTIONS} value={resolved.scaling} onChange={(e) => onTrait("scaling", e.target.value)} />
              </Field>
            )}
          </div>
          {ingressable && (
            <Checkbox checked={resolved.ingress} onChange={(v) => onTrait("ingress", v)}
                      label="Reachable from outside the cluster"
                      description="Route external traffic to it through the platform ingress." />
          )}
          {sharedRulesOn && (
            <Checkbox checked={resolved.shared_env} onChange={(v) => onTrait("shared_env", v)}
                      label="Receive the platform's shared config/secrets"
                      description="Mount the umbrella's shared ConfigMap/Secrets as environment variables." />
          )}
          {overrides.length > 0 && (
            <div>
              <Button variant="outline" size="1" onClick={onReset}>Reset to pattern</Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function summaryLine(rt) {
  const parts = [];
  parts.push(rt.exposure === "none" ? "no service" : `${rt.exposure} :${rt.port}`);
  if (rt.ingress) parts.push("ingress");
  parts.push(rt.scaling === "auto" ? "autoscale" : rt.scaling);
  if (!rt.shared_env) parts.push("no shared env");
  return parts.join(" · ");
}

/* ---------- pattern picker dialog ---------- */

function PatternPicker({ open, onClose, onPick }) {
  return (
    <Dialog open={open} onClose={onClose} size="3" title="What kind of workload is it?"
            description="Pick the closest match — every choice is adjustable afterwards. Custom asks the questions one by one.">
      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
        {PATTERNS.map((p) => (
          <PatternCard key={p.id} pattern={p} onClick={() => onPick(p.id)} />
        ))}
      </div>
    </Dialog>
  );
}

function PatternCard({ pattern, onClick }) {
  const [hover, setHover] = React.useState(false);
  return (
    <button
      type="button"
      onClick={onClick}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: "flex", flexDirection: "column", alignItems: "flex-start", gap: 6,
        textAlign: "left", padding: 14, width: "100%", boxSizing: "border-box",
        background: "var(--surface-card)",
        border: `1px solid ${hover ? "var(--accent-solid)" : "var(--border-default)"}`,
        borderRadius: "var(--radius-4)",
        boxShadow: hover ? "var(--shadow-accent)" : "none",
        cursor: "pointer",
        transition: "border-color .18s var(--ease-standard), box-shadow .2s var(--ease-standard)",
      }}
    >
      <span style={{ fontFamily: "var(--font-sans)", fontSize: 14, fontWeight: 700, color: "var(--text-1)" }}>{pattern.label}</span>
      <span style={{ fontFamily: "var(--font-sans)", fontSize: 12, color: "var(--text-2)", lineHeight: 1.45 }}>like {pattern.example}</span>
      <span style={{ display: "flex", gap: 5, flexWrap: "wrap" }}>
        {pattern.badges.map((b) => (
          <Badge key={b} size="1" variant="outline">{b}</Badge>
        ))}
      </span>
    </button>
  );
}

const formBackBtn = {
  display: "inline-flex", alignItems: "center", gap: 6, marginBottom: 14, padding: "6px 10px 6px 6px",
  background: "transparent", border: "none", color: "var(--text-2)", fontFamily: "var(--font-sans)",
  fontSize: 13, fontWeight: 500, cursor: "pointer", borderRadius: "var(--radius-3)",
};
const sectionLabel = { fontSize: 13, fontWeight: 600, color: "var(--text-1)" };
const chevronBtn = {
  border: "none", background: "transparent", cursor: "pointer", color: "var(--text-muted)",
  fontSize: 13, width: 22, height: 22, display: "flex", alignItems: "center", justifyContent: "center",
  borderRadius: "var(--radius-2)", flexShrink: 0,
};
const chipBtn = { border: "none", background: "transparent", padding: 0, cursor: "pointer", flexShrink: 0 };
const depRemoveBtn = {
  border: "none", background: "transparent", cursor: "pointer", color: "var(--text-muted)",
  fontSize: 16, lineHeight: 1, padding: "0 2px",
};
const warnBox = {
  display: "flex", flexDirection: "column", gap: 8,
  padding: "12px 14px", borderRadius: "var(--radius-4)",
  background: "var(--amber-3)", border: "1px solid var(--amber-6)",
  color: "var(--amber-11)", fontFamily: "var(--font-sans)", fontSize: 13, lineHeight: 1.5,
};
