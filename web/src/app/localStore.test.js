// SPDX-License-Identifier: Apache-2.0
import { beforeEach, describe, expect, it, vi } from "vitest";

describe("anonymous chart persistence", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.resetModules();
    vi.restoreAllMocks();
  });

  it("upserts by name and returns newest first", async () => {
    const store = await import("./localStore.js");
    vi.spyOn(Date, "now").mockReturnValueOnce(10).mockReturnValueOnce(20).mockReturnValueOnce(30);
    store.saveLocalChart({ name: "old", spec: { subcharts: [] } });
    store.saveLocalChart({ name: "new", spec: { subcharts: [{ name: "api" }] } });
    store.saveLocalChart({ name: "old", spec: { subcharts: [{ name: "worker" }] } });
    expect(store.loadLocalCharts().map((c) => c.name)).toEqual(["old", "new"]);
    expect(store.loadLocalCharts()[0].subchartCount).toBe(1);
  });

  it("removes a remembered chart", async () => {
    const store = await import("./localStore.js");
    store.saveLocalChart({ name: "demo", spec: {} });
    store.removeLocalChart("demo");
    expect(store.loadLocalCharts()).toEqual([]);
  });
});
