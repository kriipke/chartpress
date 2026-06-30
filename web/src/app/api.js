// chartpress API client — the real backend (internal/server).
//
//   POST /generate         {umbrellaChartName, description, subcharts, rules}
//                          → {name, namespace, phase, manifestYaml}   (async; phase=Pending)
//   GET  /charts           → [{name, phase, subchartCount, lastGenerated, message?, downloadUrl?}]
//   GET  /charts/{name}    → one chart summary (for polling a single row)
//   POST /text-to-config   {prompt} → a drafted, normalized spec to pre-fill the form
//
// Paths are same-origin relative (matching deployment, where an ingress routes
// /generate, /charts, /text-to-config to chartpress-server and / to the SPA).
// In dev, vite.config.js proxies them to the local server. Override the origin
// with VITE_API_BASE if the API lives elsewhere.

const BASE = import.meta.env?.VITE_API_BASE ?? "";

// The server writes errors as plain text via http.Error; surface them verbatim
// (the design's InlineError preserves whitespace) instead of paraphrasing.
async function request(path, options) {
  const res = await fetch(BASE + path, options);
  if (!res.ok) {
    const text = (await res.text().catch(() => "")).trim();
    throw new Error(text || `Request failed (${res.status})`);
  }
  const ct = res.headers.get("content-type") || "";
  return ct.includes("application/json") ? res.json() : res.text();
}

const jsonPost = (path, body) =>
  request(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

// Apply a spec. Returns immediately with phase "Pending"; the operator renders
// asynchronously and advances Generating → Ready (or Failed).
export function generateChart(spec) {
  return jsonPost("/generate", spec);
}

// List every chart in the namespace, newest activity first is not guaranteed by
// the server, so callers may sort. Each row carries its current phase and, when
// Ready, a freshly presigned downloadUrl.
export function listCharts() {
  return request("/charts", { method: "GET" });
}

// Fetch a single chart's current summary (used when polling one row).
export function getChart(name) {
  return request(`/charts/${encodeURIComponent(name)}`, { method: "GET" });
}

// Draft a spec from a natural-language prompt (server calls the LLM drafter).
// Returns a normalized engine.Spec for the form to pre-fill and the user to edit.
export function draftFromPrompt(prompt) {
  return jsonPost("/text-to-config", { prompt });
}
