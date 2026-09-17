# Chartpress Open-Source and Reproducibility Readiness Plan

**Status:** Proposed  
**License decision:** Apache License 2.0  
**Scope:** Licensing, provenance, reproducible builds, hermetic verification, regression coverage, and repository hygiene

## 1. Objective

Establish a clean, public, permissively licensed baseline of Chartpress that can be built, inspected, modified, and verified from a fresh clone without relying on mutable local state or external services during test execution.

The finished repository should provide:

- unambiguous permission to use, modify, and redistribute the project;
- a documented provenance story for source, templates, documentation, images, and checked-in archives;
- pinned and reproducible Go and JavaScript dependency installation;
- one canonical verification command used locally and in CI;
- a containerized verification environment that runs without network access after it is built;
- stable regression tests for the engine, CLI-facing behavior, HTTP API, operator, Helm chart, and web application;
- a clean checkout before and after verification; and
- clear contributor, security, development, and release documentation.

## 2. License decision

Adopt **Apache License 2.0** for the repository.

Apache-2.0 remains permissive while providing stronger and more explicit obligations than the shorter permissive alternatives. In particular, it includes an express patent license, patent-termination provisions, preservation of notices, a requirement to identify modified files, and no grant of trademark rights.

### Required changes

- [ ] Replace the current root `LICENSE` with the complete, unmodified Apache License 2.0 text.
- [ ] Add a root `NOTICE` containing only accurate project copyright and attribution notices.
- [ ] Confirm the copyright holder and year range used in `NOTICE`.
- [ ] Add `SPDX-License-Identifier: Apache-2.0` to project-owned source, script, and template files where the file format supports comments without changing behavior.
- [ ] Add `"license": "Apache-2.0"` to `web/package.json` and refresh `web/package-lock.json` with the existing npm version.
- [ ] Add `org.opencontainers.image.licenses=Apache-2.0` and source-revision labels to the production images.
- [ ] Replace proprietary and source-available language in:
  - `README.adoc`
  - `CONTRIBUTING.adoc`
  - `crds/README.md`
- [ ] Keep contribution policy separate from licensing. The project may decline outside contributions, but its documentation must not prohibit uses, forks, modification, or redistribution granted by Apache-2.0.
- [ ] Add the licensing change to `CHANGELOG.adoc`.
- [ ] After the change reaches the default branch, verify that GitHub identifies the repository license as Apache-2.0.

## 3. Provenance and third-party material

Relicensing is valid only for material the project owns or has permission to distribute under compatible terms.

- [ ] Review all human-authored commit identities and confirm that the project has the right to relicense their contributions.
- [ ] Inventory non-source assets, including:
  - `docs/*.png`
  - `docs/favicon/*`
  - `web/src/assets/*`
  - `tests/chart.zip`
  - templates and documentation adapted from other projects
- [ ] Record the origin and license of any third-party material in `NOTICE` or a dedicated `THIRD_PARTY_NOTICES.md` when required.
- [ ] Remove or replace material whose origin or license cannot be established.
- [ ] Add an automated dependency-license report for Go modules and npm packages.
- [ ] Fail CI if a newly introduced dependency has an unapproved or unknown license.
- [ ] Keep downloaded dependencies and generated `node_modules` out of version control.

## 4. Repository hygiene

### Immediate cleanup

- [ ] Remove `presets/chart99.zip`; it is an HTML nginx error response, not a ZIP archive.
- [ ] Decide whether `tests/chart.zip` is a durable golden fixture or an accidental generated artifact.
  - If it is a fixture, document its purpose and add a deterministic regeneration command.
  - If it is not required, remove it.
- [ ] Verify that every checked-in archive can be listed and extracted successfully.
- [ ] Add `.gitattributes` entries for binary images and archives.
- [ ] Remove stale generated output and confirm `.gitignore` covers all local build products.
- [ ] Add a `make check-clean` target that fails when verification modifies tracked files or leaves unexpected untracked files.

### Documentation consistency

- [ ] Correct `README.adoc` so the documented test command matches the commands used by CI.
- [ ] Distinguish unit/regression tests from the live-server curl smoke test currently exposed by `make tests`.
- [ ] Document supported versions of Go, Node.js, npm, and Helm.
- [ ] Document which tests use local loopback sockets and confirm that they never contact public endpoints.
- [ ] Add a short architecture map showing how the engine, server, operator, CRDs, chart, templates, and web client fit together.

