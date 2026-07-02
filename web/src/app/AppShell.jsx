// AppShell — top nav (Generate / Charts + GitHub sign-in), the generate wizard
// (Choose → Prompt → Rich form), the live charts lifecycle, the per-chart file
// explorer, and the profile screen.
//
// Generation is applied via /generate (phase Pending); the charts list then polls
// /charts every ~2.5s while anything is Pending/Generating, advancing to Ready or
// Failed. Opening a Ready chart fetches its rendered files (read-only) from
// /charts/{name}/files. Sign-in is optional and non-gating: /auth/me tells us
// whether it's configured and who (if anyone) is signed in — the app is fully
// usable signed-out.
import React from "react";
import { ChooseScreen } from "./ChooseScreen.jsx";
import { PromptScreen } from "./PromptScreen.jsx";
import { ComposeScreen } from "./ComposeScreen.jsx";
import { RichFormScreen } from "./RichFormScreen.jsx";
import { ChartsScreen } from "./ChartsScreen.jsx";
import { ProfileScreen } from "./ProfileScreen.jsx";
import { ChartExplorer, UserMenu, SignInButton } from "../design/components";
import { Github } from "./Icons.jsx";
import {
  generateChart, listCharts, getMe, logout, getChartFiles, githubLoginUrl,
} from "./api.js";
import logoUrl from "../assets/logo-blue.png";

const POLL_MS = 2500;
const isActive = (c) => ["pending", "generating"].includes(String(c.phase).toLowerCase());

// Pending/Generating (no settle time) float to the top, then most-recent first.
function sortCharts(list) {
  const ts = (c) => (c.lastGenerated ? Date.parse(c.lastGenerated) || 0 : Infinity);
  return [...list].sort((a, b) => ts(b) - ts(a));
}

export function AppShell() {
  const [nav, setNav] = React.useState("charts"); // generate | charts | profile | explorer
  const [step, setStep] = React.useState("choose"); // choose | prompt | compose | form
  const [draftSpec, setDraftSpec] = React.useState(null);
  const [draftFrom, setDraftFrom] = React.useState("choose"); // which step the form's Back returns to
  const [charts, setCharts] = React.useState([]);
  const [listError, setListError] = React.useState("");
  const [refreshing, setRefreshing] = React.useState(false);
  const [openChart, setOpenChart] = React.useState(null); // chart being explored
  const [auth, setAuth] = React.useState({ configured: false, user: null });

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

  // Initial load: charts + auth status.
  React.useEffect(() => { refreshCharts(); }, [refreshCharts]);
  React.useEffect(() => {
    let live = true;
    getMe()
      .then((m) => { if (live) setAuth({ configured: !!(m && m.configured), user: m && m.authenticated ? m.user : null }); })
      .catch(() => { if (live) setAuth({ configured: false, user: null }); });
    return () => { live = false; };
  }, []);

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

  const goGenerate = () => { setNav("generate"); setStep("choose"); setDraftSpec(null); setDraftFrom("choose"); setOpenChart(null); };
  const goCharts = () => { setNav("charts"); setOpenChart(null); refreshCharts(); };
  const openChartView = (chart) => { setOpenChart(chart); setNav("explorer"); };

  const signIn = () => { window.location.href = githubLoginUrl; };
  const signOut = () => {
    logout().catch(() => {}).finally(() => {
      setAuth((a) => ({ ...a, user: null }));
      setNav("charts");
      setOpenChart(null);
    });
  };

  // Map the /auth/me user (login-based) to the display shape UserMenu/Profile want.
  const displayUser = auth.user ? {
    name: auth.user.name || auth.user.login,
    handle: auth.user.login ? "@" + auth.user.login : "",
    email: auth.user.email,
    registry: auth.user.registry,
    avatarUrl: auth.user.avatarUrl,
  } : null;

  return (
    <div style={{ minHeight: "100vh", background: "var(--surface-page)" }}>
      <nav style={navBar}>
        <div style={{ display: "flex", alignItems: "center", gap: 28 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 9 }}>
            <img src={logoUrl} alt="chartpress logo" width="38" height="38" style={{ display: "block", objectFit: "contain" }} />
            <span style={{ fontSize: 20, fontWeight: 700, letterSpacing: "-0.01em", color: "var(--slate-12)" }}>
              chart<span style={{ color: "var(--accent-9)" }}>press</span>
            </span>
          </div>
          <div style={{ display: "flex", gap: 4 }}>
            <NavLink active={nav === "generate"} onClick={goGenerate}>Generate</NavLink>
            <NavLink active={nav === "charts" || nav === "explorer"} onClick={goCharts}>Charts</NavLink>
          </div>
        </div>
        <AuthControl
          configured={auth.configured}
          user={displayUser}
          onSignIn={signIn}
          onProfile={() => { setNav("profile"); setOpenChart(null); }}
          onLogout={signOut}
        />
      </nav>

      {nav === "explorer" && openChart ? (
        <main style={{ padding: "24px 28px 40px", height: "calc(100vh - 65px)", boxSizing: "border-box" }}>
          <ChartViewContainer chart={openChart} onBack={goCharts} onDownload={onDownload} onRegenerate={goGenerate} />
        </main>
      ) : (
        <main style={{ padding: "40px 28px 72px" }}>
          {nav === "profile" && displayUser && (
            <ProfileScreen user={displayUser} charts={charts} onBack={() => setNav("charts")} />
          )}
          {nav === "charts" && (
            <ChartsScreen charts={charts} polling={refreshing} error={listError} onDownload={onDownload} onGenerate={goGenerate} onOpen={openChartView} />
          )}
          {nav === "generate" && step === "choose" && (
            <ChooseScreen
              onManual={() => { setDraftSpec(null); setDraftFrom("choose"); setStep("form"); }}
              onPrompt={() => setStep("prompt")}
              onCompose={() => setStep("compose")}
            />
          )}
          {nav === "generate" && step === "prompt" && (
            <PromptScreen onBack={() => setStep("choose")} onDrafted={(spec) => { setDraftSpec(spec); setDraftFrom("prompt"); setStep("form"); }} />
          )}
          {nav === "generate" && step === "compose" && (
            <ComposeScreen onBack={() => setStep("choose")} onDrafted={(spec) => { setDraftSpec(spec); setDraftFrom("compose"); setStep("form"); }} />
          )}
          {nav === "generate" && step === "form" && (
            <RichFormScreen initialSpec={draftSpec} onBack={() => setStep(draftFrom)} onSubmit={handleSubmit} />
          )}
        </main>
      )}
    </div>
  );
}

