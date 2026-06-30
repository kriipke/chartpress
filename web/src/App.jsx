import { useState } from "react";
import "./App.css";

const WORKLOADS = ["deployment", "statefulset", "daemonset"];

export default function App() {
  const [umbrellaChartName, setUmbrellaChartName] = useState("");
  const [subcharts, setSubcharts] = useState([{ name: "", workload: "deployment" }]);
  const [loading, setLoading] = useState(false);
  const [downloadUrl, setDownloadUrl] = useState("");
  const [error, setError] = useState("");

  const addSubchart = () =>
    setSubcharts((prev) => [...prev, { name: "", workload: "deployment" }]);

  const removeSubchart = (index) =>
    setSubcharts((prev) => prev.filter((_, i) => i !== index));

  const updateSubchart = (index, field, value) =>
    setSubcharts((prev) =>
      prev.map((sc, i) => (i === index ? { ...sc, [field]: value } : sc))
    );

  const handleGenerate = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    setDownloadUrl("");
    try {
      const payload = {
        umbrellaChartName: umbrellaChartName.trim() || "umbrella-chart",
        subcharts: subcharts
          .filter((s) => s.name.trim())
          .map((s) => ({ name: s.name.trim(), workload: s.workload })),
      };

      // The server returns JSON ({ downloadUrl }) to browser clients; only
      // clients sending `Accept: application/zip` (the CLI) get the raw zip.
      const response = await fetch("/chartpress/generate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        throw new Error((await response.text()) || `Request failed (${response.status})`);
      }

      const data = await response.json();
      if (!data.downloadUrl) {
        throw new Error("Server did not return a download URL");
      }
      setDownloadUrl(data.downloadUrl);
    } catch (err) {
      setError(err.message || String(err));
    } finally {
      setLoading(false);
    }
  };

  const treeStructure = () => {
    const name = umbrellaChartName.trim() || "umbrella-chart";
    const named = subcharts.filter((s) => s.name.trim());
    let tree = `${name}/\n`;
    tree += "├── Chart.yaml\n";
    tree += "├── values.yaml\n";
    tree += "└── charts/\n";
    if (named.length === 0) {
      tree += "        (add a subchart to populate)\n";
      return tree;
    }
    named.forEach((sc, i) => {
      const last = i === named.length - 1;
      const branch = last ? "    └──" : "    ├──";
      const pipe = last ? "        " : "    │   ";
      tree += `${branch} ${sc.name}/   (${sc.workload})\n`;
      tree += `${pipe}├── Chart.yaml\n`;
      tree += `${pipe}├── values.yaml\n`;
      tree += `${pipe}└── templates/\n`;
    });
    return tree;
  };

  return (
    <div className="cp">
      <nav className="cp-nav">
        <span className="cp-logo">ChartPress</span>
        <div className="cp-nav-links">
          <a href="/chartpress/">Home</a>
          <a
            href="https://github.com/kriipke/chartpress"
            target="_blank"
            rel="noopener noreferrer"
          >
            GitHub
          </a>
        </div>
      </nav>

      <main className="cp-main">
        <form className="cp-panel" onSubmit={handleGenerate}>
          <h1>Generate a Helm chart</h1>
          <p className="cp-sub">
            Define an umbrella chart and its subcharts, then generate.
          </p>

          <label className="cp-field">
            <span>Umbrella chart name</span>
            <input
              type="text"
              placeholder="my-umbrella"
              value={umbrellaChartName}
              onChange={(e) => setUmbrellaChartName(e.target.value)}
            />
          </label>

          <div className="cp-subcharts">
            <div className="cp-subcharts-head">
              <span>Subcharts</span>
              <button type="button" className="cp-add" onClick={addSubchart}>
                + Add subchart
              </button>
            </div>

            {subcharts.map((sc, i) => (
              <div className="cp-subchart-row" key={i}>
                <input
                  type="text"
                  placeholder="subchart name"
                  value={sc.name}
                  onChange={(e) => updateSubchart(i, "name", e.target.value)}
                />
                <select
                  value={sc.workload}
                  onChange={(e) => updateSubchart(i, "workload", e.target.value)}
                >
                  {WORKLOADS.map((w) => (
                    <option key={w} value={w}>
                      {w}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  className="cp-remove"
                  onClick={() => removeSubchart(i)}
                  aria-label="Remove subchart"
                  disabled={subcharts.length === 1}
                >
                  ×
                </button>
              </div>
            ))}
          </div>

          <button type="submit" className="cp-generate" disabled={loading}>
            {loading ? "Generating…" : "Generate chart"}
          </button>

          {error ? <p className="cp-error">{error}</p> : null}
          {downloadUrl ? (
            <a className="cp-download" href={downloadUrl}>
              ↓ Download generated chart
            </a>
          ) : null}
        </form>

        <aside className="cp-preview">
          <span className="cp-preview-label">Structure preview</span>
          <pre className="cp-tree">{treeStructure()}</pre>
        </aside>
      </main>
    </div>
  );
}
