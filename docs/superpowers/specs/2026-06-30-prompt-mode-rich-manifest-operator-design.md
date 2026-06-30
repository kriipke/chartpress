# Design: Prompt-mode + Rich Manifest + Operator Pipeline

- **Date:** 2026-06-30
- **Status:** Draft — awaiting review
- **Repo:** `github.com/kriipke/chartpress` (`/Users/spencer/chartpress`)
- **Source of the ported feature:** the `~/chartpress-zinc-devops` fork (`ziinc-platform/chartpress`)

## 1. Goal

Add a "generate from a natural-language prompt" path to the deployed web app, and in
doing so promote chartpress from a synchronous template-copier into a declarative,
operator-driven pipeline that renders a **rich `ChartpressConfig`** (description +
per-subchart workloads + a fully-honored `rules` block).

User-facing flow:

1. Land on a **choose screen**: *Generate manually* or *Generate from a prompt*.
2. **Manual** → the rich form (empty).
3. **Prompt** → enter an app **name** + a **text description** → the spec is drafted by
   an LLM → land on the **same rich form, pre-filled and editable**.
4. Submit → the chart is generated asynchronously → **browse the list of charts** and
   download via the Charts view.

## 2. Architecture

Four stages — drafting → admission → rendering → discovery:

```
                          ┌─────────────────────────────────────────────┐
 browser (React SPA)      │ backend (cmd/server, internal/server)        │
 ───────────────────      │                                              │
  Prompt screen  ── POST /text-to-config ──▶ draft spec via OpenAI ──┐    │
                          │                                          │    │
  Rich form      ── POST /generate (spec) ─▶ wrap spec → manifest    │    │
                          │                  SSA-apply ChartpressConfig CR │
                          │                          │                │    │
  Charts browser ─ GET /charts (poll) ──────▶ list CRs + mint presigned   │
                          │                    download URLs (read S3)│    │
                          └──────────────────────────┼────────────────────┘
                                                     │ (CR applied)
                                                     ▼
                          ┌─────────────────────────────────────────────┐
                          │ operator (cmd/operator) — client-go watch    │
                          │  reconcile ChartpressConfig:                 │
                          │   spec → rich chart render → zip → S3 upload │
                          │   → status{phase, artifactKey, ...}          │
                          └──────────────────────────┬───────────────────┘
                                                     ▼
                                       S3-compatible bucket (BYO, external)
                                       charts/<name>.zip  (presigned GET)
```

**Role split (locked):**

- **`/text-to-config`** — prompt → **`.spec` contents only** (no envelope). LLM-drafted.
- **`/generate`** — takes a **spec**, wraps it into a full `ChartpressConfig` manifest
  (`apiVersion`/`kind`/`metadata`), **server-side-applies it as a CR**, returns
  immediately (async). Does **not** render.
- **Operator** — watches `ChartpressConfig` CRs, runs the **rich generation engine**,
  uploads the zip to object storage, writes `status`.
- **Frontend** — drafts/edits the spec, applies via `/generate`, and **browses** charts
  via `/charts` (the web app is a forward-compatible stand-in for a future GitOps
  operator that would apply the same manifests).

## 3. Data model

### 3.1 Spec (the `/generate` request body, and the CR `.spec`)

```jsonc
{
  "umbrellaChartName": "demo-platform",      // required, kebab-case
  "description": "Example platform chart",    // optional
  "subcharts": [                              // required, >= 1
    { "name": "api", "workload": "deployment", "description": "" }
  ],
  "rules": {                                  // optional; missing → defaults below
    "ingress": "alb",                         // single enum: alb|nginx|traefik|istio|gce|none
    "common_annotations": false,
    "linked_templates": true,
    "resource_names_match_chart_name": false,
    "shared_secrets_config": false,
    "shared_newrelic_config": false,
    "generate_umbrella_readme": true,
    "generate_subchart_readme": true,
    "include_docs": true
  }
}
```

**Rule defaults (applied when `rules` absent or a field omitted):**
`ingress: "alb"`, `linked_templates: true`, `generate_umbrella_readme: true`,
`generate_subchart_readme: true`, `include_docs: true`, everything else `false`.
This makes the current web app's `{ umbrellaChartName, subcharts }` body a **minimal
valid spec** whose output ≈ today's output (backward compatible — no separate legacy
branch).

**Key change vs the zinc fork:** `rules.possible_ingresses: []string` →
`rules.ingress: string` (single, platform-wide controller), and workload enum is
**3 values** (job/cronjob removed).

### 3.2 Manifest (produced by `/generate`, stored as the CR, returned to the user)

