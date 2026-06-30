# Information Architecture — chartpress web app (prompt-mode + charts browser)

- **Date:** 2026-06-30
- **Audience:** Claude Design (prototyping handoff)
- **Source design:** [2026-06-30-prompt-mode-rich-manifest-operator-design.md](2026-06-30-prompt-mode-rich-manifest-operator-design.md)
- **Scope of this doc:** the *user-facing* frontend only (screens, navigation, content, states, flows). Backend/operator/storage are summarized only where they shape the UI.

---

## 1. What the app does (one paragraph)

chartpress turns a structured spec into a downloadable Helm chart bundle. A user
describes an "umbrella" application made of one or more **subcharts** (workloads),
toggles a set of **rules** that shape the generated chart, and submits. Generation
is **asynchronous**: the submitted chart appears in a **Charts** list where it
progresses through statuses until a **Download** link is available. There are two
ways to fill in the spec: **manually** via a form, or **from a natural-language
prompt** that an LLM drafts into the same form for review/editing.

---

## 2. Top-level structure (sitemap)

Two permanent navigation areas. No deep routing (lightweight screen state, not a
router). Charts is always reachable.

```
chartpress
├── Generate  (wizard)
│   ├── Choose            ← entry: "manually" vs "from a prompt"
│   ├── Prompt            ← name + description → drafts the spec
│   └── Rich form         ← shared screen; the spec editor (manual or pre-filled)
└── Charts  (browser)     ← permanent destination; list of all generated charts
```

Primary user goal funnels into the **Rich form** (both paths land there), then
submission moves the user to **Charts**.

---

## 3. End-to-end screen flow

```
              ┌────────────────────────────────────────────────────────────┐
              │                         GENERATE                            │
              │                                                            │
   ┌──────────┤  [Choose]                                                  │
   │ entry    │   ┌─────────────────┐   ┌──────────────────────┐          │
   │          │   │ Generate        │   │ Generate from a      │          │
   │          │   │ manually        │   │ prompt               │          │
   │          │   └───────┬─────────┘   └──────────┬───────────┘          │
   │          │           │                        │                      │
   │          │           │              [Prompt]  ▼                      │
   │          │           │      ┌───────────────────────────────┐        │
   │          │           │      │ App name + Describe your app   │        │
   │          │           │      │  → submit → (LLM drafting…)    │        │
   │          │           │      └───────────────┬───────────────┘        │
   │          │           │                      │ pre-fills              │
   │          │           ▼                      ▼                        │
   │          │   ┌────────────────────────────────────────────┐         │
   │          │   │ [Rich form]  (shared, editable)             │         │
   │          │   │  name · description · subchart rows · rules │         │
   │          │   │  · structure preview · Submit               │         │
   │          │   └───────────────────┬─────────────────────────┘         │
   │          └───────────────────────┼─────────────────────────────────┘
   │                                  │ submit (async, returns immediately)
   │                                  ▼
   │          ┌────────────────────────────────────────────────────────────┐
   └─────────▶│  CHARTS  (browser)                                          │
              │  rows: name · phase badge · #subcharts · lastGenerated      │
              │        · Download (when Ready) / message (when Failed)      │
              │  polls every ~2–3s while anything is Pending/Generating     │
              └────────────────────────────────────────────────────────────┘
```

Key timing fact for the prototype: **submission does not block**. The user is
sent to Charts immediately and watches the new row transition
`Pending → Generating → Ready` (or `Failed`).

---

## 4. Screens in detail

### 4.1 Choose
Entry point of the Generate wizard.

- Two large, equal-weight cards:
  - **Generate manually** → opens the Rich form **empty**.
  - **Generate from a prompt** → opens the **Prompt** screen.
- No other content. This is a fork, not a form.

### 4.2 Prompt
Collects the minimum needed for the LLM to draft a spec.

| Field | Type | Notes |
|---|---|---|
| **App name** | text input | Becomes the chart name; **overrides** the LLM's suggested name. kebab-case. |
| **Describe your app** | multi-line textarea | Free-form natural language. The main input. |

- Primary action: **Draft / Continue** → calls the drafting endpoint.
- **Loading state is important here**: drafting is an LLM call (seconds). Show a
  clear "drafting your chart…" state; the button should be disabled/spinner.
- On success → transition to the **Rich form, pre-filled** from the draft, with
  the typed App name already applied over the LLM's name.
- On failure → stay on Prompt, show an error, keep the user's text intact.

### 4.3 Rich form (the core screen — shared by both paths)
This is the spec editor. Same component whether reached empty (manual) or
pre-filled (from prompt). Everything is editable.

**Section A — Umbrella (the app itself)**
| Field | Type | Required | Validation / default |
|---|---|---|---|
| Umbrella chart name | text | yes | kebab-case `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` |
| Description | text | no | free text |

**Section B — Subcharts (repeatable rows, ≥ 1 required)**
Each row:
| Field | Type | Required | Options / validation |
|---|---|---|---|
| Name | text | yes | kebab-case (same regex as above) |
| Workload | dropdown | yes | `deployment` · `statefulset` · `daemonset` (exactly 3) |
| Description | text | no | free text |

- Add-row and remove-row controls. Must always keep ≥ 1 row (don't allow
  removing the last one, or re-add a blank).

**Section C — Rules** (1 dropdown + 8 checkboxes; see §5 for meanings/defaults)
- **Ingress** dropdown (6 options): `alb` · `nginx` · `traefik` · `istio` · `gce` · `none`.
- 8 checkboxes, pre-seeded to the locked defaults (5 off, 3 on — see §5).