## 5. Canonical developer commands

Create stable, non-interactive commands with consistent behavior locally and in CI.

Suggested targets:

```make
make test-go       # Go unit and regression tests
make vet           # go vet
make test-web      # frontend unit/component tests
make build-web     # production frontend build
make lint-chart    # Helm lint with deterministic test values
make smoke         # optional live-server HTTP smoke test
make verify        # all required static, test, chart, and build checks
make verify-offline
make check-clean
```

Implementation requirements:

- [ ] Make `make verify` the single required local and CI quality gate.
- [ ] Keep the live-server curl test under `make smoke`; do not make it the only or primary test target.
- [ ] Ensure targets do not require interactive input.
- [ ] Ensure test output identifies the package or subsystem that failed.
- [ ] Avoid hidden dependency installation inside test targets.
- [ ] Run dependency installation in an explicit setup or image-build phase.
- [ ] Run `make check-clean` after `make verify` in CI.

## 6. Reproducible toolchain and dependency setup

### Go

- [ ] Pin the build image to the Go version declared by `go.mod` (`1.23.6`) instead of the mutable `golang:1.23` tag.
- [ ] Pin the image by immutable digest for release and verification workflows.
- [ ] Run `go mod download` during image construction.
- [ ] Set `GOTOOLCHAIN=local` during offline verification so Go cannot attempt to download another toolchain.
- [ ] Keep `GOMODCACHE` and `GOCACHE` outside the source checkout.
- [ ] Add `go mod verify` to the setup or verification workflow.

### JavaScript

- [ ] Declare and pin the supported Node.js and npm versions.
- [ ] Continue using `npm ci` with `web/package-lock.json`.
- [ ] Install npm packages during image construction, not during test execution.
- [ ] Keep `node_modules`, Vite output, browser caches, and npm caches outside the tracked source tree where practical.
- [ ] Disable audit, update-notifier, and telemetry network calls in hermetic verification.

### Helm and system tools

- [ ] Pin Helm to a tested version and verify its downloaded checksum during image construction.
- [ ] Pin any additional command-line tools in the same way.
- [ ] Avoid requiring Docker, Kubernetes, S3, an OCI registry, GitHub, or an AI service while running the test suite.

## 7. Hermetic verification image

Add a dedicated development/verification image separate from the production runtime image.

The image should:

- contain Go 1.23.6, the pinned Node.js/npm toolchain, Helm, Git, and required shell utilities;
- pre-download Go modules and npm packages during the image build;
- contain all tools needed by `make verify`;
- use caches outside the repository checkout;
- run as an unprivileged user where possible;
- require no mounted Docker socket;
- require no credentials or secrets; and
- successfully run the required checks with container networking disabled.

Add a command such as:

```sh
docker build -f Dockerfile.verify -t chartpress-verify .
docker run --rm --network=none chartpress-verify make verify
```

Acceptance checks:

- [ ] The image builds from a fresh clone.
- [ ] A second image build reuses pinned dependency layers.
- [ ] `make verify` succeeds under `--network=none`.
- [ ] Loopback-only `httptest` servers still work with external networking disabled.
- [ ] No test tries to resolve a public hostname.
- [ ] Verification does not change the checkout.

## 8. Regression-test coverage

### Existing Go surface

Preserve the current broad Go suite covering:

- configuration parsing and normalization;
- compose conversion;
- CRD/schema consistency;
- Helm rendering and deployment manifests;
- chart-generation rules and artifacts;
- object-store configuration and local signing behavior;
- operator reconciliation with fake clients; and
- HTTP handlers and AI-client behavior through fakes or local `httptest` servers.

Enhancements:

- [ ] Add table-driven negative cases for validation and malformed input.
- [ ] Add deterministic archive-content assertions rather than comparing timestamps or temporary paths.
- [ ] Add cross-surface contract tests so Go types, CRDs, presets, and frontend option definitions cannot drift.
- [ ] Add regression coverage for CLI exit codes, stdout/stderr separation, and generated output paths.
- [ ] Confirm all external-client code is injectable and testable through interfaces or local fakes.
- [ ] Run the complete Go suite repeatedly in CI to identify shared-state or ordering flakes.

### Frontend surface

The web package currently has a production build but no first-party test command. Add a pinned frontend test harness.