```jsonc
{
  "apiVersion": "chartpress.dev/v1alpha1",
  "kind": "ChartpressConfig",
  "metadata": { "name": "demo-platform" },   // == spec.umbrellaChartName
  "spec": { /* the spec above */ }
}
```

### 3.3 CR `status` (operator-owned)

```jsonc
{
  "phase": "Pending | Generating | Ready | Failed",
  "observedGeneration": 3,
  "artifactKey": "charts/demo-platform.zip",
  "lastGenerated": "2026-06-30T03:10:00Z",
  "message": ""
}
```

## 4. The rich generation engine (Go-side, scaffold-only)

**Altitude (locked):** the manifest is a **structural scaffold**. It controls which
subcharts exist, each subchart's workload, which optional artifacts are emitted, and the
rule-driven structure. The per-component `values.yaml` ships as the **template's existing
example defaults** for the user to edit after download — the manifest never carries
concrete values (images/ports/hosts).

**Mechanism (locked):** **Go-side template generation** — the generator programmatically
emits/rewrites template files and `values.yaml`/`Chart.yaml` entries per rule (not
values-gated partials). Correctness is guarded by **golden-file tests** that
`helm template` the output and assert.

### 4.1 Workload → template selection

- Allowed: `deployment | statefulset | daemonset`. `job`/`cronjob` are **rejected** at
  the form, the LLM JSON-schema enum, server validation, and the CRD enum (no templates
  exist for them; faking them would be dishonest).
- Today every subchart emits `deployment.yaml → {{ include "umbrella-chart.deployment" }}`.
  The engine instead emits, per subchart, the workload-appropriate manifest file that
  includes the matching umbrella named template (`umbrella-chart.statefulset` /
  `…daemonset`), which already exist.

### 4.2 The six rules (all fully built out)

