// AppShell — top nav (Generate / Charts), the generate wizard (Choose → Prompt
// → Rich form), and the live charts lifecycle. Submitting applies the spec via
// /generate (phase Pending); the charts list then polls /charts every ~2.5s
// while anything is still Pending or Generating, advancing to Ready or Failed.
import React from "react";
import { ChooseScreen } from "./ChooseScreen.jsx";
import { PromptScreen } from "./PromptScreen.jsx";
import { RichFormScreen } from "./RichFormScreen.jsx";
import { ChartsScreen } from "./ChartsScreen.jsx";
import { Github } from "./Icons.jsx";
import { generateChart, listCharts } from "./api.js";

const POLL_MS = 2500;
const isActive = (c) => ["pending", "generating"].includes(String(c.phase).toLowerCase());

// Pending/Generating (no settle time) float to the top, then most-recent first.
function sortCharts(list) {
  const ts = (c) => (c.lastGenerated ? Date.parse(c.lastGenerated) || 0 : Infinity);
  return [...list].sort((a, b) => ts(b) - ts(a));
}

export function AppShell() {
  const [nav, setNav] = React.useState("generate"); // generate | charts
  const [step, setStep] = React.useState("choose"); // choose | prompt | form
  const [draftSpec, setDraftSpec] = React.useState(null);
  const [charts, setCharts] = React.useState([]);
  const [listError, setListError] = React.useState("");
  const [refreshing, setRefreshing] = React.useState(false);

  // Refresh from the server, preserving any just-submitted row not yet visible.
  const refreshCharts = React.useCallback(async () => {
    setRefreshing(true);
    try {
      const server = await listCharts();
      const list = Array.isArray(server) ? server : [];
      setListError("");
      setCharts((prev) => {
        const names = new Set(list.map((c) => c.name));
        const pendingLocal = prev.filter((c) => c._optimistic && !names.has(c.name));
        return sortCharts([...pendingLocal, ...list]);
      });
    } catch (err) {
      setListError((err && err.message) || String(err));
    } finally {
      setRefreshing(false);
    }
  }, []);

  // Keep the latest refreshCharts in a ref so the poll interval never goes stale.
  const refreshRef = React.useRef(refreshCharts);
  React.useEffect(() => { refreshRef.current = refreshCharts; }, [refreshCharts]);

  // Initial load.
  React.useEffect(() => { refreshCharts(); }, [refreshCharts]);

  // Poll only while something is still building.
  const anyActive = charts.some(isActive);
  React.useEffect(() => {
    if (!anyActive) return undefined;
    const id = setInterval(() => refreshRef.current(), POLL_MS);
    return () => clearInterval(id);
  }, [anyActive]);

  const handleSubmit = async (spec) => {
    // Throws on a server error; RichFormScreen catches and surfaces it inline.
    const res = await generateChart(spec);
    const row = {
      name: (res && res.name) || spec.umbrellaChartName,
      phase: (res && res.phase) || "Pending",
      subchartCount: spec.subcharts.length,
      lastGenerated: "",
      _optimistic: true,
      isNew: true,
    };
    setCharts((cs) => sortCharts([row, ...cs.filter((c) => c.name !== row.name)]));
    setNav("charts");
    setStep("choose");
    setDraftSpec(null);
    setTimeout(() => refreshRef.current(), 600); // begin reconciling with the operator
  };

  const onDownload = (c) => {
    if (c.downloadUrl && c.downloadUrl !== "#") {
      window.open(c.downloadUrl, "_blank", "noopener,noreferrer");
    }
  };

  const goGenerate = () => { setNav("generate"); setStep("choose"); setDraftSpec(null); };
  const goCharts = () => { setNav("charts"); refreshCharts(); };

  return (
    <div style={{ minHeight: "100vh", background: "var(--surface-page)" }}>
      <nav style={navBar}>
        <div style={{ display: "flex", alignItems: "center", gap: 28 }}>
          <span style={{ fontSize: 20, fontWeight: 700, letterSpacing: "-0.01em", color: "var(--slate-12)" }}>
            chart<span style={{ color: "var(--violet-9)" }}>press</span>
          </span>
          <div style={{ display: "flex", gap: 4 }}>
            <NavLink active={nav === "generate"} onClick={goGenerate}>Generate</NavLink>
            <NavLink active={nav === "charts"} onClick={goCharts}>Charts</NavLink>
          </div>
        </div>
        <a href="https://github.com/kriipke/chartpress" target="_blank" rel="noopener noreferrer" style={ghLink}>
          <Github size={16} /> GitHub
        </a>
      </nav>

      <main style={{ padding: "40px 28px 72px" }}>
        {nav === "charts" && (
          <ChartsScreen charts={charts} polling={refreshing} error={listError} onDownload={onDownload} onGenerate={goGenerate} />
        )}
        {nav === "generate" && step === "choose" && (
          <ChooseScreen onManual={() => { setDraftSpec(null); setStep("form"); }} onPrompt={() => setStep("prompt")} />
        )}
        {nav === "generate" && step === "prompt" && (
          <PromptScreen onBack={() => setStep("choose")} onDrafted={(spec) => { setDraftSpec(spec); setStep("form"); }} />
        )}
        {nav === "generate" && step === "form" && (
          <RichFormScreen initialSpec={draftSpec} onBack={() => setStep(draftSpec ? "prompt" : "choose")} onSubmit={handleSubmit} />
        )}
      </main>
    </div>
  );
}

function NavLink({ active, onClick, children }) {
  const [hover, setHover] = React.useState(false);
  return (
    <button onClick={onClick} onMouseEnter={() => setHover(true)} onMouseLeave={() => setHover(false)}
      style={{
        padding: "7px 12px", border: "none", borderRadius: "var(--radius-3)", cursor: "pointer",
        fontFamily: "var(--font-sans)", fontSize: 14, fontWeight: active ? 600 : 500,
        background: active ? "var(--violet-3)" : hover ? "var(--slate-3)" : "transparent",
        color: active ? "var(--violet-11)" : "var(--text-2)", transition: "background-color .15s, color .15s",
      }}>
      {children}
    </button>
  );
}

const navBar = {
  display: "flex", alignItems: "center", justifyContent: "space-between",
  padding: "14px 28px", background: "var(--surface-card)", borderBottom: "1px solid var(--border-default)",
  position: "sticky", top: 0, zIndex: 10,
};
const ghLink = {
  display: "inline-flex", alignItems: "center", gap: 6, color: "var(--text-2)",
  textDecoration: "none", fontSize: 14, fontWeight: 500, fontFamily: "var(--font-sans)",
};