// ChartViewContainer fetches a Ready chart's rendered files (read-only) and hands
// them to the ChartExplorer. Failed charts skip the fetch — the explorer shows the
// error panel from the chart's message.
function ChartViewContainer({ chart, onBack, onDownload, onRegenerate }) {
  const failed = String(chart.phase).toLowerCase() === "failed";
  const [data, setData] = React.useState(null);
  const [loading, setLoading] = React.useState(!failed);
  const [error, setError] = React.useState("");

  React.useEffect(() => {
    if (failed) { setData(null); setLoading(false); setError(""); return undefined; }
    let live = true;
    setLoading(true); setError(""); setData(null);
    getChartFiles(chart.name)
      .then((d) => { if (live) setData(d); })
      .catch((e) => { if (live) setError((e && e.message) || String(e)); })
      .finally(() => { if (live) setLoading(false); });
    return () => { live = false; };
  }, [chart.name, failed]);

  return (
    <ChartExplorer
      name={chart.name}
      phase={chart.phase}
      nodes={data ? data.nodes : []}
      files={data ? data.files : {}}
      message={chart.message}
      onBack={onBack}
      onDownload={() => onDownload(chart)}
      onRegenerate={onRegenerate}
      readOnly
      loading={loading}
      loadError={error}
    />
  );
}

// AuthControl fills the nav's right slot: the UserMenu when signed in, a
// SignInButton when sign-in is configured but signed out, and the GitHub repo
// link otherwise (sign-in disabled on this server).
function AuthControl({ configured, user, onSignIn, onProfile, onLogout }) {
  if (user) {
    return (
      <UserMenu
        name={user.name}
        handle={user.handle}
        avatarUrl={user.avatarUrl}
        onSelect={(action) => { if (action === "profile") onProfile(); if (action === "logout") onLogout(); }}
      />
    );
  }
  if (configured) {
    return <SignInButton onClick={onSignIn} />;
  }
  return (
    <a href="https://github.com/kriipke/chartpress" target="_blank" rel="noopener noreferrer" style={ghLink}>
      <Github size={16} /> GitHub
    </a>
  );
}

function NavLink({ active, onClick, children }) {
  const [hover, setHover] = React.useState(false);
  return (
    <button onClick={onClick} onMouseEnter={() => setHover(true)} onMouseLeave={() => setHover(false)}
      style={{
        padding: "7px 12px", border: "none", borderRadius: "var(--radius-3)", cursor: "pointer",
        fontFamily: "var(--font-sans)", fontSize: 14, fontWeight: active ? 600 : 500,
        background: active ? "var(--accent-3)" : hover ? "var(--slate-3)" : "transparent",
        color: active ? "var(--accent-11)" : "var(--text-2)", transition: "background-color .15s, color .15s",
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