| Rule | `true` behavior | Default |
|---|---|---|
| `linked_templates` | Subchart manifests are thin `{{ include "umbrella-chart.<kind>" . }}` against the shared umbrella `.tpl` partials (DRY). | `true` |
| `linked_templates: false` | **Fully inlined, self-contained subcharts**: the engine inlines each umbrella named-template *body* (stripping `define`/`end`) into the subchart's own `templates/`, removing the include-stubs, so a subchart has zero dependency on umbrella partials. *All other rules are then applied to the per-subchart inlined copies instead of the shared partials.* | — |
| `resource_names_match_chart_name` | Resource `metadata.name`/selectors use exactly `{{ .Chart.Name }}`; engine rewrites each subchart `<name>.fullname` helper to emit `{{ .Chart.Name }}`. `false` keeps the release-prefixed `fullname`. | `false` |
| `common_annotations` | Engine wires `global.commonAnnotations` (seeded `app.kubernetes.io/part-of: <umbrella>`, `chartpress.dev/managed: "true"`) and rewrites the annotations helper to merge it onto **every resource** (alongside the existing checksums). | `false` |
| `shared_secrets_config` | Engine emits an umbrella-level Opaque `Secret` `<umbrella>-shared-secrets` (placeholder `stringData` from `global.sharedSecrets.data`) and wires **every subchart** to consume it via `envFrom: [{ secretRef }]` (rides the deployment template's existing `.Values.envFrom`). First concrete umbrella-rendered resource. | `false` |
| `shared_newrelic_config` | Engine emits an umbrella `ConfigMap` `<umbrella>-newrelic-config` (`NEW_RELIC_ENABLED`, distributed-tracing, `NEW_RELIC_LABELS`) **and** a dedicated `Secret` `<umbrella>-newrelic-license` (placeholder `NEW_RELIC_LICENSE_KEY`); every subchart gets `envFrom` the ConfigMap + `secretKeyRef` for the license, plus a **per-subchart `NEW_RELIC_APP_NAME` = subchart name**. Decoupled from `shared_secrets_config`. | `false` |
| `ingress` (single enum) | Platform-wide controller. `alb`/`nginx`/`traefik`/`gce` → `Ingress` with the right `ingressClassName` + controller annotations; **`istio` → `Gateway` + `VirtualService`**; `none` → ingress disabled. Per-subchart host/path stays in subchart values; the `{{- if .Values.ingress.host }}` guard stays. Legacy `aws` maps to `alb`. | `"alb"` |

### 4.3 Optional-artifact rules (file toggles)

- `generate_umbrella_readme: false` → drop umbrella `README.adoc`.
- `generate_subchart_readme: false` → drop each subchart `README.adoc`.
- `include_docs: false` → drop the umbrella `docs/` directory.
- Implemented by removing entries from the loaded chart's files before
  `chartutil.SaveDir`. Default `true` when omitted (so a minimal/legacy spec still gets
  READMEs + docs rather than Go zero-value `false` silently stripping them).

### 4.4 Descriptions

- `spec.description` → umbrella `Chart.yaml` `description`.
- `subcharts[].description` → that subchart's `Chart.yaml` `description`.
- Empty → fallback `"<name> chart generated by chartpress"`.

## 5. Backend (`internal/server`)

- **`POST /text-to-config`** — body `{ "prompt": "..." }` → returns the **spec**
  (`{ umbrellaChartName, description, subcharts, rules }`) drafted by the LLM. Uses the
  **OpenAI Responses API**, model **`gpt-4.1`** (override via `OPENAI_MODEL`), strict
  JSON-schema structured output, key from `OPENAI_API_KEY`. Ported from the zinc fork's
  `draftManifestFromPrompt` but the schema is the **spec** (no envelope) with the
  `ingress` single-enum + 3-workload changes.
- **`POST /generate`** — body = **spec**. Validate (§7) → wrap into manifest
  (`metadata.name = umbrellaChartName`) → **server-side apply** the `ChartpressConfig`
  CR (field manager `chartpress-backend`) into the backend's namespace
  (`POD_NAMESPACE`, downward API) → return `{ name, namespace, phase: "Pending",
  manifestYaml }`. Async — no rendering here.
- **`GET /charts`** — list `ChartpressConfig` CRs (via `client-go`) → array of
  `{ name, phase, subchartCount, lastGenerated, message, downloadUrl }`, where
  `downloadUrl` is a **freshly minted presigned GET URL** from `status.artifactKey`
  (only for `phase=Ready`).
- **`GET /charts/<name>`** — same shape for one CR (focused polling).
- **`/download/`** — retained but, in the object-storage model, downloads are presigned
  direct-to-bucket; `/charts` carries the URLs. (Legacy local `/download` may be dropped
  once nothing serves local artifacts.)

## 6. Operator (`cmd/operator`)

- Separate binary, **same container image** as the backend, different entrypoint.
  `client-go` **informer/watch loop** (no controller-runtime; `client-go`/`apimachinery`
  v0.32.2 already in `go.mod` transitively via Helm). **One replica** (no leader
  election).
- **Reconcile (level-based):** on add/update where
  `status.observedGeneration != metadata.generation` → `phase=Generating` → run the §4
  engine from `spec` → zip → upload to `charts/<name>.zip` (overwrite) → set
  `phase=Ready`, `observedGeneration`, `artifactKey`, `lastGenerated`. Any error →
  `phase=Failed`, `message`.
- **Finalizer** (`chartpress.dev/artifact-cleanup`): on CR delete, remove the
  bucket object so the browse list stays consistent with live CRs.
- Templates (`templates/umbrella`, `templates/subchart`) are baked into the image.

## 7. Validation (`/generate` and operator, defense-in-depth)

- `umbrellaChartName` present and matches `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`.
- `subcharts` length ≥ 1; each `name` matches the same regex.
- `workload ∈ { deployment, statefulset, daemonset }`.
- `rules.ingress ∈ { alb, nginx, traefik, istio, gce, none }`.
- **No** `metadata.name == umbrellaChartName` check at `/generate` (that's an
  envelope concern; the backend sets `metadata.name` itself when wrapping).
- The CRD `openAPIV3Schema` enforces structure at admission; the operator marks `Failed`
  on any render error.

## 8. Object storage (BYO, external, S3-compatible)

- Client: `github.com/minio/minio-go/v7` (new direct dep; works with AWS S3 / R2 /
  MinIO). Config via Helm `values` + Secret: `endpoint`, `bucket`, `region`,
  `accessKey`, `secretKey`, `useSSL`.
- **Operator** holds write creds (upload). **Backend** holds creds to **mint presigned
  GET URLs** at browse time (always-valid links; the bucket stays private otherwise).
- Downloads are **presigned direct-to-bucket** → the bucket endpoint must be
  browser-reachable (external S3/R2 already is). **No bundled MinIO.**
- Key scheme: `charts/<name>.zip`, latest-per-CR (overwrite). No history in v1.

## 9. CRD changes (`crds/crd-helmchart.yaml`)

The CRD (`chartpressconfigs.chartpress.dev`, kind `ChartpressConfig`, `v1alpha1`,
Namespaced, status subresource — already present) is updated to match the spec model:

- `rules.possible_ingresses: []` → **`rules.ingress: string`** with
  `enum: [alb, nginx, traefik, istio, gce, none]`.
- Workload `enum` → drop `job`, `cronjob` (keep `deployment`, `statefulset`,
  `daemonset`).
- Add `spec.description: string` and `subcharts[].description: string`.
- Document the `status` fields (§3.3) in the schema.
- Update the example CRs (`helmchart-iot.yaml`, `helmchart-ml.yaml`) and `crds/README.md`
  to the new shapes.

`rules` stays **required** in the CRD `spec`. This is consistent with the
"optional within the spec" rule (§3.1): `/generate` **normalizes and fills the default
rules before applying the CR**, so the persisted `ChartpressConfig` always carries a
complete `rules` block even when the inbound `/generate` body omitted it.

## 10. Packaging & deploy (Helm `chart/`)

- **CRD install** (`crds/` shipped/applied with the chart).
- **Operator Deployment** (same image, `command: [operator]`) + **ServiceAccount** +
  **Role/RoleBinding**: `get/list/watch` + `update`/`patch` `chartpressconfigs` and
  `chartpressconfigs/status`, plus finalizer update.
- **Backend RBAC**: `create/get/list/watch` `chartpressconfigs`.
- **Secrets/values:** S3 config Secret (operator: write; backend: presign) and
  `OPENAI_API_KEY` Secret (backend), with `OPENAI_MODEL` value (default `gpt-4.1`).
  Follows the existing `backend.openai` pattern and the repo's Secret-wiring conventions.
- Downward-API `POD_NAMESPACE` env on the backend.

## 11. Frontend (`web/src`)

Lightweight screen state-machine in `App.jsx` (no router dep). Two nav areas:

- **Generate (wizard):**
  - *Choose* → two cards: Manual / From a prompt.
  - *Prompt* → **App name** + **Describe your app** → `POST /text-to-config` → pre-fill
    the rich form from the returned spec, with the **typed name overriding** the LLM's
    `umbrellaChartName`; everything editable.
  - *Rich form* (shared): `umbrellaChartName`, `description`; subchart rows
    (`name`, `workload` dropdown ×3, optional `description`); rules section
    (`ingress` dropdown ×6, 8 checkboxes seeded to locked defaults); structure preview.
    Submit → `POST /generate` → Charts.
- **Charts (browser, permanent nav destination):** polls `GET /charts` (~2–3s while any
  chart is Pending/Generating); each row shows name, **phase badge**, subchart count,
  `lastGenerated`, a **Download** button (presigned) when `Ready`, or `message` when
  `Failed`.

Vite `base`/asset-path handling is unchanged (served at root).

## 12. Testing

- **Generation engine:** golden-file tests — `helm template` the generated chart for
  representative specs (each rule on/off, each workload, each ingress controller incl.
  istio, `linked_templates` true/false) and assert on rendered output; `helm lint`.
- **Backend:** unit tests for spec validation, normalization, manifest wrapping, and
  legacy-minimal-spec defaulting.
- **Operator:** reconcile test against a `client-go` **fake client** (+ a fake/stub S3
  uploader): assert phase transitions, `artifactKey`, finalizer behavior.

## 13. Phased build plan

1. **Engine** — rich generation in `internal/` (workload selection, 6 rules,
   readme/docs toggles, descriptions) behind a `GenerateChart(spec)` function +
   golden-file tests. No API/operator/frontend changes yet.
2. **Backend + CRD** — spec model, validation, `/generate` (wrap + SSA apply),
   `/text-to-config` (spec), `/charts`; update the CRD + examples; RBAC for the backend.
3. **Operator** — `cmd/operator`, watch loop, reconcile→render→S3→status, finalizer;
   Helm operator Deployment + RBAC + S3/OpenAI wiring.
4. **Frontend** — choose/prompt/rich-form wizard + Charts browser + polling.

Each phase is independently reviewable and leaves the tree building.

## 14. Out of scope (explicit)

- `job`/`cronjob` workloads (no templates; rejected for now).
- The manifest driving concrete `values.yaml` content (images/ports/hosts) — scaffold
  only.
- Bundled in-cluster MinIO; artifact history/versioning; full `status.conditions[]`.
- A real GitOps operator that watches manifests in git (the web app's `/generate` is the
  current applier; the cluster operator is built here).
- Leader election / multi-replica operator.

## 15. Open risks

- **istio** rendering (`Gateway` + `VirtualService`) is the heaviest engine piece and a
  different resource model than the `Ingress` controllers — highest implementation risk.
- **`linked_templates: false`** inlining is Go string-surgery on template bodies; golden
  tests are essential to keep it honest.
- **Presigned + private bucket** requires the bucket endpoint be browser-reachable;
  documented as a deploy prerequisite.
- Backend↔apiserver and operator↔apiserver **RBAC** must be provisioned for the pipeline
  to work end-to-end (consistent with the repo's existing self-provisioned-RBAC pattern).
