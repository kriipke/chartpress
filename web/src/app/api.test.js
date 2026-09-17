// SPDX-License-Identifier: Apache-2.0
import { beforeEach, describe, expect, it, vi } from "vitest";

describe("API client", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.resetModules();
    vi.stubGlobal("fetch", vi.fn());
  });

  it("posts JSON and sends a stable anonymous owner header", async () => {
    fetch.mockResolvedValue({ ok: true, headers: new Headers({ "content-type": "application/json" }), json: async () => ({ phase: "Pending" }) });
    const { generateChart, listCharts } = await import("./api.js");
    await generateChart({ umbrellaChartName: "demo" });
    await listCharts();

    const first = fetch.mock.calls[0][1];
    const second = fetch.mock.calls[1][1];
    expect(first.method).toBe("POST");
    expect(JSON.parse(first.body)).toEqual({ umbrellaChartName: "demo" });
    expect(first.headers["X-Chartpress-Client"]).toMatch(/^[a-f0-9]+$/);
    expect(second.headers["X-Chartpress-Client"]).toBe(first.headers["X-Chartpress-Client"]);
  });

  it("preserves server error text and status", async () => {
    fetch.mockResolvedValue({ ok: false, status: 422, text: async () => "invalid spec\n" });
    const { listCharts } = await import("./api.js");
    await expect(listCharts()).rejects.toMatchObject({ message: "invalid spec", status: 422 });
  });
});
