// Charts browser — the permanent destination. Empty state · steady list · live
// polling (a freshly submitted chart transitions Pending → Generating → Ready).
import React from "react";
import { Card, ChartsTable, EmptyState, Button, Badge } from "../design/components";
import { Package, Plus } from "./Icons.jsx";

export function ChartsScreen({ charts, polling, error, onDownload, onGenerate }) {
  const active = charts.filter((c) => ["pending", "generating"].includes(String(c.phase).toLowerCase())).length;

  if (charts.length === 0) {
    return (
      <div style={{ maxWidth: 900, margin: "0 auto" }}>
        <ChartsHeader active={0} onGenerate={onGenerate} />
        {error && <ListError error={error} />}
        <Card padding={0}>
          <EmptyState
            icon={<Package size={24} />}
            title="No charts yet"
            description="Generate your first umbrella chart and it'll appear here while it builds."
            action={<Button leadingIcon={<Plus size={16} />} onClick={onGenerate}>Generate a chart</Button>}
          />
        </Card>
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 900, margin: "0 auto" }}>
      <ChartsHeader active={active} onGenerate={onGenerate} />
      {error && <ListError error={error} />}
      <Card padding={0} style={{ overflow: "hidden" }}>
        <ChartsTable charts={charts} onDownload={onDownload} />
      </Card>
      <p style={{ margin: "12px 2px 0", fontSize: 12, color: "var(--text-muted)" }}>
        {active > 0
          ? `Polling every ~2.5s · ${active} chart${active > 1 ? "s" : ""} still building`
          : polling
          ? "Refreshing…"
          : "All charts settled · polling paused"}
      </p>
    </div>
  );
}

function ChartsHeader({ active, onGenerate }) {
  return (
    <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 18 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <h1 style={{ margin: 0, fontSize: 24, fontWeight: 700, color: "var(--text-1)", letterSpacing: "-0.01em" }}>Charts</h1>
        {active > 0 && <Badge color="accent" variant="soft" leadingDot>{active} building</Badge>}
      </div>
      <Button leadingIcon={<Plus size={16} />} onClick={onGenerate}>Generate a chart</Button>
    </div>
  );
}

function ListError({ error }) {
  return (
    <p style={{ margin: "0 2px 14px", fontSize: 12, color: "var(--red-11)" }}>
      Couldn't refresh the charts list: {error}
    </p>
  );
}
