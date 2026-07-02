# Subchart Configurator — Design

Status: proposed, rev 3 · 2026-07-02 · companion to [subchart-patterns.md](subchart-patterns.md)

Rev 2 reworked the design **pattern-first**: the primary input per subchart is a
named pattern, not a trait questionnaire. Rev 1's questionnaire survives as the
Custom escape hatch and the override panel; its trait vocabulary survives
unchanged as the engine layer. The reasoning for the flip is summarized under
[Why pattern-first](#why-pattern-first).

Rev 3 records the design-review resolutions: intent storage with frozen
pattern defaults, dependent defaulting, a warnings tier in validation, the
single-file sectioned handoff, the curated dependency registry, closed-set
text-to-config classification, the registry contract test, and handoff
emission merged into phase 1.

## Problem

A Subchart today carries only `name`, `workload`, `description`. The generator
treats every subchart as the same kind of thing: a horizontally-scalable HTTP
service. `service.yaml`, `ingress.yaml`, `hpa.yaml`, and `networkPolicy.yaml`
are emitted unconditionally, probes default to `httpGet`, and shared-config
injection is all-or-nothing. Real platforms mix API servers, gRPC backends,
queue workers, stream processors, and singletons — and the product thesis
(scaffold with best practices front-loaded, plus a handoff prompt an LLM
finishes against the app code) needs to know which one each subchart is.

## Why pattern-first

Rev 1 gathered five orthogonal traits per subchart (exposure, port, ingress,
scaling, shared_env) and derived the output from them. Four things overturned
that:

1. **Traits can't produce the handoff prompt.** Patterns with identical traits
   have different handoff checklists (API microservice vs webhook ingest;
   realtime gateway vs admin dashboard). The pattern name is the key that
   carries half the product; five trait answers under-specify it.
2. **The trait space is mostly incoherent.** Twelve named patterns cover the
   real combinations; the full trait product is dominated by meaningless ones.
   Patterns are the coherent points in trait space.
3. **Recognition beats derivation for this audience.** The target user is the
   developer who bolts practices on late — asking "can more than one replica
   run at once?" presumes the knowledge the tool exists to supply. Picking
   "Stream processor" doesn't. The pattern *is* the teaching.
4. **The LLM handoff makes fine-grained questioning redundant.** Whatever the
   scaffold doesn't pin down, the handoff step answers later *from the code* —
   a better source than the user's memory at scaffold time.

The rev-1 objection to a single enum ("axes are orthogonal — a gRPC server can
be a singleton") is answered by overridability: the pattern sets trait
*defaults*, not constraints, so off-pattern workloads stay expressible without
new patterns.

## Concepts

Two glossary terms (to be added to CONTEXT.md when this lands):

> **Pattern**: The named workload shape a subchart is declared as (`pattern:
> worker`). Expands to Trait defaults, selects the pattern's front-loaded
> extras, and keys the generated handoff checklist. The canonical list lives in
> [subchart-patterns.md](subchart-patterns.md); `custom` opts out of defaults.
> _Avoid_: role, type, kind, template

> **Traits**: The per-subchart generation controls that describe how a subchart
> is templated — how it is contacted (`exposure`, `port`, `ingress`), whether
> it can scale (`scaling`), and whether it consumes the platform's shared
> configuration (`shared_env`). Defaulted by the Pattern, individually
> overridable, and the vocabulary the engine actually generates from.
> _Avoid_: options, profile, flags

## Spec shape

```yaml
umbrellaChartName: saas-platform
subcharts:
  - name: orders-api
    pattern: api-microservice        # the primary input
  - name: emailer
    pattern: worker
  - name: dispatcher
    pattern: scheduler
  - name: pricing
    pattern: grpc-service
    ingress: true                    # trait override: the rare externally-exposed gRPC
  - name: legacy-daemon
    pattern: custom                  # escape hatch: traits stated explicitly
    workload: statefulset
    exposure: tcp
    port: 5432
    scaling: fixed
```

Pattern ids (kebab-case, mirroring the patterns doc): `api-microservice` ·
`grpc-service` · `edge-gateway` · `web-frontend` · `worker` ·
`stream-processor` · `scheduler` · `realtime-gateway` · `ml-inference` ·
`webhook-ingest` · `admin-dashboard` · `node-agent` · `custom`.

Trait keys stay flat on the subchart entry, in the same style as `workload`:
`exposure` ∈ http · grpc · tcp · none; `port` int; `ingress` bool; `scaling` ∈
auto · fixed · singleton; `shared_env` bool. `workload` itself is now also
pattern-defaulted (e.g. `node-agent` → `daemonset`) and overridable like any
trait.

## Resolution semantics

**The spec stores intent, not resolution.** A persisted ChartpressConfig (or
CR) carries only `pattern` plus the trait keys the user explicitly wrote;
resolution is recomputed on every generate. Two consequences:

- **Pattern defaults are frozen once shipped.** The summary table in the
  patterns doc is an API contract, not documentation: changing a shipped
  pattern's defaults requires a new pattern id or a major version. Old specs
  therefore keep meaning the same chart.
- **Trait fields are pointers** in the Go schema (`*string`, `*int`, `*bool`)
  so "unset" is distinguishable from a zero value — unlike `Rules`, whose
  plain bools are never pattern-defaulted.

Resolution happens in shared engine code (`engine.Resolve`), called by the
HTTP/CLI decode layer and by the operator at reconcile time, before
Normalize/Validate:

1. `pattern` omitted → `api-microservice` (back-compat, see below).
2. **Explicit trait keys are read first.** They are the user's intent and the
   only keys validation may reject.
3. Remaining traits are filled by **dependent defaulting** — defaults are
   functions of the already-resolved keys, not a flat merge:
   - `workload` ← pattern default.
   - `exposure` ← pattern default.
   - `port` ← 8080 for `http`/`tcp`, 50051 for `grpc`, absent for `none`.
   - `ingress` ← pattern default *if compatible* with the resolved exposure
     (`http`/`grpc` and `rules.ingress != none`), else `false`.
   - `scaling` ← pattern default; ignored entirely for `daemonset` (which
     never gets an HPA — a daemonset is per-node by definition).
   - `shared_env` ← pattern default.

   The governing principle: **a default can never cause a validation error;
   only keys the user actually wrote can fail.** `pattern: api-microservice`
   with `exposure: tcp` resolves to `ingress: false` silently; `pattern:
   custom` with `exposure: none` and `ingress: true` still errors, because
   `ingress` was explicit.
4. `pattern: custom` defaults like `api-microservice` (so partial specs stay
   valid — dependent defaulting keeps them coherent), selects no pattern
   extras, and gets a generic handoff section.

Generated output is then:

```
scaffold = base template ⊕ trait tailoring ⊕ pattern extras
handoff  = pattern checklist, adjusted by trait overrides
```

Trait tailoring is rev 1's engine mapping, unchanged. Pattern extras are the
per-pattern front-loaded items that traits alone don't express — №9's
`startupProbe` stub, №11's "protect this route" README warning, №8's
session-affinity note, №6's "replicas ≤ partitions" comment.

## Engine mapping — resolved trait → generated output

Unchanged from rev 1; this is the tailoring layer (`internal/engine/traits.go`),
now fed by pattern resolution instead of directly by the user:

| Trait value | Effect on the subchart |
|---|---|
| `exposure: none` | Drop `service.yaml`, `ingress.yaml`, `networkPolicy.yaml`. Remove `service`/`ingress`/`networkPolicy` values blocks and `httpGet` probes; no container `ports`. |
| `exposure: http` | Today's output, unchanged. |
| `exposure: grpc` | `appProtocol: grpc`; `grpc:` probes; controller-specific gRPC ingress annotations (nginx `backend-protocol: GRPC`, ALB `backend-protocol-version: GRPC`; istio none). |
| `exposure: tcp` | Service kept; `tcpSocket` probes; `ingress` forced off. |
| `ingress: false` | Drop `ingress.yaml` and the values ingress block. |
| `scaling: auto` | Keep `hpa.yaml` and the full `podCount` block — static by default with the dynamic (HPA) subtree ready to switch on. "Auto" means autoscaling is *available*, sized during handoff, not forced on before resources are sized (also the back-compat anchor: today's output). |
| `scaling: fixed` | Drop `hpa.yaml`; `podCount.type: static` only. |
| `scaling: singleton` | Drop `hpa.yaml`; pinned `static: 1` with `# singleton — do not raise`; `strategy: Recreate`; `pdb` dropped; statefulset renders `replicas: 1`. |
| `shared_env: false` | `applySharedSecretsSubchart` / `applySharedNewrelicSubchart` skip this subchart; umbrella-level Secret/ConfigMap still render. |

## Validation invariants (`engine.Validate`, post-resolution)

Errors (blocking) — by construction of dependent defaulting, only explicit
user-written keys can trip these:

- `pattern` ∈ the id list above.
- `exposure`/`scaling` ∈ their enums; `port` in 1–65535 when `exposure != none`.
- `ingress: true` requires `exposure` ∈ {http, grpc} **and** `rules.ingress != none`.
- `scaling: singleton` invalid with `workload: daemonset`.
- Overrides are validated by the same invariants — no special cases. A
  contradictory override (e.g. `pattern: node-agent` + `scaling: singleton`)
  fails like any hand-written spec would.
- `shared_env: true` with both shared rules off stays a silent no-op.

**Warnings (non-blocking, new tier)** — `Validate` returns warnings alongside
the error; they surface in the web UI, CLI output, the CR `status.message`,
and the generated handoff. Initial set:

- An `edge-gateway` subchart coexists with other `ingress: true` subcharts —
  the pattern's premise is "the only public entry"; intended?
- An `admin-dashboard` subchart is reachable via ingress — "protect this
  route" (until an ingress-class/auth trait exists).

Warnings never mutate resolution: cross-subchart *auto-demotion* (selecting
`edge-gateway` flipping siblings' ingress off) was considered and rejected —
resolution stays a pure function of one subchart entry.

## The handoff artifact

**One `HANDOFF.md` at the umbrella root**, generated as a sectioned document
(`handoff.go` is a section assembler, not a per-chart emitter):

1. **Protocol header** — ground rules (best practices are load-bearing;
   placeholders replaced, never deleted), the validation loop (`make
   template/lint/test`), and the completion semantics.
2. **Platform-level items** — umbrella values layering, `dependencies:`
   wiring, and any warnings emitted by `Validate`.
3. **One generated section per subchart**, keyed by its pattern: the
   pattern's checklist as verifiable questions, adjusted by trait overrides
   (e.g. `ingress: true` on `grpc-service` adds the external-gRPC items);
   `custom` gets the generic checklist.

Completion is incremental: **delete a subchart's section** when that
component is finished; **delete the file** when the chart is done. The
generated README links to it while it exists. Emission is gated by a new
umbrella rule `generate_handoff` (default **true** — it's the product
thesis), so strict no-handoff output remains available; `AGENTS.md`,
`CLAUDE.md`, and `docs/best-practices.adoc` remain unconditional (they
already say "if present" about the handoff).

Two recorded trade-offs: a subchart extracted for standalone use (the
`linked_templates: false` path) leaves its handoff behind — accepted; and the
static generic `HANDOFF.md` that shipped in `templates/umbrella/` during the
interim was **removed when phase 1 landed** — the generated file is the only
handoff (never both, or the checklists drift).

## Backward compatibility

Omitted `pattern` resolves to `api-microservice`, whose trait defaults
(`deployment · http · 8080 · ingress · auto · shared_env`) reproduce today's
manifests byte-for-byte. Existing ChartpressConfigs, presets, CRD resources,
and fixtures need no edits. The only new output for old specs is `HANDOFF.md`,
which is additive and gated by `rules.generate_handoff`.

## Web UI

**The pattern picker is the configurator.** Adding a subchart presents a card
grid — twelve patterns plus Custom. Each card: pattern name, a "like
`orders-api`" example line (recognition beats abstract description), and a
badge strip of what it front-loads (`no service` / `singleton` / `grpc
probes`). Selecting a card fills the row.

- **Collapsed row**: name · pattern chip · any override chips
  (`grpc-service · +ingress`). Scannable when a prompt draft lands with six
  subcharts.
- **Advanced disclosure**: the trait controls from rev 1, pre-filled with the
  pattern's defaults; edits become overrides, with a "Reset to pattern"
  affordance. Conditional visibility and impossible-combination prevention
  carry over from rev 1.
- **Custom card**: opens the rev-1 five-question flow (contacted how? port?
  external? replicas? shared env?) — the questionnaire's only remaining
  primary role.
- **StructurePreview** renders from resolved traits (worker rows lose
  `service.yaml`/`ingress.yaml`, singletons lose `hpa.yaml`) and shows
  `HANDOFF.md` in the tree.

`spec.js` imports the registry from `web/src/app/patterns.json` (the verbatim,
CI-checked copy of the engine's embedded registry), keeps
`EXPOSURES`/`SCALING_MODES` for the advanced panel, and `normalizeSpec`
mirrors `engine.Resolve` (dependent defaulting included) client-side so the
preview, warnings, and validation match the server.

## Text-to-config

The server prompt classifies each described component into a pattern id —
twelve labels plus overrides is a far more robust LLM target than four
independent trait inferences that can come back incoherent. Inference hints:
"worker / consumer / executor / processor" → `worker`; "Kafka consumer /
consumer group / CDC" → `stream-processor`; "scheduler / migrator /
leader-elected" → `scheduler`; "webhook" → `webhook-ingest`; "admin / internal
dashboard" → `admin-dashboard`.

**Classification contract (closed set, forced choice):** the model must
always pick the nearest of the twelve ids; `custom` is *reserved for humans*
and never emitted; trait overrides are emitted only for facts stated in the
user's text ("on port 3000", "can't run two at once"), never inferred. The
misclassification cost is bounded by the RichForm review step — a
wrong-but-plausible pattern chip is one click to correct, which beats both a
meaningless `custom` and a blocking "unclassified" state.

The two guardrails from the patterns doc (dependency rule, sidecar rule) are
part of the same prompt. When the dependency rule fires, chartpress emits the
umbrella `dependencies:` stanza and a values-block skeleton (resolved: emit,
don't just recommend) — sourced **only** from a curated, version-pinned
registry shipped as in-repo data and frozen per release like pattern
defaults; infrastructure not in the registry degrades to a stub entry with
`TODO(chartpress)` markers for repository and version. Text-to-config decides
*which* dependency, never *where from*.

## Surfaces touched

1. **Engine** — `types.go` (Subchart fields: `pattern` + pointer trait keys,
   enums), `resolve.go` (shared resolution: dependent defaulting, called by
   server decode and operator reconcile), `traits.go` (tailoring),
   `patterns.json` + `patterns.go` (embedded registry: defaults, labels,
   examples, badges, checklist snippets; frozen per release),
   `dependencies.json` (curated upstream-chart registry), `handoff.go`
   (sectioned HANDOFF.md assembler), `rules.go` (shared-config functions take
   the Subchart; new `generate_handoff` rule), `ingress.go` (gRPC
   annotations). Tests per concern: `resolve_test.go`, `traits_*_test.go`,
   `handoff_test.go`, plus the registry contract test below.
2. **CRD** — subchart items gain `pattern` + the five trait properties (enum
   checks only — dependent defaulting can't be expressed in OpenAPI and stays
   in `engine.Resolve`); umbrella rules gain `generate_handoff`; validation
   stays reconcile-time (no admission webhook — matches the operator's
   existing design), with warnings surfaced in `status.message`; iot/ml
   examples rewritten in pattern vocabulary.
3. **HTTP API** — no route changes; `/generate` body grows the same fields.
4. **Web** — pattern picker, advanced overrides, Custom path, preview. The
   registry ships as a verbatim copy at `web/src/app/patterns.json` (the web
   Docker build context is `./web/`, so a shared file is off the table); a Go
   contract test compares it against the embedded engine copy and fails CI on
   drift.
5. **Text-to-config** — classification prompt + guardrails + dependencies
   emission.
6. **Presets** — rewritten pattern-first; `event-driven-platform` =
   `edge-gateway` + `api-microservice` + `worker` + `scheduler` with Kafka as
   a dependency (the showcase of patterns and the dependency rule together).

## Phasing

1. **Engine: patterns + traits + resolution + HANDOFF emission, CRD,
   presets** — useful from CLI/CRD alone; back-compat defaults make it a safe
   standalone release. Handoff emission ships *in* this phase, not after it:
   the moment output differentiates by pattern, identical-trait patterns
   (№1 vs №10) are distinguishable only by their handoff, and the interim
   static `HANDOFF.md` becomes actively wrong for workers/singletons ("fill
   in your probe endpoints" on a chart with no probes). The single-file
   sectioned shape makes emission cheap enough to include — checklist
   snippets live in the same registry entry as the trait defaults.
2. **Web pattern picker** — DONE. `spec.js` imports the CI-checked
   `patterns.json` and mirrors `engine.ResolveTraits` (dependent defaulting)
   plus `engine.Warnings`; RichForm rows are a chevron-expandable card with a
   collapsed chip strip (pattern badge · override chips · summary line) and an
   expanded trait panel (the five questions, pre-answered by the pattern, with
   conditional visibility — port/ingress hidden for `none`, scaling hidden for
   daemonset, shared-env shown only when a shared rule is on); the pattern
   picker is a Dialog of the thirteen cards; StructurePreview prunes each
   subchart's manifest list from the resolved traits; warnings render in an
   amber banner above submit. New rows open expanded, drafted/loaded rows
   collapsed. Editing a trait to its pattern-default value drops the override
   (keeps the spec clean intent); "Reset to pattern" clears all overrides.
   `generate_handoff` also surfaced as an Output-files rule toggle.
3. **Text-to-config classification + dependency emission + operator warnings**
   — DONE. The OpenAI draft prompt classifies each component into one of the
   twelve pattern ids (closed set; `custom` reserved for humans; overrides only
   from stated facts) and lists infrastructure under a top-level
   `dependencies` array; the strict JSON schema models trait overrides as
   nullable (null = use the pattern default). `dependencies.json` +
   `dependencies.go` are the curated, version-pinned registry: a known key
   becomes a pinned umbrella `Chart.yaml` `dependencies:` entry (aliased to the
   key) with a values skeleton and a review-version TODO; an unknown key
   becomes a stub entry with TODO repository/version. The `Dependencies` spec
   field flows through the CLI, HTTP body, and CRD; the operator surfaces
   `engine.Warnings` in the Ready `status.message`; the web form carries and
   edits the dependency list (accent chip for known, amber "(TODO)" for
   stubs) and the preview annotates `Chart.yaml`.

   Discovered during the build (recorded as a known behavior, not a defect):
   Helm will not render a chart whose declared dependencies aren't fetched, so
   a generated chart *with* infrastructure dependencies needs `helm dependency
   build` (new `make deps` target) — and any TODO-stub repository resolved —
   before `make template`/`make test`. The no-dependency common case is
   unchanged and still renders with zero setup. This is inherent to declaring
   Helm dependencies, and the generated Makefile + HANDOFF document the step.

## Alternatives considered

- **Trait questionnaire as the primary input (rev 1 of this doc)** —
  superseded: under-specifies the handoff artifact, asks users to derive what
  patterns let them recognize, and duplicates work the LLM handoff does better
  from the code. Retained as the Custom path and the override panel.
- **A single `role` enum with no overrides** — rejected: forces an enum
  explosion for off-pattern workloads. Pattern-plus-overridable-traits keeps
  twelve patterns sufficient.
- **A nested `traits:` object** — rejected: `workload` set the precedent for
  flat keys; nesting complicates the CRD schema and the LLM target.
- **Handoff content inside the generated README** — rejected in favor of
  `HANDOFF.md`; different reader, different lifetime.
- **Per-subchart `HANDOFF.md` files (and a two-tier root+leaf variant)** —
  rejected in review: one sectioned root file keeps the ground rules
  single-sourced and progress visible in one place; the cost (an extracted
  standalone subchart leaves its handoff behind) is accepted and documented.
- **Storing resolved traits in the spec (snapshot)** — rejected: the spec
  stores intent; stability comes from freezing pattern defaults instead.
- **Cross-subchart auto-demotion for `edge-gateway`** — rejected: resolution
  stays per-subchart-pure; the warnings tier carries the compositional
  insight.
- **A served `GET /patterns` endpoint** — rejected: embedded JSON + a CI
  contract test gives one source of truth without runtime coupling.
- **LLM-chosen dependency repositories** — excluded by policy:
  nondeterministic and a supply-chain hole; the curated registry decides
  *where from*.
- **A metrics/scrape-port trait** (and №12's headless-Service nuance) —
  deferred: a handoff item in v1, a candidate trait later.
