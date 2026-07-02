# Subchart Patterns — Outline (revised)

Status: outline, rev 4 · 2026-07-02 · companion to [subchart-configurator.md](subchart-configurator.md)

As of rev 3, **patterns are a first-class spec field** (`pattern: worker`) —
the primary input each subchart is configured by. Traits are the expansion
and override vocabulary underneath. Rev 2's open questions are resolved in
[Resolutions](#resolutions-rev-3); the full argument for the flip lives in
the companion doc.

Rev 4 folds in the design-review resolutions (recorded in full in the
companion doc): the summary table is **frozen** per release, defaults apply
**dependently** (a default can never cause a validation error), the handoff
is a **single sectioned root file**, dependencies are emitted from a
**curated registry**, and text-to-config classification is **closed-set**.

## Purpose

Chartpress produces the **first half** of a Helm chart: a scaffold that has
the best practices most developers bolt on late — or never — already in
place, plus a **handoff prompt** an LLM can use, together with the actual
application code, to finish the chart. Most charts are built piece-meal:
Deployment first, Service when something needs to reach it, probes after the
first bad rollout, PDB/networkPolicy/securityContext never. The scaffold
inverts that: the practices come first, the app-specific values come second.

Each pattern below therefore defines two halves:

- **Front-loaded** — what the generator emits, correct-by-construction, with
  the best practices that are knowable *without* seeing the app code.
- **Handoff** — what only the application code can answer. This is the
  contract for the generated prompt: per pattern, the questions the LLM must
  resolve against the codebase to finish the chart.

## Inclusion criterion

A pattern belongs on this list only if **the workload is the developer's own
code**. A half-finished chart plus "here's how to finish it against your
code" is meaningless for software the developer didn't write. That excludes
self-hosted infrastructure — databases, caches, brokers, search engines,
consensus stores, object storage, observability backends, off-the-shelf
collectors. Those are solved by mature upstream charts and belong in the
umbrella's `dependencies:` block, not in generated templates (see
[Guardrails](#guardrails-for-text-to-config)). The previous revision listed
them as patterns 8–10 and 14–18; they are deliberately dropped, not
forgotten.

Trait vocabulary (from the configurator design): `workload` ∈ deployment ·
statefulset · daemonset; `exposure` ∈ http · grpc · tcp · none; `ingress`
bool; `scaling` ∈ auto · fixed · singleton; `shared_env` bool. Each pattern
has a spec-facing id (see the [summary table](#pattern--traits-summary));
`pattern: <id>` expands to that row's trait defaults and any explicit trait
key on the subchart overrides. `pattern: custom` is the escape hatch: it
defaults like `api-microservice` (so partial specs stay valid under
dependent defaulting), selects no pattern extras, and gets the generic
handoff section.

## Front-loaded in every pattern

The cross-cutting practices the scaffold carries regardless of pattern —
this list *is* the product thesis, so it leads the doc:

- **Probes exist from day one** (type chosen by `exposure`), with the
  endpoint/command left as an explicitly marked handoff item — a scaffold
  probe that must be filled beats a probe added after the first bad rollout.
- **`securityContext`**: `runAsNonRoot`, `readOnlyRootFilesystem`, dropped
  capabilities — the practice least likely to be retrofitted, because by
  then something depends on running as root.
- **`resources` requests/limits** present with placeholder values and a
  comment convention the handoff prompt fills from observed/expected load.
- **PDB whenever more than one replica can run**; dropped for singletons
  (where it only blocks drains).
- **HPA only where it's valid** — the piece-meal failure mode in reverse:
  no autoscaling footguns on queue-coupled or singleton workloads.
- **networkPolicy** scoped to the pattern's actual traffic shape, not
  allow-all.
- **Standard labels, helpers, pinned `apiVersion`s, documented
  `values.yaml`**, graceful-shutdown scaffolding (`terminationGracePeriodSeconds`
  + preStop hook placeholder).

Per-pattern sections list only what they *add or change* on top of this.

---

## The patterns

### 1. API microservice
- **Examples**: REST backend, `orders-api`, `users-api`
- **Traits**: `deployment · exposure: http · port · ingress: true · scaling: auto · shared_env: true`
- **Front-loaded**: today's default output — the back-compat anchor every
  other pattern is a delta from.
- **Handoff**: actual listen port vs the placeholder; real health/readiness
  endpoints (and whether readiness checks dependencies); env vars and which
  come from shared config vs own Secret; graceful-shutdown behavior (drain
  time → `terminationGracePeriodSeconds`); resource baseline.

### 2. Internal gRPC service
- **Examples**: `pricing-engine`, service-mesh backends, internal RPC tiers
- **Traits**: `deployment · exposure: grpc · ingress: false · scaling: auto`
- **Front-loaded**: `appProtocol: grpc`, native `grpc:` probes, no ingress
  (external gRPC is the exception — don't default to it).
- **Handoff**: whether the server registers the gRPC health service (if not:
  instruct adding it — that's the best-practice teaching moment — or fall
  back to `grpc_health_probe`); reflection on/off; message size limits.

### 3. Edge gateway / BFF
- **Examples**: backend-for-frontend, GraphQL federation router, custom API
  gateway
- **Traits**: `deployment · exposure: http · ingress: true · scaling: auto`
- **Front-loaded**: same as №1; its meaning is *compositional* — typically
  the only `ingress: true` subchart, siblings demoted to internal.
- **Handoff**: upstream service names → env/config wiring to sibling
  subcharts' Service DNS; route table; per-upstream timeouts and the ingress
  timeout that must exceed them.
- **Scope note**: only when the gateway is developer code. "Put nginx/Kong
  in front" is a dependency (see Guardrails).

### 4. Web frontend
- **Examples**: SPA served by nginx, SSR app (Next.js), static site
- **Traits**: `deployment · exposure: http · ingress: true · scaling: auto · shared_env: false`
- **Front-loaded**: `shared_env: false` default (frontends rarely need
  platform secrets); README wording on asset-path/base-URL ↔ ingress-path
  coupling.
- **Handoff**: the base-path question is *build-time* — the image bakes in
  the asset prefix, so ingress path changes mean rebuilds, not values edits;
  SSR vs static (SSR flips it closer to №1: env vars, real readiness);
  cache-header strategy.

### 5. Job executor / queue worker
- **Examples**: email sender, image resizer, Sidekiq/Celery-style consumer
- **Traits**: `deployment · exposure: none · scaling: auto`
- **Front-loaded**: the headline `exposure: none` case — no Service, no
  ingress, no ports, no networkPolicy ingress rules, no HTTP probes. The #1
  piece-meal error this tool prevents is copy-pasting a web chart for a
  worker.
- **Handoff**: queue connection env; liveness signal for a process nobody
  contacts (exec heartbeat, staleness file); in-flight-job drain time →
  `terminationGracePeriodSeconds` (the most under-set value in worker
  charts); concurrency per pod (informs the resources block).

### 6. Stream processor
- **Examples**: Kafka consumer group, enrichment stage, CDC consumer
- **Traits**: `deployment · exposure: none · scaling: fixed`
- **Front-loaded**: №5 minus the HPA — replica count is coupled to partition
  count, and autoscaling churns consumer-group rebalances. The scaffold
  encodes the practice ("don't HPA a consumer group") that piece-meal charts
  learn in production.
- **Handoff**: consumer group id and topic(s); partition count → the fixed
  replica value (with the "≤ partitions" comment); offset-commit semantics →
  whether rolling restart is safe or `maxUnavailable` must be tightened.

### 7. Scheduler / singleton controller
- **Examples**: cron dispatcher, outbox relay, leader-elected reconciler
- **Traits**: `deployment · exposure: none · scaling: singleton`
- **Front-loaded**: pinned `static: 1` with "do not raise" comment,
  `strategy: Recreate`, no HPA, no PDB — three correlated decisions
  piece-meal charts get wrong independently.
- **Handoff**: what happens if two ever overlap (idempotent? locked? data
  race?) — determines whether `Recreate` suffices or the app needs leader
  election; and whether the workload is actually №F2 (a CronJob) in
  disguise.

### 8. Realtime gateway
- **Examples**: WebSocket hub, SSE fan-out, push edge
- **Traits**: `deployment · exposure: http · ingress: true · scaling: fixed`
- **Front-loaded**: `fixed` scaling (CPU-HPA scale-downs sever long-lived
  connections); README flags session-affinity annotation as a known manual
  step.
- **Handoff**: connection drain on shutdown (close frames, client retry) →
  preStop + grace period; sticky-session requirement real or not; max
  connections per pod → resources.

### 9. ML inference service
- **Examples**: custom model server, embedding service, LLM wrapper
- **Traits**: `deployment · exposure: http or grpc · ingress: false · scaling: fixed`
- **Front-loaded**: №1/№2 shape plus a `startupProbe` stub — the
  slow-boot-probe interaction is exactly the kind of thing added after the
  first CrashLoopBackOff instead of before.
- **Handoff**: model load time → startupProbe budget; GPU resource requests
  and node selector/toleration; where the model comes from (baked in image /
  init container / volume).
- **Scope note**: only for developer-written servers; "run vLLM/Triton" is a
  dependency.

### 10. Webhook ingest endpoint
- **Examples**: Stripe/GitHub/Twilio webhook receiver
- **Traits**: `deployment · exposure: http · ingress: true · scaling: auto · shared_env: true`
- **Front-loaded**: public ingress even on otherwise-internal platforms;
  thin validate-and-enqueue shape that pairs with №5.
- **Handoff**: signature-verification secret wiring (its own Secret, not
  shared config); ingress body-size limit vs provider payloads; idempotency
  (webhooks redeliver — does the handler dedupe?).

### 11. Admin / back-office dashboard
- **Examples**: internal admin UI, ops console (developer-built)
- **Traits**: `deployment · exposure: http · ingress: true · scaling: fixed · shared_env: true`
- **Front-loaded**: ⚠ the traits can't yet say "route it, but on the private
  ingress class behind auth" — until an ingress-class/auth trait exists, the
  scaffold's README carries an explicit **"protect this route"** warning.
  The dashboard exposed to the internet by default is a canonical
  piece-meal accident; the warning is the mitigation.
- **Handoff**: the auth mechanism (SSO proxy annotations, basic-auth secret,
  app-level) → concrete ingress annotations; internal ingress class name.

### 12. Node agent
- **Examples**: developer-written log shipper, custom node exporter,
  security agent
- **Traits**: `daemonset · exposure: none (or tcp for a scrape port) · scaling: n/a`
- **Front-loaded**: the daemonset representative — scaling question hidden,
  no ingress ever, Service only as a headless scrape target.
- **Handoff**: hostPath mounts and why; tolerations (run on control-plane /
  tainted nodes?); RBAC the agent needs — all values-level, all
  app-determined.

---

## Future-workload patterns

Both are developer code and squarely on-goal — migrations are among the most
skipped-until-late practices in real charts — but they need workload types
outside today's `AllowedWorkloads`. Listed as the motivating cases, not
faked with traits.

### F1. Migrator / bootstrap task ⚠
- **Examples**: schema migration, seed data, index builder
- **Needs**: `workload: job` + Helm hook (`pre-install,pre-upgrade`).
- **Closest today**: №7 that idles after finishing — wrong semantics.
- **Handoff (once real)**: migration command; idempotency/re-run safety;
  hook weight relative to the app's rollout.

### F2. Recurring batch job ⚠
- **Examples**: nightly report, retention sweep, backup
- **Needs**: `workload: cronjob` with `schedule`; would land with F1 (shared
  Job plumbing).
- **Front-loadable then**: `concurrencyPolicy: Forbid` default,
  history limits — the settings piece-meal CronJobs omit.
- **Handoff (once real)**: schedule; expected runtime → `activeDeadlineSeconds`;
  overlap tolerance.

---

## Guardrails for text-to-config

Not patterns — the two decision rules the server prompt needs so pattern
recognition doesn't overreach. These absorb patterns 24–25 of rev 1 and the
entire dropped infrastructure family:

1. **Dependency rule**: "with a Postgres database / Redis / Kafka /
   Elasticsearch / Prometheus…" → an umbrella `dependencies:` entry with a
   values block, **never** a generated subchart. Generated templates are for
   the developer's code; mature upstream charts are the best practice for
   everything else — recommending them *is* the tool teaching best
   practices. Chartpress emits the stanza itself, sourced **only** from a
   curated, version-pinned registry shipped as in-repo data (frozen per
   release, each pin carrying a `TODO(chartpress): review version` marker;
   start tiny — postgres, redis/valkey, kafka, rabbitmq). Infrastructure not
   in the registry degrades to a stub entry with `TODO(chartpress)` markers
   for repository and version. The model decides *which* dependency, never
   *where from*. A chart that declares dependencies needs `helm dependency
   build` (the generated `make deps` target) — and any TODO stub resolved —
   before it renders; that is Helm's behavior, documented in the generated
   Makefile and HANDOFF.
2. **Sidecar rule**: "the api runs with a cloud-sql-proxy sidecar" → a
   container in the owning subchart's pod, **never** a new subchart.
   Possible future home: a values-level `sidecars:` list; explicitly not a
   traits concern.
3. **Closed-set rule**: the model always picks the nearest of the twelve
   pattern ids; `custom` is reserved for humans and never emitted; trait
   overrides only for facts stated in the user's text ("on port 3000",
   "can't run two at once"), never inferred. Misclassification cost is
   bounded by the web review step — a wrong-but-plausible pattern chip is
   one click to correct.

---

## Pattern → traits summary

This table is **normative and frozen**: `pattern: <id>` resolves to exactly
these trait defaults (№9 defaults to `http`; №12 to `none`), and a shipped
pattern's defaults never change — a change requires a new pattern id or a
major version, because specs store intent and are re-resolved on every
generate.

Defaults apply **dependently**, after explicit keys are read: a default can
never cause a validation error, only user-written keys can. So an
`exposure: tcp` override silently resolves the inherited `ingress: true` to
`false`; `port` defaults to 8080 (`http`/`tcp`) or 50051 (`grpc`) and is
absent for `none`; a `daemonset` resolves `scaling` to fixed and never gets
an HPA (that is what №12's "n/a" cell means; explicit `auto`/`singleton` on
a daemonset is an error).

`scaling: auto` means autoscaling is *available*, not active: the scaffold
ships `hpa.yaml` and the `podCount` block static-by-default with the dynamic
(HPA) subtree ready to switch on once resources are sized during handoff.
`fixed` and `singleton` remove the HPA machinery entirely.

| # | Pattern | `pattern:` id | workload | exposure | ingress | scaling | shared_env |
|---|---------|---------------|----------|----------|---------|---------|------------|
| 1 | API microservice | `api-microservice` | deployment | http | true | auto | true |
| 2 | Internal gRPC service | `grpc-service` | deployment | grpc | false | auto | true |
| 3 | Edge gateway / BFF | `edge-gateway` | deployment | http | true | auto | true |
| 4 | Web frontend | `web-frontend` | deployment | http | true | auto | false |
| 5 | Job executor / worker | `worker` | deployment | none | — | auto | true |
| 6 | Stream processor | `stream-processor` | deployment | none | — | fixed | true |
| 7 | Scheduler / singleton | `scheduler` | deployment | none | — | singleton | true |
| 8 | Realtime gateway | `realtime-gateway` | deployment | http | true | fixed | true |
| 9 | ML inference | `ml-inference` | deployment | http/grpc | false | fixed | true |
| 10 | Webhook ingest endpoint | `webhook-ingest` | deployment | http | true | auto | true |
| 11 | Admin dashboard | `admin-dashboard` | deployment | http | true | fixed | true |
| 12 | Node agent | `node-agent` | daemonset | none/tcp | — | n/a | false |
| — | Escape hatch | `custom` | explicit | explicit | explicit | explicit | explicit |
| F1 | Migrator ⚠ | — (future) | job (future) | none | — | one-shot | true |
| F2 | Recurring batch job ⚠ | — (future) | cronjob (future) | none | — | scheduled | true |

Notable: with infrastructure delegated to dependencies, **statefulset
disappears from the pattern list**. Developer-authored statefulsets are rare
enough that the workload can stay supported without a named pattern —
worth revisiting if real usage disagrees.

## The handoff prompt (sketch)

The per-pattern **Handoff** bullets above are the content; the prompt itself
is one generated artifact per subchart (or one per umbrella with per-subchart
sections) that tells an LLM-with-the-codebase:

1. **What this scaffold is** — the pattern name, what was generated and why,
   which practices are already in place and must not be removed (the
   "purpose" half the user asked the template to carry).
2. **What to resolve against the code** — the pattern's handoff checklist,
   phrased as verifiable questions ("find the listen port", "find or add the
   health endpoint", "measure/estimate drain time").
3. **How to finish** — where each answer lands (values key, probe field,
   annotation), and the rule that placeholders are *replaced*, never
   deleted.
4. **What not to do** — the guardrails, plus pattern-specific footguns
   ("don't add an HPA to this consumer group", "don't raise the singleton's
   replica count").

**Resolved (rev 4, supersedes rev 3's one-per-subchart): one `HANDOFF.md`
at the umbrella root**, generated as a sectioned document — a protocol
header (ground rules, validation loop), platform-level items (values
layering, `dependencies:` wiring, validation warnings), then one section
per subchart keyed by its pattern, adjusted by trait overrides. It's still
addressed to a different reader (an LLM session) at a different time (after
code exists); completion is now incremental — delete a subchart's *section*
when that component is finished, delete the *file* when the chart is done.
The generated README links to it while it exists. Emission is gated by a
new umbrella rule `generate_handoff` (default true). Accepted trade-off: a
subchart extracted for standalone use leaves its handoff behind.

## Resolutions (rev 3)

Formerly the open questions, decided when the configurator design was
reworked pattern-first (the companion doc carries the full argument):

- **Patterns are a first-class field.** `pattern: worker` expands to the
  summary table's trait defaults; explicit trait keys override. The rev-2
  worry — that a pattern field reintroduces the rejected `role` enum through
  the back door — is answered by overridability (the pattern sets defaults,
  not constraints) and by what traits cannot carry: patterns with identical
  traits (№1 vs №10, №8 vs №11) have different handoff checklists, so the
  field is not redundant with its own expansion.
- **Chartpress emits the `dependencies:` stanza** (plus a values-block
  skeleton and a README note naming the chosen upstream chart) when the
  dependency rule fires — not a README recommendation alone. On-thesis: the
  best practice is done, not suggested.
- **Presets** are rewritten pattern-first; `event-driven-platform` =
  `edge-gateway` + `api-microservice` + `worker` + `scheduler` with Kafka as
  a dependency is the showcase of the patterns and the dependency rule
  together.
- **The web UI leads with patterns** — a picker of the twelve cards plus
  Custom; the trait questionnaire survives only as the Custom path and each
  pattern's "Advanced" override panel. The engine *does* see the pattern
  name: it selects the handoff checklist and the per-pattern extras (№9's
  startupProbe stub, №11's README warning, №8's session-affinity note,
  №6's partition comment).

## Resolutions (rev 4 — design review)

Decided in the 2026-07-02 review; the companion doc carries each argument in
full:

- **Specs store intent** (pattern + explicit overrides), re-resolved on
  every generate; the summary table is frozen per release as an API
  contract.
- **Dependent defaulting** — defaults resolve after explicit keys and adapt
  to them; only user-written keys can fail validation. Fixes the
  `custom + exposure: none` trap and defines the daemonset scaling cell.
- **A warnings tier** joins `Validate` (non-blocking; web/CLI/CR status +
  handoff). First warnings: №3 coexisting with other `ingress: true`
  subcharts, №11 reachable via ingress. Cross-subchart auto-demotion was
  rejected — resolution stays per-subchart-pure.
- **Single sectioned root `HANDOFF.md`** (see above), replacing rev 3's
  one-per-subchart.
- **Curated dependency registry** with stub fallback (guardrail 1);
  **closed-set classification** (guardrail 3).
- **Pattern registry ships as embedded JSON** with a verbatim web copy and a
  CI contract test (the web Docker build context forbids a shared file).
- **Handoff emission merged into phase 1** — patterns whose traits are
  identical (№1/№10, №8/№11) are distinguishable only by their handoff, so
  differentiation and handoff ship together.
- **Deferred**: a metrics/scrape-port trait (№12's headless-Service nuance
  included) stays a handoff item in v1; no spec-level version field — the
  freeze rule carries versioning until a real major.
