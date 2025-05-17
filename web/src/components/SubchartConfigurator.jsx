import React, { useState } from "react";
import './SubchartConfigurator.css';

const initialSubchart = {
  subchart: "",
  type: "",
  image: "",
  tag: "",
  description: ""
};

export default function SubchartConfigurator() {
  const [subcharts, setSubcharts] = useState([]);
  const [form, setForm] = useState(initialSubchart);

  // Handle input change for the single-line form
  function handleChange(e) {
    const { name, value } = e.target;
    setForm((prev) => ({ ...prev, [name]: value }));
  }

  // Add subchart from form to the list
  function handleAdd() {
    if (!form.subchart || !form.type || !form.image) return; // Simple validation
    setSubcharts((prev) => [...prev, form]);
    setForm(initialSubchart);
  }

  return (
    <div style={{ maxWidth: 900, margin: "40px auto" }}>
      <h2>Configure Subcharts</h2>
      <table style={{ width: "100%", borderCollapse: "collapse" }}>
        <thead>
          <tr>
            <th>Subchart</th>
            <th>Type</th>
            <th>Image</th>
            <th>Tag</th>
            <th>Description</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {/* Input row for the new subchart */}
          <tr>
            <td>
              <input
                name="subchart"
                value={form.subchart}
                onChange={handleChange}
                placeholder="e.g. API"
              />
            </td>
            <td>
              <input
                name="type"
                value={form.type}
                onChange={handleChange}
                placeholder="e.g. Deployment"
              />
            </td>
            <td>
              <input
                name="image"
                value={form.image}
                onChange={handleChange}
                placeholder="e.g. oci://repo/api"
              />
            </td>
            <td>
              <input
                name="tag"
                value={form.tag}
                onChange={handleChange}
                placeholder="e.g. latest"
              />
            </td>
            <td>
              <input
                name="description"
                value={form.description}
                onChange={handleChange}
                placeholder="Short description"
              />
            </td>
            <td>
              <button onClick={handleAdd}>Add Subchart</button>
            </td>
          </tr>
        </tbody>
      </table>
      {/* Show saved subcharts as cards */}
      <div style={{ display: "flex", gap: 16, marginTop: 32, flexWrap: "wrap" }}>
        {subcharts.map((s, i) => (
          <div
            key={i}
            style={{
              border: "1px solid #eee",
              borderRadius: 8,
              boxShadow: "0 2px 8px #eee",
              padding: 16,
              minWidth: 220
            }}
          >
            <strong>{s.subchart}</strong>
            <div style={{ fontSize: 12, color: "#666" }}>{s.type}</div>
            <div style={{ fontSize: 13, margin: "6px 0" }}>{s.image}</div>
            <div style={{ fontSize: 12, color: "#999" }}>{s.tag}</div>
            <div style={{ marginTop: 12, fontSize: 12 }}>{s.description}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
