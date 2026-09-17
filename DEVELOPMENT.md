<!-- SPDX-License-Identifier: Apache-2.0 -->
# Development and verification

## Supported toolchain

- Go 1.23.6 (the `go.mod` toolchain)
- Node.js 20.18.3 and npm 10.8.2
- Helm 3.17.3
- kubeconform 0.7.0 for generated-template validation

Install dependencies explicitly with `go mod download all` (the `all` form, not the bare `go mod download`, is required so `make license-check` can inspect every module in the graph offline) and `npm ci --no-audit --no-fund --ignore-scripts --prefix web`. Tests never install dependencies themselves.

`make verify` is the canonical quality gate. It runs Go tests and vet, frontend tests and production build, Helm lint, generated-template validation, archive validation, dependency-license policy, and `go mod verify`. `make smoke` is a separate, optional curl test against a running server. `make check-clean` asserts that verification did not alter the checkout.

Tests may bind local loopback sockets via Go's `httptest`; they do not contact public endpoints. Production client integrations are replaced with fakes in tests.

For a hermetic run:

```sh
docker build -f Dockerfile.verify -t chartpress-verify .
docker run --rm --network=none chartpress-verify make verify-offline
```

The image downloads modules and npm packages only while it is built. Its caches live outside `/workspace`, and verification runs as an unprivileged user.

## Architecture map

```text
CLI ───────────────┐
HTTP API ──────────┼─> engine ─> templates ─> generated Helm archive
operator + CRDs ───┘      │                         │
                           └─ contract tests <────── chart + CRD copies
web client ─────────────> HTTP API
Helm deployment chart ──> server + operator + web client
```

## Releases

Run `make verify && make check-clean` from a fresh clone, update `CHANGELOG.adoc`, and tag the exact clean commit. Production-image workflows are gated by the same `make verify` command. Record the full 40-character commit SHA in release notes.