**Section D — Structure preview**
- A read-only preview of the chart structure that *would* be generated from the
  current form state (subcharts present, which optional files are included, etc.).
  Updates as the form changes. This is a confidence-builder, not an editor.

**Primary action: Submit** → fires the async generate call → navigate to **Charts**.

### 4.4 Charts (browser)
Permanent nav destination. Lists every generated chart.

- **Polls** the list endpoint every ~**2–3s** while any chart is
  `Pending` or `Generating`; can back off / stop polling when all are settled
  (`Ready`/`Failed`).
- Each row shows:
  | Column | Content |
  |---|---|
  | Name | umbrella chart name |
  | **Phase badge** | `Pending` · `Generating` · `Ready` · `Failed` (4 visual states) |
  | Subchart count | integer |
  | Last generated | timestamp |
  | Action / status | **Download** button when `Ready`; error **message** text when `Failed`; nothing actionable while `Pending`/`Generating` |
- Download is a direct (presigned) link — a normal download button is fine.
- **Empty state** needed (no charts yet → prompt the user toward Generate).

---

## 5. Rules reference (for labels, tooltips, defaults)

These shape the generated chart. Designer needs them for ordering, grouping,
default checkbox state, and tooltip microcopy.

| Rule (control) | Default | Plain-language tooltip |
|---|---|---|
| **Ingress** (dropdown) | `alb` | Which ingress controller the whole platform uses. `istio` produces a Gateway/VirtualService instead of an Ingress; `none` disables ingress. |
| `linked_templates` | **ON** | Subcharts share common template logic (DRY). Turn off to make each subchart fully self-contained. |
| `generate_umbrella_readme` | **ON** | Include a README for the umbrella chart. |
| `generate_subchart_readme` | **ON** | Include a README inside each subchart. |
| `include_docs` | **ON** | Include the `docs/` directory. |
| `common_annotations` | off | Add shared annotations (part-of, managed-by) to every resource. |
| `resource_names_match_chart_name` | off | Name resources exactly after the chart, without the release-name prefix. |
| `shared_secrets_config` | off | Create one shared Secret and inject it into every subchart. |
| `shared_newrelic_config` | off | Create shared New Relic config + license and wire it into every subchart. |

Suggested grouping for the UI: **"Output files"** (the 3 ON-by-default toggles +
docs) vs **"Chart behavior"** (the rest). Ingress sits at the top of the rules
section as the only dropdown.

---

## 6. Data the UI handles

### 6.1 Spec (what the form produces / consumes)
```jsonc
{
  "umbrellaChartName": "demo-platform",      // required, kebab-case
  "description": "Example platform chart",    // optional
  "subcharts": [                              // ≥ 1
    { "name": "api", "workload": "deployment", "description": "" }
  ],
  "rules": {                                  // optional; UI seeds defaults
    "ingress": "alb",
    "linked_templates": true,
    "generate_umbrella_readme": true,
    "generate_subchart_readme": true,
    "include_docs": true,
    "common_annotations": false,
    "resource_names_match_chart_name": false,
    "shared_secrets_config": false,
    "shared_newrelic_config": false
  }
}
```
The **prompt path** receives this same shape back from the LLM and uses it to
pre-fill the form.

### 6.2 Chart status (what each Charts row renders)
```jsonc
{
  "name": "demo-platform",
  "phase": "Pending | Generating | Ready | Failed",
  "subchartCount": 1,
  "lastGenerated": "2026-06-30T03:10:00Z",
  "message": "",            // shown when Failed
  "downloadUrl": "https://…" // present only when Ready
}
```

---

## 7. States the prototype must cover

| Screen | States to design |
|---|---|
| Prompt | idle · **submitting/drafting (LLM, seconds)** · error (keep input) |
| Rich form | empty (manual) · pre-filled (from prompt) · field validation errors · submitting |
| Charts | **empty** (no charts) · loading first fetch · steady list · live polling/transition · row `Ready` (download) · row `Failed` (message) |
| Phase badge | 4 distinct visuals: Pending, Generating (in-progress feel), Ready (success), Failed (error) |

Cross-cutting:
- **Async generation** is the defining UX moment — the transition of a fresh row
  from Pending → Generating → Ready should feel alive (the reason polling exists).
- **Validation messaging:** kebab-case names; at least one subchart; workload and
  ingress constrained to their enums (dropdowns make this mostly structural).

---

## 8. Component inventory (for a design system pass)

- Choice cards (2-up) — Choose screen.
- Text input (with kebab-case validation styling).
- Textarea — prompt description.
- Repeatable row group with add/remove — subcharts.
- Dropdown / select — workload (×3), ingress (×6).
- Checkbox group — 8 rules, pre-seeded.
- Read-only structure preview (tree/outline).
- Primary submit button (+ submitting/spinner state).
- Data table / list with rows — Charts.
- **Status badge** with 4 phases.
- Download button.
- Inline error text (form fields, failed charts, prompt failure).
- Empty state — Charts.

---

## 9. Out of scope for the UI (don't prototype)

- Concrete values (images, ports, hosts) — the chart ships example defaults the
  user edits *after* download. The form never collects these.
- `job` / `cronjob` workloads (only the 3 listed exist).
- Artifact history/versioning (latest-per-chart only; one Download per row).
- Any auth/login screens (not described in the source design).