- [ ] Add Vitest and Testing Library for deterministic unit and component tests.
- [ ] Add tests for `spec.js`, `api.js`, and `localStore.js` before screen-level tests.
- [ ] Add component tests for validation errors, loading states, API failures, authentication state, and keyboard/accessibility behavior.
- [ ] Add a small Playwright smoke suite only where a real browser materially improves coverage.
- [ ] Install browser binaries and system dependencies during image construction.
- [ ] Run browser tests offline against local fixtures or an in-process fake API.
- [ ] Do not use screenshot appearance as a correctness requirement; assert DOM, accessibility, state, navigation, and measurable behavior.

### Helm and generated artifacts

- [ ] Keep `helm lint` in the required gate with explicit deterministic values.
- [ ] Run `templates/umbrella/tests/validate-templates.sh` from the canonical verification target if it remains supported.
- [ ] Add schema and rendering checks for both canonical and chart-shipped CRD copies.
- [ ] Verify generated archives contain only expected relative paths and deterministic file modes.
- [ ] Ensure generated charts can be inspected without downloading chart dependencies.

## 9. CI consolidation

- [ ] Replace duplicated workflow command sequences with `make verify` or a reusable workflow.
- [ ] Keep publishing jobs dependent on the complete verification gate.
- [ ] Pin third-party GitHub Actions to full commit SHAs.
- [ ] Use least-privilege workflow permissions.
- [ ] Separate pull-request verification from image publication.
- [ ] Add a no-network verification job using the dedicated image.
- [ ] Add dependency-license and secret-scanning jobs.
- [ ] Add a clean-tree assertion after generated-artifact tests.
- [ ] Cache dependencies by lockfile hash without making correctness depend on the cache.

## 10. Security and configuration boundaries

- [ ] Add a root `SECURITY.md` with supported versions and a private reporting channel.
- [ ] Verify that test fixtures contain no usable credentials, tokens, endpoints, or personal data.
- [ ] Keep all OpenAI, S3, GitHub, and Kubernetes clients behind replaceable interfaces.
- [ ] Reject unsafe file paths and archive traversal at input boundaries.
- [ ] Keep generated files inside an explicitly configured output directory.
- [ ] Test owner isolation and authorization rules without requiring a live identity provider.
- [ ] Document that production integrations require credentials, while verification uses fakes and local fixtures.

## 11. Sequencing

### Phase A — Legal and provenance baseline

1. Audit ownership and third-party material.
2. Adopt Apache-2.0 and add accurate notices.
3. Update all contradictory documentation and metadata.
4. Remove or quarantine material with unclear provenance.

**Exit criterion:** A fresh visitor can determine the license and provenance of every distributed component without encountering contradictory terms.

### Phase B — Clean and reproducible baseline

1. Remove the invalid preset archive.
2. Define deterministic handling for generated fixtures.
3. Pin toolchains and add canonical Make targets.
4. Consolidate CI around `make verify`.

**Exit criterion:** `make verify && make check-clean` passes twice from a fresh clone.

### Phase C — Hermetic execution

1. Add `Dockerfile.verify`.
2. Pre-install every dependency and tool during image construction.
3. Run the full required suite with external networking disabled.
4. Remove or mock any unexpected network dependency.

**Exit criterion:** The verification image passes with `--network=none`, no credentials, and no mounted services.

### Phase D — Coverage completion

1. Add the frontend unit/component harness.
2. Add cross-surface schema and contract tests.
3. Add deterministic CLI and archive tests.
4. Exercise the suite repeatedly to eliminate flakes.

**Exit criterion:** Every maintained subsystem has objective regression coverage, and repeated verification runs are stable.

## 12. Definition of done

The readiness work is complete when all of the following are true:

- [ ] The default branch contains the standard Apache-2.0 license and accurate notices.
- [ ] No repository documentation contradicts the granted license.
- [ ] Third-party and generated material has documented provenance.
- [ ] The invalid `presets/chart99.zip` artifact is gone.
- [ ] Go, Node.js, npm, Helm, modules, and packages are pinned or lockfile-controlled.
- [ ] `make verify` is the canonical local and CI gate.
- [ ] `make verify` succeeds inside the verification image with external networking disabled.
- [ ] Verification leaves the checkout clean.
- [ ] Go, frontend, Helm, CRD, CLI, and artifact behavior have stable regression coverage.
- [ ] Tests require no real cluster, object store, registry, identity provider, or AI endpoint.
- [ ] Production image publication is gated on the same required verification suite.
- [ ] A full 40-character commit SHA can identify the resulting clean baseline.

