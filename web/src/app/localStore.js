// localStore — browser-side persistence for anonymous users.
//
// Signed-in users get a server-persisted, owner-scoped library (GET /charts).
// Anonymous users have no server identity to key a durable list on, so their
// list-of-record lives here in localStorage: the chart name + the exact spec
// they generated, remembered across reloads. The backend still renders each
// chart (the operator needs a CR), but those anonymous CRs are reaped after a
// TTL — so this local record outlives the server artifact and lets the user
// re-generate an expired chart from the remembered spec.
//
// Every request also carries a stable, random client id (see clientId) in the
// X-Chartpress-Client header; the backend scopes anonymous charts to it so two
// browsers don't see or clobber each other's charts.

const CLIENT_ID_KEY = "cp.clientId.v1";
const CHARTS_KEY = "cp.anonCharts.v1";

// Guard every access: localStorage can throw (private mode, disabled storage,
// quota). A failure degrades to in-memory-only behavior rather than crashing.
function safeGet(key) {
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
}
function safeSet(key, value) {
  try {
    window.localStorage.setItem(key, value);
  } catch {
    /* ignore: storage unavailable */
  }
}

// A cheap, dependency-free random token for the anonymous client id.
function randomId() {
  try {
    const buf = new Uint8Array(16);
    (window.crypto || window.msCrypto).getRandomValues(buf);
    return Array.from(buf, (b) => b.toString(16).padStart(2, "0")).join("");
  } catch {
    return `${Date.now().toString(16)}${Math.random().toString(16).slice(2)}`;
  }
}

// clientId returns this browser's stable anonymous id, generating and persisting
// one on first use. Sent on every API request so anonymous charts are scoped.
export function clientId() {
  let id = safeGet(CLIENT_ID_KEY);
  if (!id) {
    id = randomId();
    safeSet(CLIENT_ID_KEY, id);
  }
  return id;
}

function readCharts() {
  const raw = safeGet(CHARTS_KEY);
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function writeCharts(list) {
  safeSet(CHARTS_KEY, JSON.stringify(list));
}

// loadLocalCharts returns the remembered anonymous charts, newest first.
export function loadLocalCharts() {
  return readCharts()
    .slice()
    .sort((a, b) => (b.createdAt || 0) - (a.createdAt || 0));
}

// saveLocalChart records (or updates) a generated chart by name, keeping the
// spec so it can be regenerated after the server-side TTL reaps it.
export function saveLocalChart({ name, spec, subchartCount }) {
  const list = readCharts().filter((c) => c.name !== name);
  list.push({
    name,
    spec,
    subchartCount:
      subchartCount != null
        ? subchartCount
        : (spec && Array.isArray(spec.subcharts) ? spec.subcharts.length : 0),
    createdAt: Date.now(),
  });
  writeCharts(list);
}

// removeLocalChart forgets a chart (e.g. the user dismisses an expired row).
export function removeLocalChart(name) {
  writeCharts(readCharts().filter((c) => c.name !== name));
}
