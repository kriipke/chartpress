// SPDX-License-Identifier: Apache-2.0
import { describe, expect, it } from "vitest";
import {
  DEFAULT_RULES,
  cleanSubchart,
  isKebab,
  normalizeSpec,
  resolveTraits,
  specWarnings,
  traitOverrides,
} from "./spec.js";

describe("spec contract", () => {
  it.each([
    ["api", true], ["api-server-2", true], ["Api", false], ["-api", false], ["api_2", false], ["", false],
  ])("validates kebab-case %s", (value, valid) => {
    expect(isKebab(value)).toBe(valid);
  });

  it("resolves dependent defaults without invalid combinations", () => {
    expect(resolveTraits({ pattern: "worker" }, DEFAULT_RULES)).toMatchObject({ exposure: "none", port: 0, ingress: false });
    expect(resolveTraits({ pattern: "api-microservice", workload: "daemonset" }, DEFAULT_RULES).scaling).toBe("fixed");
  });

  it("normalizes missing collections and strips UI-only state", () => {
    expect(normalizeSpec(null).subcharts).toHaveLength(1);
    expect(cleanSubchart({ name: " api ", pattern: "worker", expanded: true, port: 0 })).toEqual({
      name: "api", description: "", pattern: "worker",
    });
  });

  it("reports meaningful overrides and exposure warnings", () => {
    // admin-dashboard already defaults to ingress: true, so it takes an
    // explicit override of a *different* trait to produce a chip; the
    // ingress-exposure warning fires from the pattern default alone.
    const chart = { name: "admin", pattern: "admin-dashboard", workload: "statefulset" };
    expect(traitOverrides(chart, DEFAULT_RULES)).toContain("workload");
    expect(specWarnings({ rules: DEFAULT_RULES, subcharts: [chart] })[0]).toContain("protect this route");
  });
});
